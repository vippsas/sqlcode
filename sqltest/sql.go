package sqltest

import (
	"embed"

	"github.com/vippsas/sqlcode"
)

//go:embed *.sql
var sqlfs embed.FS

var SQL = sqlcode.MustInclude(
	sqlcode.Options{},
	sqlfs,
)
