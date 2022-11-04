package sqlcode

import (
	"context"
	"database/sql"
	"fmt"
	mssql "github.com/denisenkom/go-mssqldb"
	"github.com/pkg/errors"
	"github.com/vippsas/sqlcode/sqlparser"
	"io/fs"
	"strconv"
	"strings"
)

type Deployable struct {
	SchemaSuffix string
	ParsedFiles  []string // mainly for use in error messages etc
	CodeBase     sqlparser.Document

	// cache over whether it has been uploaded to a given DB
	// (the same physical DB can be in this map multiple times under
	// different interfaces; that's fine; in general the same interface
	// seems to be acquired)
	uploaded map[DB]struct{}
}

func (d Deployable) WithSchemaSuffix(schemaSuffix string) Deployable {
	return Deployable{
		SchemaSuffix: schemaSuffix,
		CodeBase:     d.CodeBase,
		uploaded:     make(map[DB]struct{}),
	}
}

// impersonate manages impersonating another user (presumably one with fewer privleges)
// for an operation
func impersonate(ctx context.Context, dbc DB, username string, f func(conn *sql.Conn) error) error {
	if strings.Contains(username, "\"") {
		panic("assertion failed")
	}

	conn, err := dbc.Conn(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = conn.Close()
	}()

	// a cookie is used to be able to revert the connection back to original
	// privileges
	var executeAsCookie []byte

	// Note: we don't want to time out when messing with privileges, so
	// use context.Background here
	err = conn.QueryRowContext(context.Background(), fmt.Sprintf(`
		declare @cookie varbinary(8000);
		execute as user = '%s' with cookie into @cookie;
		select @cookie
`, username)).Scan(&executeAsCookie)
	if err != nil {
		return err
	}

	// OK we have dropped privileges, now make sure that no matter what
	// happens, we revert to original privileges before returning the connection
	// to the pool. We don't want to time out on this operation so use context.Background
	defer func() {
		_, _ = conn.ExecContext(context.Background(), `revert with cookie = @cookie`, sql.Named("cookie", executeAsCookie))
	}()

	return f(conn)
}

// Upload will create and upload the schema; resulting in an error
// if the schema already exists
func (d *Deployable) Upload(ctx context.Context, dbc DB) error {
	// First, impersonate a user with minimal privileges to get at least
	// some level of sandboxing so that migration scripts can't do anything
	// the caller didn't expect them to.
	return impersonate(ctx, dbc, "sqlcode-deploy-sandbox-user", func(conn *sql.Conn) error {
		tx, err := conn.BeginTx(ctx, nil)
		if err != nil {
			return err
		}

		_, err = tx.ExecContext(ctx, `sqlcode.CreateCodeSchema`,
			sql.Named("schemasuffix", d.SchemaSuffix),
		)
		if err != nil {
			_ = tx.Rollback()
			return err
		}

		preprocessed, err := Preprocess(d.CodeBase, d.SchemaSuffix)
		if err != nil {
			_ = tx.Rollback()
			return err
		}
		for _, b := range preprocessed.Batches {
			_, err := tx.ExecContext(ctx, b.Lines)
			if err != nil {
				_ = tx.Rollback()
				sqlerr, ok := err.(mssql.Error)
				if !ok {
					return err
				} else {
					return SQLUserError{
						Wrapped: sqlerr,
						Batch:   b,
					}
				}
			}
		}
		err = tx.Commit()
		if err != nil {
			return err
		}

		d.markAsUploaded(dbc)

		return nil

	})

}

// EnsureUploaded checks that the schema with the suffix already exists,
// and if not, creates and uploads it. This is suitable for hash-based
// schema suffixes.  A lock will be
// taken (globally in SQL) during the process so that multiple concurrent calls
// from services starting at the same time line up nicely
func (d *Deployable) EnsureUploaded(ctx context.Context, dbc DB) error {
	if d.IsUploadedFromCache(dbc) {
		return nil
	}

	lockResourceName := "sqlcode.EnsureUploaded/" + d.SchemaSuffix

	var lockRetCode int
	err := dbc.QueryRowContext(ctx, `
declare @retcode int;
exec @retcode = sp_getapplock @Resource = @resource, @LockMode = 'Shared', @LockOwner = 'Session', @LockTimeout = @timeoutMs;
select @retcode;
`,
		sql.Named("resource", lockResourceName),
		sql.Named("timeoutMs", 20000),
	).Scan(&lockRetCode)
	if err != nil {
		return err
	}
	if lockRetCode < 0 {
		return errors.New("was not able to get lock before timeout")
	}

	defer func() {
		_, _ = dbc.ExecContext(ctx, `sp_releaseapplock`,
			sql.Named("Resource", lockResourceName),
			sql.Named("LockOwner", "Session"),
		)
	}()

	exists, err := Exists(ctx, dbc, d.SchemaSuffix)
	if err != nil {
		return err
	}

	if exists {
		return nil
	}

	return d.Upload(ctx, dbc)
}

// UploadWithOverwrite will always drop the schema if it exists, before
// uploading. This is suitable for named schema suffixes.
func (d Deployable) DropAndUpload(ctx context.Context, dbc DB) error {
	exists, err := Exists(ctx, dbc, d.SchemaSuffix)
	if err != nil {
		return err
	}

	if exists {
		err = Drop(ctx, dbc, d.SchemaSuffix)
		if err != nil {
			return err
		}
	}

	return d.Upload(ctx, dbc)
}

// Patch will preprocess the sql passed in so that it will call SQL code
// deployed by the receiver Deployable
func (d Deployable) Patch(sql string) string {
	return preprocessString(d.SchemaSuffix, sql)
}

func (d *Deployable) markAsUploaded(dbc DB) {
	d.uploaded[dbc] = struct{}{}
}

func (d *Deployable) IsUploadedFromCache(dbc DB) bool {
	_, found := d.uploaded[dbc]
	return found
}

// TODO: StringConst. This requires parsing a SQL literal, a bit too complex
// to code up just-in-case

func (d Deployable) IntConst(s string) (int, error) {
	for _, declare := range d.CodeBase.Declares {
		if declare.VariableName == s {
			// TODO: more robust integer SQL parsing than this; only works
			// in most common cases
			return strconv.Atoi(declare.Literal.RawValue)
		}
	}
	return 0, errors.New("No `declare `" + s + "` found")
}

func (d Deployable) MustIntConst(s string) int {
	result, err := d.IntConst(s)
	if err != nil {
		panic(err)
	}
	return result
}

// Options that affect file parsing etc; pass an empty struct to get
// default options.
type Options struct {
	IncludeTags []string

	// if this is set, parsing or ordering failed and it's up to the caller
	// to know what one is doing..
	PartialParseResults bool
}

// Include is used to package SQL code included using the `embed`
// go feature; constructing a Deployable of SQL code. Currently, only a
// single packageName is supported, but we can make string e.g. `...string` later.
func Include(opts Options, fsys ...fs.FS) (result Deployable, err error) {

	parsedFiles, doc, err := sqlparser.ParseFilesystems(fsys, opts.IncludeTags)
	if len(doc.Errors) > 0 && !opts.PartialParseResults {
		return Deployable{}, SQLCodeParseErrors{Errors: doc.Errors}
	}

	result.CodeBase = doc
	result.ParsedFiles = parsedFiles
	result.SchemaSuffix = SchemaSuffixFromHash(result.CodeBase)
	result.uploaded = make(map[DB]struct{})
	return
}

func MustInclude(opts Options, fsys ...fs.FS) Deployable {
	result, err := Include(opts, fsys...)
	if err != nil {
		panic(err)
	}
	return result
}
