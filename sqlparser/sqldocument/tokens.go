package sqldocument

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/vippsas/sqlcode/v2/sqlparser/internal/utils"
)

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

type ScannerInput struct {
	input string
	file  FileRef
}

func (si *ScannerInput) SetInput(input []byte) {
	si.input = string(input)
}

func (si *ScannerInput) SetFile(file FileRef) {
	si.file = file
}

type TokenScanner struct {
	ScannerInput
	NextToken  func() TokenType // Function to find to the next token
	startIndex int              // Byte index where current token starts
	curIndex   int              // Current byte position in Input
	tokenType  TokenType        // Type of the current token

	startLine        int // Line number (0-indexed) where current token starts
	stopLine         int // Line number (0-indexed) where current token ends
	indexAtStartLine int // Byte index at the start of startLine (after newline)
	indexAtStopLine  int // Byte index at the start of stopLine (after newline)

	reservedWord string // Lowercase version of token if it's a reserved word, empty otherwise
}

func (s *TokenScanner) IncIndexes() {
	s.startIndex = s.curIndex
	s.startLine = s.stopLine
	s.indexAtStartLine = s.indexAtStopLine
	s.SetReservedWord("")
}

// TokenType returns the type of the current token.
func (s *TokenScanner) TokenType() TokenType {
	return s.tokenType
}

func (s *TokenScanner) SetToken(token TokenType) {
	s.tokenType = token
}

func (s *TokenScanner) Input() string {
	return s.input
}

// Token returns the text of the current token as a substring of Input.
func (s *TokenScanner) Token() string {
	return s.input[s.startIndex:s.curIndex]
}

// TokenLower returns the current token text converted to lowercase.
// Useful for case-insensitive keyword matching.
func (s *TokenScanner) TokenLower() string {
	return strings.ToLower(s.Token())
}

func (s *TokenScanner) TokenRune(peek int) (rune, int) {
	i := s.curIndex + peek
	return utf8.DecodeRuneInString(s.input[i:])
}

func (s *TokenScanner) TokenChar() string {
	return s.input[s.curIndex:]
}

// ReservedWord returns the lowercase reserved word if the current token
// is a ReservedWordToken, or an empty string otherwise.
func (s *TokenScanner) ReservedWord() string {
	return s.reservedWord
}

func (s *TokenScanner) SetReservedWord(wrd string) {
	s.reservedWord = wrd
}

func (s *TokenScanner) IncCurIndex(i int) {
	s.curIndex += i
}

func (s *TokenScanner) SetCurIndex() {
	s.curIndex = len(s.input)
}

// Start returns the position where the current token begins.
// Line and column are 1-indexed.
func (s *TokenScanner) Start() Pos {
	return Pos{
		Line: s.startLine + 1,
		Col:  s.startIndex - s.indexAtStartLine + 1,
		File: s.file,
	}
}

// If we just saw the whitespace token that bumped the linecount,
// we are at the "start of line", even if this contains some space after the \n:
func (s *TokenScanner) IsStartOfLine() bool {
	return s.stopLine > s.startLine
}

// Stop returns the position where the current token ends.
// Line and column are 1-indexed.
func (s *TokenScanner) Stop() Pos {
	return Pos{
		Line: s.stopLine + 1,
		Col:  s.curIndex - s.indexAtStopLine + 1,
		File: s.file,
	}
}

// bumpLine increments the line counter and records the byte position
// after the newline character. The offset parameter is the position
// of the newline within the current scan operation.
func (s *TokenScanner) BumpLine(offset int) {
	s.stopLine++
	s.indexAtStopLine = s.curIndex + offset + 1
}

// SkipWhitespaceComments advances past any whitespace and comment tokens.
// Stops when a non-whitespace, non-comment token is encountered.
func (s *TokenScanner) SkipWhitespaceComments() {
	for {
		switch s.TokenType() {
		case WhitespaceToken, MultilineCommentToken, SinglelineCommentToken:
		default:
			return
		}
		s.NextToken()
	}
}

// SkipWhitespace advances past any whitespace tokens.
// Stops when a non-whitespace token is encountered.
// Unlike SkipWhitespaceComments, this preserves comments.
func (s *TokenScanner) SkipWhitespace() {
	for {
		switch s.TokenType() {
		case WhitespaceToken:
		default:
			return
		}
		s.NextToken()
	}
}

// NextNonWhitespaceToken advances to the next token and then skips
// any whitespace, returning the type of the first non-whitespace token.
func (s *TokenScanner) NextNonWhitespaceToken() TokenType {
	utils.DPrint("NextNonWhitespaceToken called at index %d\n", s.curIndex)
	utils.DPrint("%#v\n", s.NextToken)
	s.NextToken()
	s.SkipWhitespace()
	return s.TokenType()
}

// NextNonWhitespaceCommentToken advances to the next token and then skips
// any whitespace and comments, returning the type of the first significant token.
func (s *TokenScanner) NextNonWhitespaceCommentToken() TokenType {
	s.NextToken()
	s.SkipWhitespaceComments()
	return s.TokenType()
}

// scanMultilineComment assumes one has advanced over '/*'
func (s *TokenScanner) ScanMultilineComment() TokenType {
	prevWasStar := false
	for i, r := range s.input[s.curIndex:] {
		if r == '*' {
			prevWasStar = true
		} else if prevWasStar && r == '/' {
			s.curIndex += i + 1
			return MultilineCommentToken
		} else if r == '\n' {
			s.BumpLine(i)
		}
	}
	s.curIndex = len(s.input)
	return MultilineCommentToken
}

// scanSinglelineComment assumes one has advanced over --
func (s *TokenScanner) ScanSinglelineComment() TokenType {
	utils.DPrint("Scanning singleline comment at index %d: %#q\n", s.curIndex, s.input[s.curIndex:])
	isPragma := strings.HasPrefix(s.input[s.curIndex:], "sqlcode:")
	end := strings.Index(s.input[s.curIndex:], "\n")
	if end == -1 {
		// end of file is also end of stopLine. But we're done
		s.curIndex = len(s.input)
	} else {
		// hmm, is the \n at the end part of the token or a new whitespace?
		// making it part of whitespace seems simpler somehow..
		s.curIndex += end
	}
	if isPragma {
		utils.DPrint("Found pragma comment: %#q\n", s.input[s.startIndex:s.curIndex])
		return PragmaToken
	} else {
		return SinglelineCommentToken
	}
}

func (s *TokenScanner) ScanWhitespace() TokenType {
	for i, r := range s.input[s.curIndex:] {
		if r == '\n' {
			s.BumpLine(i)
		}
		if !unicode.IsSpace(r) {
			s.curIndex += i
			return WhitespaceToken
		}
	}
	// eof
	s.curIndex = len(s.input)
	return WhitespaceToken
}
