package sqltest

import (
	"context"
	"database/sql"
)

type CtxExecer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

type CtxQuerier interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}
