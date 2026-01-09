package mssql

import "github.com/simukka/sqlcode/sqlparser/sqldocument"

// T-SQL specific tokens (range 1000-1999)
//
// Token values are partitioned by dialect to avoid collisions:
//   - 0-999:    Common tokens shared across dialects (sqldocument package)
//   - 1000-1999: T-SQL specific tokens (this package)
//   - 2000-2999: Reserved for other dialects (e.g., PostgreSQL)
//
// This design allows dialect-specific code to use concrete token types
// while common code can use ToCommonToken() for abstraction.
const (
	// T-SQL specific string literals
	//
	// T-SQL distinguishes between varchar ('...') and nvarchar (N'...')
	// string literals. Both use single quotes with '' as the escape sequence.
	VarcharLiteralToken sqldocument.TokenType = iota + sqldocument.TSQLTokenStart
	NVarcharLiteralToken

	// T-SQL specific identifier styles
	//
	// T-SQL uses square brackets for quoted identifiers: [My Table]
	// Brackets are escaped by doubling: [My]]Table] represents "My]Table"
	BracketQuotedIdentifierToken // [identifier]

	// T-SQL specific errors
	//
	// Unlike standard SQL, T-SQL does not support double-quoted strings.
	// Double quotes are reserved for QUOTED_IDENTIFIER mode identifiers,
	// but sqlcode requires bracket notation for consistency.
	DoubleQuoteErrorToken // T-SQL doesn't support double-quoted strings
	UnterminatedVarcharLiteralErrorToken
	UnterminatedQuotedIdentifierErrorToken
)

// ToCommonToken maps T-SQL specific tokens to their common equivalents
// for dialect-agnostic processing.
//
// This abstraction layer allows higher-level code to work with logical
// token categories (e.g., "string literal") without knowing the specific
// dialect syntax (varchar vs nvarchar, brackets vs double quotes).
//
// Tokens that are already common tokens pass through unchanged.
func ToCommonToken(tt sqldocument.TokenType) sqldocument.TokenType {
	switch tt {
	case VarcharLiteralToken, NVarcharLiteralToken:
		return sqldocument.StringLiteralToken
	case BracketQuotedIdentifierToken:
		return sqldocument.QuotedIdentifierToken
	case UnterminatedVarcharLiteralErrorToken, UnterminatedQuotedIdentifierErrorToken:
		return sqldocument.UnterminatedStringErrorToken
	default:
		return tt
	}
}
