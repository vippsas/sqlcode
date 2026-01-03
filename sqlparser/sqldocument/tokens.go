package sqldocument

// TokenType represents the type of a lexical token.
// Common tokens are defined in the range 1-999.
// Dialect-specific tokens use ranges starting at 1000, 2000, etc.
type TokenType int

// Token range constants for dialect-specific extensions.
const (
	// CommonTokenStart is the start of common token range (1-999)
	CommonTokenStart TokenType = 1
	// TSQLTokenStart is the start of T-SQL specific tokens (1000-1999)
	TSQLTokenStart TokenType = 1000
	// PGSQLTokenStart is the start of PostgreSQL specific tokens (2000-2999)
	PGSQLTokenStart TokenType = 2000
)

// Common tokens shared across all SQL dialects.
// These represent fundamental SQL constructs.
const (
	// Structural tokens
	EOFToken TokenType = iota + 1
	WhitespaceToken
	LeftParenToken
	RightParenToken
	SemicolonToken
	EqualToken
	CommaToken
	DotToken

	// Literals
	StringLiteralToken // Generic string literal (dialect determines quote style)
	NumberToken

	// Comments
	MultilineCommentToken
	SinglelineCommentToken

	// Identifiers
	ReservedWordToken
	IdentifierToken // Generic identifier (quoted or unquoted)
	QuotedIdentifierToken
	UnquotedIdentifierToken
	VariableIdentifierToken

	// Special
	OtherToken

	// Errors
	UnterminatedStringErrorToken
	UnterminatedIdentifierErrorToken
	NonUTF8ErrorToken

	// Batch/statement separation (common concept, dialect-specific syntax)
	BatchSeparatorToken
	MalformedBatchSeparatorToken

	// Pragma (sqlcode-specific)
	PragmaToken
)

// TokenCategory represents broad categories of tokens for dialect-agnostic handling.
type TokenCategory int

const (
	CategoryUnknown TokenCategory = iota
	CategoryWhitespace
	CategoryComment
	CategoryLiteral
	CategoryIdentifier
	CategoryKeyword
	CategoryOperator
	CategoryPunctuation
	CategoryEOF
	CategoryError
)

// Category returns the broad category of a token for dialect-agnostic processing.
func (tt TokenType) Category() TokenCategory {
	switch tt {
	case EOFToken:
		return CategoryEOF
	case WhitespaceToken:
		return CategoryWhitespace
	case MultilineCommentToken, SinglelineCommentToken, PragmaToken:
		return CategoryComment
	case StringLiteralToken, NumberToken:
		return CategoryLiteral
	case IdentifierToken, QuotedIdentifierToken, UnquotedIdentifierToken, VariableIdentifierToken:
		return CategoryIdentifier
	case ReservedWordToken:
		return CategoryKeyword
	case LeftParenToken, RightParenToken, SemicolonToken, CommaToken, DotToken:
		return CategoryPunctuation
	case EqualToken:
		return CategoryOperator
	case UnterminatedStringErrorToken, UnterminatedIdentifierErrorToken, NonUTF8ErrorToken:
		return CategoryError
	default:
		return CategoryUnknown
	}
}
