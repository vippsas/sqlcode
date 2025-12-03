package sqlcode

import (
	"context"
	"database/sql"

	mssql "github.com/denisenkom/go-mssqldb"
	"github.com/jackc/pgx/v5/stdlib"
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
	_, err = tx.ExecContext(ctx, `sqlcode.DropCodeSchema`,
		sql.Named("schemasuffix", schemasuffix))
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
