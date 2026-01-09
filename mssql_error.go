package sqlcode

import (
	"bytes"
	"fmt"
	"strings"

	mssql "github.com/microsoft/go-mssqldb"
	"github.com/simukka/sqlcode/sqlparser/sqldocument"
)

type MSSQLUserError struct {
	Wrapped mssql.Error
	Batch   Batch
}

func (s MSSQLUserError) Error() string {
	var buf bytes.Buffer

	if _, fmterr := fmt.Fprintf(&buf, "\n"); fmterr != nil {
		panic(fmterr)
	}
	for _, item := range s.Wrapped.All {
		if _, fmterr := fmt.Fprintf(&buf, "\n%s:%d (%s): %s",
			s.Batch.StartPos.File,
			s.Batch.LineNumberInInput(int(item.LineNo)),
			item.ProcName,
			item.Message); fmterr != nil {
			panic(fmterr)
		}
	}
	return buf.String()
}

type SQLCodeParseErrors struct {
	Errors []sqldocument.Error
}

func (e SQLCodeParseErrors) Error() string {
	var msg strings.Builder
	msg.WriteString("sqlcode syntax error:\n\n")
	for _, e := range e.Errors {
		msg.WriteString(fmt.Sprintf("%s:%d:%d: %s\n", e.Pos.File, e.Pos.Line, e.Pos.Col, e.Message))
	}
	return msg.String()
}
