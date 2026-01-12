package pgsql

import "github.com/vippsas/sqlcode/v2/sqlparser/sqldocument"

type PGSqlDocument struct {
	creates []sqldocument.Create
	errors  []sqldocument.Error

	sqldocument.Pragma
}

func (d *PGSqlDocument) Parse(input []byte, file sqldocument.FileRef) error {
	return nil
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
