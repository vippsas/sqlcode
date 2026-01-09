package sqltest

import (
	"embed"

	"github.com/simukka/sqlcode/v2"
)

//go:embed *.sql
var sqlfs embed.FS

var SQL = sqlcode.MustInclude(
	sqlcode.Options{},
	sqlfs,
)
