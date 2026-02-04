package pgsql

import (
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/vippsas/sqlcode/v2/sqlparser/sqldocument"
)

var PGSqlStatementTokens = []string{"create"}

type PGSqlDocument struct {
	creates []sqldocument.Create
	errors  []sqldocument.Error

	sqldocument.Pragma
}

var _ sqldocument.Document = (*PGSqlDocument)(nil)

func (d *PGSqlDocument) Parse(input []byte, file sqldocument.FileRef) error {
	s := NewScanner(file, string(input))
	s.NextNonWhitespaceToken()

	err := d.ParsePragmas(s)
	if err != nil {
		d.addError(s, err.Error())
	}

	batch := sqldocument.NewBatch()
	batch.StatementTokens = PGSqlStatementTokens
	batch.ReservedTokenHandlers = map[string]func(sqldocument.Scanner, *sqldocument.Batch) int{
		"create": func(s sqldocument.Scanner, b *sqldocument.Batch) int {
			res := &sqldocument.Create{Driver: &stdlib.Driver{}}
			err := b.ParseCreate(s, res)
			if err != nil {
				d.addError(s, err.Error())
			}

			res.Body = append(res.Body, res.Body...)
			res.Docstring = b.DocString
			d.creates = append(d.creates, *res)

			// continue parsing
			return 0
		},
	}

	hasMore := batch.Parse(s)
	if batch.HasErrors() {
		d.errors = append(d.errors, batch.Errors...)
	}
	if hasMore {
		panic("not yet supported")
	}

	return nil
}

func (d *PGSqlDocument) addError(s sqldocument.Scanner, message string) {
	d.errors = append(d.errors, sqldocument.Error{
		Pos:     s.Start(),
		Message: message,
	})
}

func (d PGSqlDocument) Empty() bool {
	return len(d.creates) == 0
}

func (d PGSqlDocument) HasErrors() bool {
	return len(d.errors) > 0
}

func (d PGSqlDocument) Creates() []sqldocument.Create {
	return d.creates
}

func (d PGSqlDocument) Declares() []sqldocument.Declare {
	return nil
}

func (d PGSqlDocument) Errors() []sqldocument.Error {
	return d.errors
}

func (d *PGSqlDocument) Include(doc sqldocument.Document) {
	panic("not yet implemented")
}

func (d *PGSqlDocument) Sort() {
	panic("not yet implemented")
}
