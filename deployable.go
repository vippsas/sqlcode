package sqlcode

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"strconv"
	"strings"
	"time"

	mssql "github.com/denisenkom/go-mssqldb"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	pgxstdlib "github.com/jackc/pgx/v5/stdlib"
	"github.com/vippsas/sqlcode/sqlparser"
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
	driver := dbc.Driver()
	qs := make(map[string][]interface{}, 1)

	var uploadFunc = func(conn *sql.Conn) error {
		tx, err := conn.BeginTx(ctx, nil)
		if err != nil {
			return err
		}

		for q, args := range qs {
			_, err = tx.ExecContext(ctx, q, args...)

			if err != nil {
				_ = tx.Rollback()
				return fmt.Errorf("failed to execute (%s) with arg(%s) in schema %s: %w", q, args, d.SchemaSuffix, err)
			}
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

	}

	if _, ok := driver.(*mssql.Driver); ok {
		// First, impersonate a user with minimal privileges to get at least
		// some level of sandboxing so that migration scripts can't do anything
		// the caller didn't expect them to.
		qs["sqlcode.CreateCodeSchema"] = []interface {
		}{
			sql.Named("schemasuffix", d.SchemaSuffix),
		}

		return impersonate(ctx, dbc, "sqlcode-deploy-sandbox-user", uploadFunc)
	}

	if _, ok := driver.(*stdlib.Driver); ok {
		qs[`set role "sqlcode-deploy-sandbox-user"`] = nil
		qs[`call sqlcode.createcodeschema(@schemasuffix)`] = []interface{}{
			pgx.NamedArgs{"schemasuffix": d.SchemaSuffix},
		}
		conn, err := dbc.Conn(ctx)
		if err != nil {
			return err
		}
		defer func() {
			_ = conn.Close()
		}()
		return uploadFunc(conn)
	}

	return fmt.Errorf("failed to determine sql driver to upload schema: %s", d.SchemaSuffix)
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

	driver := dbc.Driver()
	lockResourceName := "sqlcode.EnsureUploaded/" + d.SchemaSuffix

	var lockRetCode int
	var lockQs string
	var unlockQs string
	var err error

	// When a lock is opened with the Transaction lock owner,
	// that lock is released when the transaction is committed or rolled back.
	if _, ok := driver.(*pgxstdlib.Driver); ok {
		lockQs = `select sqlcode.get_applock(@resource, @timeout)`
		unlockQs = `select sqlcode.release_applock(@resource)`

		err = dbc.QueryRowContext(ctx, lockQs, pgx.NamedArgs{
			"resource":  lockResourceName,
			"timeoutMs": 20000,
		}).Scan(&lockRetCode)

		defer func() {
			dbc.ExecContext(ctx, unlockQs, pgx.NamedArgs{"resource": lockResourceName})
		}()
	}

	if _, ok := driver.(*mssql.Driver); ok {
		// TODO

		defer func() {
			// TODO: This returns an error if the lock is already released
			_, _ = dbc.ExecContext(ctx, unlockQs,
				sql.Named("Resource", lockResourceName),
				sql.Named("LockOwner", "Session"),
			)
		}()
	}

	if err != nil {
		return err
	}
	if lockRetCode < 0 {
		return errors.New("was not able to get lock before timeout")
	}
	exists, err := Exists(ctx, dbc, d.SchemaSuffix)
	if err != nil {
		return fmt.Errorf("unable to determine if schema %s exists: %w", d.SchemaSuffix, err)
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
	return 0, fmt.Errorf("no `declare %s found`", s)
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

type SchemaObject struct {
	Name       string
	SchemaId   int
	Objects    int
	CreateDate time.Time
	ModifyDate time.Time
}

func (s *SchemaObject) Suffix() string {
	return strings.Split(s.Name, "@")[1]
}

// Return a list of sqlcode schemas that have been uploaded to the database.
// This includes all current and unused schemas.
func (d *Deployable) ListUploaded(ctx context.Context, dbc DB) []*SchemaObject {
	objects := []*SchemaObject{}
	impersonate(ctx, dbc, "sqlcode-deploy-sandbox-user", func(conn *sql.Conn) error {
		rows, err := conn.QueryContext(ctx, `
		select 
			s.name
			, s.schema_id
			, o.objects
			, o.create_date
			, o.modify_date 
		from sys.schemas s
		outer apply (
			select count(o.object_id) as objects
				, min(o.create_date) as create_date
				, max(o.modify_date) as modify_date
			from sys.objects o
			where o.schema_id = s.schema_id
		) as o
		where s.name like 'code@%'`)
		if err != nil {
			return err
		}

		for rows.Next() {
			zero := &SchemaObject{}
			rows.Scan(&zero.Name, &zero.Objects, &zero.SchemaId, &zero.CreateDate, &zero.ModifyDate)
			objects = append(objects, zero)
		}

		return nil
	})
	return objects
}
