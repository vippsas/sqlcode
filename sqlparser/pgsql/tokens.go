package pgsql

import "github.com/vippsas/sqlcode/v2/sqlparser/sqldocument"

// PostgreSQL specific tokens (range 2000-2999)
//
// Token values are partitioned by dialect to avoid collisions:
//   - 0-999:    Common tokens shared across dialects (sqldocument package)
//   - 1000-1999: T-SQL specific tokens (mssql package)
//   - 2000-2999: PostgreSQL specific tokens (this package)
const (
	// PostgreSQL string literal types
	StringLiteralToken sqldocument.TokenType = iota + sqldocument.PGSQLTokenStart

	// Dollar-quoted string ($$...$$ or $tag$...$tag$)
	DollarQuotedStringToken

	// Bit string literal (B'01010')
	BitStringLiteralToken

	// Hex string literal (X'1A2B')
	HexStringLiteralToken

	// Positional parameter ($1, $2, etc.)
	PositionalParameterToken

	// Error tokens
	UnterminatedStringLiteralErrorToken
	UnterminatedQuotedIdentifierErrorToken
)
