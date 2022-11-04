package example

import (
	"embed"
	"github.com/vippsas/sqlcode"
)

//go:embed *.sql
//go:embed */*.sql
var sqlfs embed.FS

var SQL = sqlcode.MustInclude(sqlfs, "example")
