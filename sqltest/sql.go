package sqltest

import (
	"embed"

	"github.com/vippsas/sqlcode"
)

//go:embed *.sql
var sqlfs embed.FS

//go:embed *.pgsql
var pgsqlfx embed.FS

var SQL = sqlcode.MustInclude(
	sqlcode.Options{},
	sqlfs,
	pgsqlfx,
)
