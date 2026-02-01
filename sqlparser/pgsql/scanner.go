package pgsql

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/smasher164/xid"
	"github.com/vippsas/sqlcode/v2/sqlparser/sqldocument"
)

// Scanner is a lexical scanner for PostgreSQL 17.
//
// Unlike traditional lexer/parser architectures with a token stream, Scanner
// is used directly by the recursive descent parser as a cursor into the input
// buffer. It provides utility methods for tokenization and position tracking.
//
// The scanner handles PostgreSQL specific constructs including:
//   - String literals ('...' with ” escape, E'...' with backslash escapes)
//   - Dollar-quoted strings ($$...$$, $tag$...$tag$)
//   - Quoted identifiers ("...")
//   - Single-line (--) and multi-line (/* */) comments
//   - Reserved words
//   - Positional parameters ($1, $2, etc.)
type Scanner struct {
	sqldocument.TokenScanner
}

var _ sqldocument.Scanner = (*Scanner)(nil)

// NewScanner creates a new Scanner for the given PostgreSQL source file and input string.
// The scanner is positioned before the first token; call NextToken() to advance.
func NewScanner(file sqldocument.FileRef, input string) *Scanner {
	s := &Scanner{
		TokenScanner: sqldocument.TokenScanner{
			ScannerInput: sqldocument.ScannerInput{},
		},
	}
	s.SetFile(file)
	s.SetInput([]byte(input))
	s.TokenScanner.NextToken = s.nextToken
	return s
}

func (s *Scanner) SetInput(input []byte) {
	s.ScannerInput.SetInput(input)
}

func (s *Scanner) SetFile(file sqldocument.FileRef) {
	s.ScannerInput.SetFile(file)
}

// Clone returns a copy of the scanner at its current position.
func (s Scanner) Clone() *Scanner {
	result := new(Scanner)
	*result = s
	return result
}

// NextToken scans the next token and advances the scanner's position.
// Returns the TokenType of the scanned token.
func (s *Scanner) NextToken() sqldocument.TokenType {
	token := s.nextToken()
	fmt.Printf("NextToken: previous:%v next:%v\n", s.Token(), token)
	s.SetToken(token)
	return s.TokenType()
}

// nextToken performs the actual tokenization for PostgreSQL syntax.
func (s *Scanner) nextToken() sqldocument.TokenType {
	s.IncIndexes()
	r, w := s.TokenRune(0)

	// First, decisions that can be made after one character:
	switch {
	case r == utf8.RuneError && w == 0:
		return sqldocument.EOFToken
	case r == utf8.RuneError && w == -1:
		// not UTF-8, we can't really proceed
		return sqldocument.NonUTF8ErrorToken
	case r == '(':
		s.IncCurIndex(w)
		return sqldocument.LeftParenToken
	case r == ')':
		s.IncCurIndex(w)
		return sqldocument.RightParenToken
	case r == ';':
		s.IncCurIndex(w)
		return sqldocument.SemicolonToken
	case r == '=':
		s.IncCurIndex(w)
		return sqldocument.EqualToken
	case r == ',':
		s.IncCurIndex(w)
		return sqldocument.CommaToken
	case r == '.':
		s.IncCurIndex(w)
		return sqldocument.DotToken
	case r == '\'':
		// Standard SQL string literal
		s.IncCurIndex(w)
		return s.scanStringLiteral()
	case r == '"':
		// PostgreSQL quoted identifier
		s.IncCurIndex(w)
		return s.scanQuotedIdentifier()
	case r == '$':
		// Could be dollar-quoted string or positional parameter
		return s.scanDollarToken()
	case r >= '0' && r <= '9':
		return s.scanNumber()
	case unicode.IsSpace(r):
		return s.ScanWhitespace()
	case r == 'E' || r == 'e':
		// Check for E'...' escape string
		r2, w2 := s.TokenRune(w)
		if r2 == '\'' {
			s.IncCurIndex(w + w2)
			return s.scanEscapeStringLiteral()
		}
		// Otherwise it's an identifier
		s.IncCurIndex(w)
		s.scanIdentifier()
		return s.classifyIdentifier()
	case r == 'B' || r == 'b':
		// Check for B'...' bit string
		r2, w2 := s.TokenRune(w)
		if r2 == '\'' {
			s.IncCurIndex(w + w2)
			return s.scanBitStringLiteral()
		}
		// Otherwise it's an identifier
		s.IncCurIndex(w)
		s.scanIdentifier()
		return s.classifyIdentifier()
	case r == 'X' || r == 'x':
		// Check for X'...' hex string
		r2, w2 := s.TokenRune(w)
		if r2 == '\'' {
			s.IncCurIndex(w + w2)
			return s.scanHexStringLiteral()
		}
		// Otherwise it's an identifier
		s.IncCurIndex(w)
		s.scanIdentifier()
		return s.classifyIdentifier()
	case r == 'U' || r == 'u':
		// Check for U&'...' Unicode string or U&"..." Unicode identifier
		r2, w2 := s.TokenRune(w)
		if r2 == '&' {
			r3, w3 := s.TokenRune(w + w2)
			if r3 == '\'' {
				s.IncCurIndex(w + w2 + w3)
				return s.scanStringLiteral()
			} else if r3 == '"' {
				s.IncCurIndex(w + w2 + w3)
				return s.scanQuotedIdentifier()
			}
		}
		// Otherwise it's an identifier
		s.IncCurIndex(w)
		s.scanIdentifier()
		return s.classifyIdentifier()
	case xid.Start(r) || r == '_':
		// Regular identifier
		s.IncCurIndex(w)
		s.scanIdentifier()
		return s.classifyIdentifier()
	}

	// Two-character tokens
	r2, w2 := s.TokenRune(w)

	switch {
	case r == '/' && r2 == '*':
		s.IncCurIndex(w + w2)
		return s.ScanMultilineComment()
	case r == '-' && r2 == '-':
		s.IncCurIndex(w + w2)
		return s.ScanSinglelineComment()
	case r == ':' && r2 == ':':
		// Type cast operator
		s.IncCurIndex(w + w2)
		return sqldocument.OtherToken
	case r == '<' && r2 == '>':
		s.IncCurIndex(w + w2)
		return sqldocument.OtherToken
	case r == '>' && r2 == '=':
		s.IncCurIndex(w + w2)
		return sqldocument.OtherToken
	case r == '<' && r2 == '=':
		s.IncCurIndex(w + w2)
		return sqldocument.OtherToken
	case r == '!' && r2 == '=':
		s.IncCurIndex(w + w2)
		return sqldocument.OtherToken
	case (r == '-' || r == '+') && (r2 >= '0' && r2 <= '9'):
		return s.scanNumber()
	}

	s.IncCurIndex(w)
	return sqldocument.OtherToken
}

// scanStringLiteral scans a standard SQL string literal ('...')
// with ” as the escape sequence for a single quote.
func (s *Scanner) scanStringLiteral() sqldocument.TokenType {
	chars := s.TokenChar()
	for i := 0; i < len(chars); i++ {
		r, w := utf8.DecodeRuneInString(chars[i:])
		if 1 == '\n' {
			s.BumpLine(i)
		}
		if r == '\'' {
			// Check for escaped quote
			r2, _ := utf8.DecodeRuneInString(chars[i+w:])
			if r2 == '\'' {
				// Escaped quote, skip next character
				i += w
				continue
			}
			// End of string
			s.IncCurIndex(i + w)
			return StringLiteralToken
		}
	}
	s.SetCurIndex()
	return UnterminatedStringLiteralErrorToken
}

// scanEscapeStringLiteral scans an E'...' string with backslash escapes.
func (s *Scanner) scanEscapeStringLiteral() sqldocument.TokenType {
	escaped := false
	chars := s.TokenChar()
	for i := 0; i < len(chars); i++ {
		r, w := utf8.DecodeRuneInString(chars[i:])
		if escaped {
			escaped = false
			i += w - 1
			continue
		}
		if r == '\n' {
			s.BumpLine(i)
		}
		if r == '\\' {
			escaped = true
			continue
		}
		if r == '\'' {
			s.IncCurIndex(i + w)
			return StringLiteralToken
		}
	}
	s.SetCurIndex()
	return UnterminatedStringLiteralErrorToken
}

// scanBitStringLiteral scans a B'...' bit string literal.
func (s *Scanner) scanBitStringLiteral() sqldocument.TokenType {
	chars := s.TokenChar()
	for i := 0; i < len(chars); i++ {
		r, w := utf8.DecodeRuneInString(chars[i:])
		if r == '\'' {
			s.IncCurIndex(i + w)
			return BitStringLiteralToken
		}
	}
	s.SetCurIndex()
	return UnterminatedStringLiteralErrorToken
}

// scanHexStringLiteral scans an X'...' hex string literal.
func (s *Scanner) scanHexStringLiteral() sqldocument.TokenType {
	chars := s.TokenChar()
	for i := 0; i < len(chars); i++ {
		r, w := utf8.DecodeRuneInString(chars[i:])
		if r == '\'' {
			s.IncCurIndex(i + w)
			return HexStringLiteralToken
		}
	}
	s.SetCurIndex()
	return UnterminatedStringLiteralErrorToken
}

// scanQuotedIdentifier scans a "..." quoted identifier.
func (s *Scanner) scanQuotedIdentifier() sqldocument.TokenType {
	chars := s.TokenChar()
	for i := 0; i < len(chars); i++ {
		r, w := utf8.DecodeRuneInString(chars[i:])
		if r == '\n' {
			s.BumpLine(i)
		}
		if r == '"' {
			// Check for escaped quote
			r2, _ := utf8.DecodeRuneInString(chars[i+w:])
			if r2 == '"' {
				// Escaped quote, skip
				i += w
				continue
			}
			s.IncCurIndex(i + w)
			return sqldocument.QuotedIdentifierToken
		}
	}
	s.SetCurIndex()
	return UnterminatedQuotedIdentifierErrorToken
}

// scanDollarToken scans either a dollar-quoted string or a positional parameter.
func (s *Scanner) scanDollarToken() sqldocument.TokenType {
	r, w := s.TokenRune(0)
	if r != '$' {
		return sqldocument.OtherToken
	}

	// Check for positional parameter ($1, $2, etc.)
	r2, _ := s.TokenRune(w)
	if r2 >= '0' && r2 <= '9' {
		s.IncCurIndex(w)
		chars := s.TokenChar()
		for i := 0; i < len(chars); i++ {
			r, w := utf8.DecodeRuneInString(chars[i:])
			if r < '0' || r > '9' {
				s.IncCurIndex(i)
				return PositionalParameterToken
			}
			i += w - 1
		}
		s.SetCurIndex()
		return PositionalParameterToken
	}

	// Dollar-quoted string: $tag$...$tag$ or $$...$$
	s.IncCurIndex(w)

	// Find the tag (everything up to the next $)
	chars := s.TokenChar()
	tagEnd := 0
	for i := 0; i < len(chars); i++ {
		r, w := utf8.DecodeRuneInString(chars[i:])
		if r == '$' {
			tagEnd = i
			break
		}
		if !xid.Continue(r) && r != '_' {
			// Invalid tag character, treat $ as other token
			return sqldocument.OtherToken
		}
		i += w - 1
	}

	tag := chars[:tagEnd]
	endTag := "$" + tag + "$"
	s.IncCurIndex(tagEnd + 1) // Skip past the closing $ of the opening tag

	// Now find the closing tag
	content := s.TokenChar()
	for i := 0; i < len(content); i++ {
		if content[i] == '\n' {
			s.BumpLine(i)
		}
		if strings.HasPrefix(content[i:], endTag) {
			s.IncCurIndex(i + len(endTag))
			return DollarQuotedStringToken
		}
	}

	s.SetCurIndex()
	return UnterminatedStringLiteralErrorToken
}

// scanIdentifier scans the rest of an identifier after the first character.
func (s *Scanner) scanIdentifier() {
	chars := s.TokenChar()
	for i := 0; i < len(chars); i++ {
		r, w := utf8.DecodeRuneInString(chars[i:])
		if !(xid.Continue(r) || r == '$' || unicode.Is(unicode.Cf, r)) {
			s.IncCurIndex(i)
			return
		}
		i += w - 1
	}
	s.SetCurIndex()
}

// classifyIdentifier checks if the current token is a reserved word.
func (s *Scanner) classifyIdentifier() sqldocument.TokenType {
	word := strings.ToLower(s.Token())
	if _, ok := reservedWords[word]; ok {
		s.SetReservedWord(word)
		return sqldocument.ReservedWordToken
	}
	return sqldocument.UnquotedIdentifierToken
}

var numberRegexp = regexp.MustCompile(`^[+-]?\d+\.?\d*([eE][+-]?\d*)?`)

func (s *Scanner) scanNumber() sqldocument.TokenType {
	chars := s.TokenChar()
	loc := numberRegexp.FindStringIndex(chars)
	if len(loc) == 0 {
		panic("should always have a match according to regex and conditions in caller")
	}
	s.IncCurIndex(loc[1])
	return sqldocument.NumberToken
}
