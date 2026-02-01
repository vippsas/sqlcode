package mssql

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/smasher164/xid"
	"github.com/vippsas/sqlcode/v2/sqlparser/sqldocument"
)

// MSSqlScanner is a lexical scanner for T-SQL source code.
//
// Unlike traditional lexer/parser architectures with a token stream, MSSqlScanner
// is used directly by the recursive descent parser as a cursor into the input
// buffer. It provides utility methods for tokenization and position tracking.
//
// The scanner handles T-SQL specific constructs including:
//   - String literals ('...' and N'...')
//   - Quoted identifiers ([...])
//   - Single-line (--) and multi-line (/* */) comments
//   - Batch separators (GO)
//   - Reserved words
//   - Variables (@identifier)
type MSSqlScanner struct {
	sqldocument.TokenScanner

	afterBatchSeparator bool // True if we just saw GO; used to detect malformed separators
	startOfLine         bool // True if no non-whitespace/comment seen since start of line
}

var _ sqldocument.Scanner = (*MSSqlScanner)(nil)

// NewScanner creates a new Scanner for the given T-SQL source file and input string.
// The scanner is positioned before the first token; call NextToken() to advance.
func NewScanner(file sqldocument.FileRef, input string) *MSSqlScanner {
	s := &MSSqlScanner{
		TokenScanner: sqldocument.TokenScanner{
			ScannerInput: sqldocument.ScannerInput{},
		},
	}
	s.SetFile(file)
	s.SetInput([]byte(input))

	s.TokenScanner.NextToken = s.NextToken
	return s
}

func (s *MSSqlScanner) SetInput(input []byte) {
	s.ScannerInput.SetInput(input)
}

func (s *MSSqlScanner) SetFile(file sqldocument.FileRef) {
	s.ScannerInput.SetFile(file)
}

// Clone returns a copy of the scanner at its current position.
// This is used for look-ahead parsing where we need to tentatively
// scan tokens without committing to consuming them.
func (s MSSqlScanner) Clone() *MSSqlScanner {
	result := new(MSSqlScanner)
	*result = s
	return result
}

// NextToken scans the next token and advances the scanner's position.
//
// This method wraps the raw tokenization with batch separator handling.
// The GO batch separator has special rules in T-SQL:
//   - It must appear at the start of a line (only whitespace/comments before it)
//   - Nothing except whitespace may follow it on the same line
//   - It is not processed inside [names], 'strings', or /*comments*/
//
// If GO is followed by non-whitespace on the same line, subsequent tokens
// are returned as MalformedBatchSeparatorToken until end of line.
//
// Returns the TokenType of the scanned token.
func (s *MSSqlScanner) NextToken() sqldocument.TokenType {
	// handle startOfLine flag here; this is used to parse the 'go' batch separator
	token := s.nextToken()
	s.SetToken(token)

	// We handle the 'go' batch separator entirely in this extra layer above the
	// raw nextToken().

	// Handling the 'go' batch separator is a bit tricky; trying to reproduce some of sqlcmd rules.
	// The main thing is we don't process 'go' inside [names], 'strings' or /*comments*/;
	// this seems sane.

	// sqlcmd will allow /*comment*/ before or after the go separator; in any number,
	// but before only as long as there is no whitespace in between. Reproducing this seems a bit much
	// like "bug-compatability". To make this very simple: We do not allow comments
	// on the same line as 'go'. And doing so will not turn it into a literal,
	// but instead return MalformedBatchSeparatorToken
	if s.startOfLine && s.TokenType() == sqldocument.UnquotedIdentifierToken && s.TokenLower() == "go" {
		s.SetToken(sqldocument.BatchSeparatorToken)
		s.afterBatchSeparator = true
	} else if s.afterBatchSeparator && s.TokenType() != sqldocument.WhitespaceToken && s.TokenType() != sqldocument.EOFToken {
		s.SetToken(sqldocument.MalformedBatchSeparatorToken)
	} else if s.TokenType() == sqldocument.WhitespaceToken {
		// If we just saw the whitespace token that bumped the linecount,
		// we are at the "start of line", even if this contains some space after the \n:
		if s.IsStartOfLine() {
			s.startOfLine = true
			s.afterBatchSeparator = false
		}
		// Also, don't change the state if we did not bump the line
	} else {
		s.startOfLine = false
	}

	return s.TokenType()
}

// mssql17
func (s *MSSqlScanner) nextToken() sqldocument.TokenType {
	s.IncIndexes()
	r, w := s.TokenRune(0)

	// First, decisions that can be made after one character:
	switch {
	case r == utf8.RuneError && w == 0:
		return sqldocument.EOFToken
	case r == utf8.RuneError && w == -1:
		// not UTF-8, we can't really proceed so not advancing Scanner,
		// caller should take care to always exit..
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
		s.IncCurIndex(w)
		return s.scanStringLiteral(VarcharLiteralToken)
	case r >= '0' && r <= '9':
		return s.scanNumber()
	case r == '[':
		s.IncCurIndex(w)
		return s.scanQuotedIdentifier()
	case r == '"':
		// parser don't support double-quoted literals, just return an error token
		s.IncCurIndex(w)
		return DoubleQuoteErrorToken
	case unicode.IsSpace(r):
		// do not advance s.curIndex here, simpler to do it all in scanWhitespace(); in case r == '\n' we need stopLine number bump
		return s.ScanWhitespace()
	case r != 'N' && (xid.Start(r) || r == '@' || r == '_' || r == '＿' || r == '#'): // Unicode Start identifier
		// good guide for identifiers:
		// https://sqlquantumleap.com/reference/completely-complete-list-of-rules-for-t-sql-identifiers/
		s.IncCurIndex(w)
		s.scanIdentifier()
		if r == '@' {
			return sqldocument.VariableIdentifierToken
		} else {
			rw := strings.ToLower(s.Token())
			_, ok := reservedWords[rw]
			if ok {
				s.SetReservedWord(rw)
				return sqldocument.ReservedWordToken
			} else {
				return sqldocument.UnquotedIdentifierToken
			}
		}
	}

	// OK, we need to peek 1 character to make a decision
	r2, w2 := s.TokenRune(w)

	switch {
	case r == 'N':
		s.IncCurIndex(w)
		if r2 == '\'' { // N'varchar string' .. only upper-case N allowed
			s.IncCurIndex(w2)
			return s.scanStringLiteral(NVarcharLiteralToken)
		}
		// no, it is instead an identifier starting with N...
		s.scanIdentifier()
		rw := strings.ToLower(s.Token())
		_, ok := reservedWords[rw]
		if ok {
			s.SetReservedWord(rw)
			return sqldocument.ReservedWordToken
		} else {
			return sqldocument.UnquotedIdentifierToken
		}
	case r == '/' && r2 == '*':
		s.IncCurIndex(w + w2)
		return s.ScanMultilineComment()
	case r == '-' && r2 == '-':
		s.IncCurIndex(w + w2)
		return s.ScanSinglelineComment()
	case (r == '-' || r == '+') && (r2 >= '0' && r2 <= '9'):
		return s.scanNumber()
	}

	s.IncCurIndex(w)
	return sqldocument.OtherToken
}

// scanStringLiteral assumes one has scanned ' or N' (depending on param);
// then scans until the end of the string
func (s *MSSqlScanner) scanStringLiteral(tokenType sqldocument.TokenType) sqldocument.TokenType {
	return s.scanUntilSingleDoubleEscapes('\'', tokenType, UnterminatedVarcharLiteralErrorToken)
}

func (s *MSSqlScanner) scanQuotedIdentifier() sqldocument.TokenType {
	return s.scanUntilSingleDoubleEscapes(']', sqldocument.QuotedIdentifierToken, UnterminatedQuotedIdentifierErrorToken)
}

// scanIdentifier assumes first character of an identifier has been identified,
// and scans to the end
func (s *MSSqlScanner) scanIdentifier() {
	for i, r := range s.TokenChar() {
		if !(xid.Continue(r) || r == '$' || r == '#' || r == '@' || unicode.Is(unicode.Cf, r)) {
			s.IncCurIndex(i)
			return
		}
	}
	s.SetCurIndex()
}

// DRY helper to handle both ” and ]] escapes
func (s *MSSqlScanner) scanUntilSingleDoubleEscapes(
	endmarker rune,
	tokenType sqldocument.TokenType,
	unterminatedTokenType sqldocument.TokenType,
) sqldocument.TokenType {
	skipnext := false
	for i, r := range s.TokenChar() {
		if skipnext {
			skipnext = false
			continue
		}
		if r == '\n' {
			s.BumpLine(i)
		}
		if r == endmarker {
			r2, _ := s.TokenRune(i + 1) // r2 may be RuneError if eof
			if r2 == endmarker {
				// we have a double endmarker; this is used as escape
				skipnext = true
			} else {
				// terminating '
				s.IncCurIndex(i + 1)
				return tokenType
			}
		}
	}
	s.SetCurIndex()
	return unterminatedTokenType
}

var numberRegexp = regexp.MustCompile(`^[+-]?\d+\.?\d*([eE][+-]?\d*)?`)

func (s *MSSqlScanner) scanNumber() sqldocument.TokenType {
	// T-SQL seems to scan a number until the
	// end and then allowing a literal to start without whitespace or other things
	// in between...

	// https://docs.microsoft.com/en-us/sql/odbc/reference/appendixes/numeric-literal-syntax?view=sql-server-ver16

	char := s.TokenChar()
	loc := numberRegexp.FindStringIndex(char)
	if len(loc) == 0 {
		panic("should always have a match according to regex and conditions in caller")
	}
	s.IncCurIndex(loc[1])
	return sqldocument.NumberToken
}

// tsql (mssql) reservered words
var reservedWords = map[string]struct{}{
	"add":                            struct{}{},
	"external":                       struct{}{},
	"procedure":                      struct{}{},
	"all":                            struct{}{},
	"fetch":                          struct{}{},
	"public":                         struct{}{},
	"alter":                          struct{}{},
	"file":                           struct{}{},
	"raiserror":                      struct{}{},
	"and":                            struct{}{},
	"fillfactor":                     struct{}{},
	"read":                           struct{}{},
	"any":                            struct{}{},
	"for":                            struct{}{},
	"readtext":                       struct{}{},
	"as":                             struct{}{},
	"foreign":                        struct{}{},
	"reconfigure":                    struct{}{},
	"asc":                            struct{}{},
	"freetext":                       struct{}{},
	"references":                     struct{}{},
	"authorization":                  struct{}{},
	"freetexttable":                  struct{}{},
	"replication":                    struct{}{},
	"backup":                         struct{}{},
	"from":                           struct{}{},
	"restore":                        struct{}{},
	"begin":                          struct{}{},
	"full":                           struct{}{},
	"restrict":                       struct{}{},
	"between":                        struct{}{},
	"function":                       struct{}{},
	"return":                         struct{}{},
	"break":                          struct{}{},
	"goto":                           struct{}{},
	"revert":                         struct{}{},
	"browse":                         struct{}{},
	"grant":                          struct{}{},
	"revoke":                         struct{}{},
	"bulk":                           struct{}{},
	"group":                          struct{}{},
	"right":                          struct{}{},
	"by":                             struct{}{},
	"having":                         struct{}{},
	"rollback":                       struct{}{},
	"cascade":                        struct{}{},
	"holdlock":                       struct{}{},
	"rowcount":                       struct{}{},
	"case":                           struct{}{},
	"identity":                       struct{}{},
	"rowguidcol":                     struct{}{},
	"check":                          struct{}{},
	"identity_insert":                struct{}{},
	"rule":                           struct{}{},
	"checkpoint":                     struct{}{},
	"identitycol":                    struct{}{},
	"save":                           struct{}{},
	"close":                          struct{}{},
	"if":                             struct{}{},
	"schema":                         struct{}{},
	"clustered":                      struct{}{},
	"in":                             struct{}{},
	"securityaudit":                  struct{}{},
	"coalesce":                       struct{}{},
	"index":                          struct{}{},
	"select":                         struct{}{},
	"collate":                        struct{}{},
	"inner":                          struct{}{},
	"semantickeyphrasetable":         struct{}{},
	"column":                         struct{}{},
	"insert":                         struct{}{},
	"semanticsimilaritydetailstable": struct{}{},
	"commit":                         struct{}{},
	"intersect":                      struct{}{},
	"semanticsimilaritytable":        struct{}{},
	"compute":                        struct{}{},
	"into":                           struct{}{},
	"session_user":                   struct{}{},
	"constraint":                     struct{}{},
	"is":                             struct{}{},
	"set":                            struct{}{},
	"contains":                       struct{}{},
	"join":                           struct{}{},
	"setuser":                        struct{}{},
	"containstable":                  struct{}{},
	"key":                            struct{}{},
	"shutdown":                       struct{}{},
	"continue":                       struct{}{},
	"kill":                           struct{}{},
	"some":                           struct{}{},
	"convert":                        struct{}{},
	"left":                           struct{}{},
	"statistics":                     struct{}{},
	"create":                         struct{}{},
	"like":                           struct{}{},
	"system_user":                    struct{}{},
	"cross":                          struct{}{},
	"lineno":                         struct{}{},
	"table":                          struct{}{},
	"current":                        struct{}{},
	"load":                           struct{}{},
	"tablesample":                    struct{}{},
	"current_date":                   struct{}{},
	"merge":                          struct{}{},
	"textsize":                       struct{}{},
	"current_time":                   struct{}{},
	"national":                       struct{}{},
	"then":                           struct{}{},
	"current_timestamp":              struct{}{},
	"nocheck":                        struct{}{},
	"to":                             struct{}{},
	"current_user":                   struct{}{},
	"nonclustered":                   struct{}{},
	"top":                            struct{}{},
	"cursor":                         struct{}{},
	"not":                            struct{}{},
	"tran":                           struct{}{},
	"database":                       struct{}{},
	"null":                           struct{}{},
	"transaction":                    struct{}{},
	"dbcc":                           struct{}{},
	"nullif":                         struct{}{},
	"trigger":                        struct{}{},
	"deallocate":                     struct{}{},
	"of":                             struct{}{},
	"truncate":                       struct{}{},
	"declare":                        struct{}{},
	"off":                            struct{}{},
	"try_convert":                    struct{}{},
	"default":                        struct{}{},
	"offsets":                        struct{}{},
	"tsequal":                        struct{}{},
	"delete":                         struct{}{},
	"on":                             struct{}{},
	"union":                          struct{}{},
	"deny":                           struct{}{},
	"open":                           struct{}{},
	"unique":                         struct{}{},
	"desc":                           struct{}{},
	"opendatasource":                 struct{}{},
	"unpivot":                        struct{}{},
	"disk":                           struct{}{},
	"openquery":                      struct{}{},
	"update":                         struct{}{},
	"distinct":                       struct{}{},
	"openrowset":                     struct{}{},
	"updatetext":                     struct{}{},
	"distributed":                    struct{}{},
	"openxml":                        struct{}{},
	"use":                            struct{}{},
	"double":                         struct{}{},
	"option":                         struct{}{},
	"user":                           struct{}{},
	"drop":                           struct{}{},
	"or":                             struct{}{},
	"values":                         struct{}{},
	"dump":                           struct{}{},
	"order":                          struct{}{},
	"varying":                        struct{}{},
	"else":                           struct{}{},
	"outer":                          struct{}{},
	"view":                           struct{}{},
	"end":                            struct{}{},
	"over":                           struct{}{},
	"waitfor":                        struct{}{},
	"errlvl":                         struct{}{},
	"percent":                        struct{}{},
	"when":                           struct{}{},
	"escape":                         struct{}{},
	"pivot":                          struct{}{},
	"where":                          struct{}{},
	"except":                         struct{}{},
	"plan":                           struct{}{},
	"while":                          struct{}{},
	"exec":                           struct{}{},
	"precision":                      struct{}{},
	"with":                           struct{}{},
	"execute":                        struct{}{},
	"primary":                        struct{}{},
	"exists":                         struct{}{},
	"print":                          struct{}{},
	"writetext":                      struct{}{},
	"exit":                           struct{}{},
	"proc":                           struct{}{},
}

// apparently 'within group' is also reserved but dropping that..
