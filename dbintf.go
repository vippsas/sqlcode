package sqlcode

import (
	"context"
	"database/sql"
)

type DB interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
	Conn(ctx context.Context) (*sql.Conn, error)
	BeginTx(ctx context.Context, txOptions *sql.TxOptions) (*sql.Tx, error)
}

var _ DB = &sql.DB{}
