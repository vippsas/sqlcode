package sqldocument

import (
	"fmt"
	"slices"
)

// Document represents a SQL document that needs to be parsed.
// It will contain declarations, create statements, pragmas, and errors.
// It provides methods to access and manipulate these components
// for supported database drivers
type Document interface {
	// Parse parses the input SQL document and populates the Document structure.
	Parse(input []byte, file FileRef) error
	// Returns true if the document contains no create statements or declarations
	Empty() bool
	// Returns true if the document contains any parsing errors
	HasErrors() bool
	// Returns the list of create statements in the document
	Creates() []Create
	// Returns the list of variable declarations in the document
	Declares() []Declare
	// Returns the list of parsing errors in the document
	Errors() []Error
	// Returns the list of pragma include-if statements in the document
	PragmaIncludeIf() []string
	// Includes the content of another Document into this one
	Include(other Document)
	// Performs a topological sort of the create statements based on their dependencies
	Sort()
}

func CopyToken(s Scanner, target *[]Unparsed) {
	*target = append(*target, CreateUnparsed(s))
}

// parseCodeschemaName parses `[code] . something`, and returns `something`
// in quoted form (`[something]`). Also copy to `target`. Empty string on error.
// Note: To follow conventions, consume one extra token at the end even if we know
// it fill not be consumed by this function...
func ParseCodeschemaName(s Scanner, target *[]Unparsed, statementTokens []string) (PosString, error) {
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
func NextTokenCopyingWhitespace(s Scanner, target *[]Unparsed) {
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

func RecoverToNextStatementCopying(s Scanner, target *[]Unparsed, StatementTokens []string) {
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

func RecoverToNextStatement(s Scanner, StatementTokens []string) {
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
