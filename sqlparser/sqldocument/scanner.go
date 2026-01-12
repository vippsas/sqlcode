package sqldocument

// Scanner defines the interface for lexical scanning of SQL source code.
//
// This interface abstracts the Scanner implementation, enabling:
//   - Unit testing with mock scanners
//   - Alternative scanner implementations for different SQL dialects
//   - Easier dependency injection in parsers
//
// Implementations should return dialect-specific tokens from TokenType(),
// but CommonTokenType() should map these to common tokens for dialect-agnostic code.
type Scanner interface {
	// TokenType returns the type of the current token.
	// This may return dialect-specific token types.
	TokenType() TokenType

	// Token returns the text of the current token.
	Token() string

	// TokenLower returns the current token text converted to lowercase.
	TokenLower() string

	// ReservedWord returns the lowercase reserved word if the current token
	// is a ReservedWordToken, or an empty string otherwise.
	ReservedWord() string

	// Start returns the position where the current token begins.
	Start() Pos

	// Stop returns the position where the current token ends.
	Stop() Pos

	// NextToken scans the next token and advances the scanner's position.
	NextToken() TokenType

	// NextNonWhitespaceToken advances to the next non-whitespace token.
	NextNonWhitespaceToken() TokenType

	// NextNonWhitespaceCommentToken advances to the next significant token.
	NextNonWhitespaceCommentToken() TokenType

	// SkipWhitespace advances past any whitespace tokens.
	SkipWhitespace()

	// SkipWhitespaceComments advances past any whitespace and comment tokens.
	SkipWhitespaceComments()

	// Set the scanner's input to the given byte slice.
	SetInput([]byte)

	// Set the scanner's input file reference.
	SetFile(FileRef)
}
