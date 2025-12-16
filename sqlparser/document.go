package sqlparser

import (
	"fmt"
	"path/filepath"
	"slices"
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
	Parse(s *Scanner) error
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

// parseCodeschemaName parses `[code] . something`, and returns `something`
// in quoted form (`[something]`). Also copy to `target`. Empty string on error.
// Note: To follow conventions, consume one extra token at the end even if we know
// it fill not be consumed by this function...
func ParseCodeschemaName(s *Scanner, target *[]Unparsed, statementTokens []string) (PosString, error) {
	CopyToken(s, target)
	NextTokenCopyingWhitespace(s, target)
	if s.TokenType() != DotToken {
		RecoverToNextStatementCopying(s, target, statementTokens)
		return PosString{Value: ""}, fmt.Errorf("[code] must be followed by '.'")
	}

	CopyToken(s, target)

	NextTokenCopyingWhitespace(s, target)
	switch s.TokenType() {
	case UnquotedIdentifierToken:
		// To get something uniform for comparison, quote all names
		CopyToken(s, target)
		result := PosString{Pos: s.Start(), Value: "[" + s.Token() + "]"}
		NextTokenCopyingWhitespace(s, target)
		return result, nil
	case QuotedIdentifierToken:
		CopyToken(s, target)
		result := PosString{Pos: s.Start(), Value: s.Token()}
		NextTokenCopyingWhitespace(s, target)
		return result, nil
	default:
		RecoverToNextStatementCopying(s, target, statementTokens)
		return PosString{Value: ""}, fmt.Errorf("[code]. must be followed an identifier")
	}
}

// NextTokenCopyingWhitespace is like s.NextToken(), but if whitespace is encountered
// it is simply copied into `target`. Upon return, the scanner is located at a non-whitespace
// token, and target is either unmodified or filled with some whitespace nodes.
func NextTokenCopyingWhitespace(s *Scanner, target *[]Unparsed) {
	for {
		tt := s.NextToken()
		switch tt {
		case EOFToken, BatchSeparatorToken:
			// do not copy
			return
		case WhitespaceToken, MultilineCommentToken, SinglelineCommentToken:
			// copy, and loop around
			CopyToken(s, target)
			continue
		default:
			return
		}
	}

}

func RecoverToNextStatementCopying(s *Scanner, target *[]Unparsed, StatementTokens []string) {
	// We hit an unexpected token ... as an heuristic for continuing parsing,
	// skip parsing until we hit a reserved word that starts a statement
	// we recognize
	for {
		NextTokenCopyingWhitespace(s, target)
		switch s.TokenType() {
		case ReservedWordToken:
			if slices.Contains(StatementTokens, s.ReservedWord()) {
				return
			}
		case EOFToken:
			return
		default:
			CopyToken(s, target)
		}
	}
}

func RecoverToNextStatement(s *Scanner, StatementTokens []string) {
	// We hit an unexpected token ... as an heuristic for continuing parsing,
	// skip parsing until we hit a reserved word that starts a statement
	// we recognize
	for {
		s.NextNonWhitespaceCommentToken()
		switch s.TokenType() {
		case ReservedWordToken:
			if slices.Contains(StatementTokens, s.ReservedWord()) {
				return
			}
		case EOFToken:
			return
		}
	}
}
