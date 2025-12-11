package sqlparser

import (
	"path/filepath"
	"strings"
)

// Document represents a parsed SQL document, containing
// declarations, create statements, pragmas, and errors.
// It provides methods to access and manipulate these components
// for T-SQL and PostgreSQL
type Document interface {
	Empty() bool
	HasErrors() bool

	Creates() []Create
	Declares() []Declare
	Errors() []Error
	PragmaIncludeIf() []string
	Include(other Document)
	Sort()
	ParsePragmas(s *Scanner)
	ParseBatch(s *Scanner, isFirst bool) (hasMore bool)

	WithoutPos() Document
}

// Helper function to parse a SQL document from a string input
func ParseString(filename FileRef, input string) (result Document) {
	result = NewDocumentFromExtension(filepath.Ext(strings.ToLower(string(filename))))
	Parse(&Scanner{input: input, file: filename}, result)
	return
}

// Based on the input file extension, create the appropriate Document type
func NewDocumentFromExtension(extension string) Document {
	switch extension {
	case ".sql":
		return &TSqlDocument{}
	case ".pgsql":
		return &PGSqlDocument{}
	default:
		panic("unhandled document type: " + extension)
	}
}
