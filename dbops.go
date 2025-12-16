package sqlcode

import (
	"context"
	"database/sql"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	mssql "github.com/microsoft/go-mssqldb"
)

func Exists(ctx context.Context, dbc DB, schemasuffix string) (bool, error) {
	var schemaID int

	driver := dbc.Driver()
	var qs string

	if _, ok := driver.(*mssql.Driver); ok {
		qs = `select isnull(schema_id(@p1), 0)`
	}
	if _, ok := driver.(*stdlib.Driver); ok {
		qs = `select coalesce((select oid from pg_namespace where nspname = $1),0)`
	}

	err := dbc.QueryRowContext(ctx, qs, SchemaName(schemasuffix)).Scan(&schemaID)
	if err != nil {
		return false, err
	}
	return schemaID != 0, nil
}

func Drop(ctx context.Context, dbc DB, schemasuffix string) error {
	tx, err := dbc.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	var qs string
	var arg = []interface{}{}
	driver := dbc.Driver()

	if _, ok := driver.(*mssql.Driver); ok {
		qs = `sqlcode.DropCodeSchema`
		arg = []interface{}{sql.Named("schemasuffix", schemasuffix)}
	}

	if _, ok := dbc.Driver().(*stdlib.Driver); ok {
		qs = `call sqlcode.dropcodeschema(@schemasuffix)`
		arg = []interface{}{
			pgx.NamedArgs{"schemasuffix": schemasuffix},
		}
	}

	_, err = tx.ExecContext(ctx, qs, arg...)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return err
}
