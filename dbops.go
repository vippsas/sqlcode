package sqlcode

import (
	"context"
	"database/sql"
)

func Exists(ctx context.Context, dbc DB, schemasuffix string) (bool, error) {
	var schemaID int
	err := dbc.QueryRowContext(ctx, `select isnull(schema_id(@p1), 0)`, SchemaName(schemasuffix)).Scan(&schemaID)
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
