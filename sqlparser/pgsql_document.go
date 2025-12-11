package sqlparser

type PGSqlDocument struct {
	pragmaIncludeIf []string
	creates         []Create
	errors          []Error
}

func (d PGSqlDocument) HasErrors() bool {
	return len(d.errors) > 0
}

func (d PGSqlDocument) Creates() []Create {
	return d.creates
}

func (d PGSqlDocument) Declares() []Declare {
	return nil
}

func (d PGSqlDocument) Errors() []Error {
	return d.errors
}
func (d PGSqlDocument) PragmaIncludeIf() []string {
	return d.pragmaIncludeIf
}

func (d PGSqlDocument) Empty() bool {
	return len(d.creates) == 0
}

func (d PGSqlDocument) Sort() {

}

func (d PGSqlDocument) Include(other Document) {

}

func (d PGSqlDocument) ParsePragmas(s *Scanner) {

}

func (d PGSqlDocument) WithoutPos() Document {
	return &PGSqlDocument{}
}

func (d PGSqlDocument) ParseBatch(s *Scanner, isFirst bool) bool {
	return false
}
