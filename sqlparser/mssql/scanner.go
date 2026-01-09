package mssql

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/smasher164/xid"
	"github.com/vippsas/sqlcode/sqlparser/sqldocument"
)

// Scanner is a lexical scanner for T-SQL source code.
//
// Unlike traditional lexer/parser architectures with a token stream, Scanner
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
type Scanner struct {
	input string              // The complete source code being scanned
	file  sqldocument.FileRef // Reference to the source file for error reporting

	startIndex int                   // Byte index where current token starts
	curIndex   int                   // Current byte position in Input
	tokenType  sqldocument.TokenType // Type of the current token

	// Batch separator state machine fields.
	// The GO batch separator has special rules: it must appear at the start
	// of a line and nothing except whitespace can follow it on the same line.
	startOfLine         bool // True if no non-whitespace/comment seen since start of line
	afterBatchSeparator bool // True if we just saw GO; used to detect malformed separators

	startLine        int // Line number (0-indexed) where current token starts
	stopLine         int // Line number (0-indexed) where current token ends
	indexAtStartLine int // Byte index at the start of startLine (after newline)
	indexAtStopLine  int // Byte index at the start of stopLine (after newline)

	reservedWord string // Lowercase version of token if it's a reserved word, empty otherwise
}

// NewScanner creates a new Scanner for the given T-SQL source file and input string.
// The scanner is positioned before the first token; call NextToken() to advance.
func NewScanner(path sqldocument.FileRef, input string) *Scanner {
	return &Scanner{input: input, file: path}
}

// TokenType returns the type of the current token.
func (s *Scanner) TokenType() sqldocument.TokenType {
	return s.tokenType
}

func (s *Scanner) SetInput(input []byte) {
	s.input = string(input)
}

func (s *Scanner) SetFile(file sqldocument.FileRef) {
	s.file = file
}

func (s *Scanner) File() sqldocument.FileRef {
	return s.file
}

// Clone returns a copy of the scanner at its current position.
// This is used for look-ahead parsing where we need to tentatively
// scan tokens without committing to consuming them.
func (s Scanner) Clone() *Scanner {
	result := new(Scanner)
	*result = s
	return result
}

// Token returns the text of the current token as a substring of Input.
func (s *Scanner) Token() string {
	return s.input[s.startIndex:s.curIndex]
}

// TokenLower returns the current token text converted to lowercase.
// Useful for case-insensitive keyword matching.
func (s *Scanner) TokenLower() string {
	return strings.ToLower(s.Token())
}

// ReservedWord returns the lowercase reserved word if the current token
// is a ReservedWordToken, or an empty string otherwise.
func (s *Scanner) ReservedWord() string {
	return s.reservedWord
}

// Start returns the position where the current token begins.
// Line and column are 1-indexed.
func (s *Scanner) Start() sqldocument.Pos {
	return sqldocument.Pos{
		Line: s.startLine + 1,
		Col:  s.startIndex - s.indexAtStartLine + 1,
		File: s.file,
	}
}

// Stop returns the position where the current token ends.
// Line and column are 1-indexed.
func (s *Scanner) Stop() sqldocument.Pos {
	return sqldocument.Pos{
		Line: s.stopLine + 1,
		Col:  s.curIndex - s.indexAtStopLine + 1,
		File: s.file,
	}
}

// bumpLine increments the line counter and records the byte position
// after the newline character. The offset parameter is the position
// of the newline within the current scan operation.
func (s *Scanner) bumpLine(offset int) {
	s.stopLine++
	s.indexAtStopLine = s.curIndex + offset + 1
}

// SkipWhitespaceComments advances past any whitespace and comment tokens.
// Stops when a non-whitespace, non-comment token is encountered.
func (s *Scanner) SkipWhitespaceComments() {
	for {
		switch s.TokenType() {
		case sqldocument.WhitespaceToken, sqldocument.MultilineCommentToken, sqldocument.SinglelineCommentToken:
		default:
			return
		}
		s.NextToken()
	}
}

// SkipWhitespace advances past any whitespace tokens.
// Stops when a non-whitespace token is encountered.
// Unlike SkipWhitespaceComments, this preserves comments.
func (s *Scanner) SkipWhitespace() {
	for {
		switch s.TokenType() {
		case sqldocument.WhitespaceToken:
		default:
			return
		}
		s.NextToken()
	}
}

// NextNonWhitespaceToken advances to the next token and then skips
// any whitespace, returning the type of the first non-whitespace token.
func (s *Scanner) NextNonWhitespaceToken() sqldocument.TokenType {
	s.NextToken()
	s.SkipWhitespace()
	return s.TokenType()
}

// NextNonWhitespaceCommentToken advances to the next token and then skips
// any whitespace and comments, returning the type of the first significant token.
func (s *Scanner) NextNonWhitespaceCommentToken() sqldocument.TokenType {
	s.NextToken()
	s.SkipWhitespaceComments()
	return s.TokenType()
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
func (s *Scanner) NextToken() sqldocument.TokenType {
	// handle startOfLine flag here; this is used to parse the 'go' batch separator
	s.tokenType = s.nextToken()

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

	if s.startOfLine && s.tokenType == sqldocument.UnquotedIdentifierToken && s.TokenLower() == "go" {
		s.tokenType = sqldocument.BatchSeparatorToken
		s.afterBatchSeparator = true
	} else if s.afterBatchSeparator && s.tokenType != sqldocument.WhitespaceToken && s.tokenType != sqldocument.EOFToken {
		s.tokenType = sqldocument.MalformedBatchSeparatorToken
	} else if s.tokenType == sqldocument.WhitespaceToken {
		// If we just saw the whitespace token that bumped the linecount,
		// we are at the "start of line", even if this contains some space after the \n:
		if s.stopLine > s.startLine {
			s.startOfLine = true
			s.afterBatchSeparator = false
		}
		// Also, don't change the state if we did not bump the line
	} else {
		s.startOfLine = false
	}

	return s.tokenType
}

func (s *Scanner) nextToken() sqldocument.TokenType {
	s.startIndex = s.curIndex
	s.reservedWord = ""
	s.startLine = s.stopLine
	s.indexAtStartLine = s.indexAtStopLine
	r, w := utf8.DecodeRuneInString(s.input[s.curIndex:])

	// First, decisions that can be made after one character:
	switch {
	case r == utf8.RuneError && w == 0:
		return sqldocument.EOFToken
	case r == utf8.RuneError && w == -1:
		// not UTF-8, we can't really proceed so not advancing Scanner,
		// caller should take care to always exit..
		return sqldocument.NonUTF8ErrorToken
	case r == '(':
		s.curIndex += w
		return sqldocument.LeftParenToken
	case r == ')':
		s.curIndex += w
		return sqldocument.RightParenToken
	case r == ';':
		s.curIndex += w
		return sqldocument.SemicolonToken
	case r == '=':
		s.curIndex += w
		return sqldocument.EqualToken
	case r == ',':
		s.curIndex += w
		return sqldocument.CommaToken
	case r == '.':
		s.curIndex += w
		return sqldocument.DotToken
	case r == '\'':
		s.curIndex += w
		return s.scanStringLiteral(VarcharLiteralToken)
	case r >= '0' && r <= '9':
		return s.scanNumber()
	case r == '[':
		s.curIndex += w
		return s.scanQuotedIdentifier()
	case r == '"':
		// parser don't support double-quoted literals, just return an error token
		s.curIndex += w
		t := DoubleQuoteErrorToken
		return t
	case unicode.IsSpace(r):
		// do not advance s.curIndex here, simpler to do it all in scanWhitespace(); in case r == '\n' we need stopLine number bump
		return s.scanWhitespace()
	case r != 'N' && (xid.Start(r) || r == '@' || r == '_' || r == '＿' || r == '#'): // Unicode Start identifier
		// good guide for identifiers:
		// https://sqlquantumleap.com/reference/completely-complete-list-of-rules-for-t-sql-identifiers/
		s.curIndex += w
		s.scanIdentifier()
		if r == '@' {
			return sqldocument.VariableIdentifierToken
		} else {
			rw := strings.ToLower(s.Token())
			_, ok := reservedWords[rw]
			if ok {
				s.reservedWord = rw
				return sqldocument.ReservedWordToken
			} else {
				return sqldocument.UnquotedIdentifierToken
			}
		}
	}

	// OK, we need to peek 1 character to make a decision
	r2, w2 := utf8.DecodeRuneInString(s.input[s.curIndex+w:])

	switch {
	case r == 'N':
		s.curIndex += w
		if r2 == '\'' { // N'varchar string' .. only upper-case N allowed
			s.curIndex += w2
			return s.scanStringLiteral(NVarcharLiteralToken)
		}
		// no, it is instead an identifier starting with N...
		s.scanIdentifier()
		rw := strings.ToLower(s.Token())
		_, ok := reservedWords[rw]
		if ok {
			s.reservedWord = rw
			return sqldocument.ReservedWordToken
		} else {
			return sqldocument.UnquotedIdentifierToken
		}
	case r == '/' && r2 == '*':
		s.curIndex += w + w2
		return s.scanMultilineComment()
	case r == '-' && r2 == '-':
		s.curIndex += w + w2
		return s.scanSinglelineComment()
	case (r == '-' || r == '+') && (r2 >= '0' && r2 <= '9'):
		return s.scanNumber()
	}

	s.curIndex += w
	return sqldocument.OtherToken
}

// scanMultilineComment assumes one has advanced over '/*'
func (s *Scanner) scanMultilineComment() sqldocument.TokenType {
	prevWasStar := false
	for i, r := range s.input[s.curIndex:] {
		if r == '*' {
			prevWasStar = true
		} else if prevWasStar && r == '/' {
			s.curIndex += i + 1
			return sqldocument.MultilineCommentToken
		} else if r == '\n' {
			s.bumpLine(i)
		}
	}
	s.curIndex = len(s.input)
	return sqldocument.MultilineCommentToken
}

// scanSinglelineComment assumes one has advanced over --
func (s *Scanner) scanSinglelineComment() sqldocument.TokenType {
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
		return sqldocument.PragmaToken
	} else {
		return sqldocument.SinglelineCommentToken
	}
}

// scanStringLiteral assumes one has scanned ' or N' (depending on param);
// then scans until the end of the string
func (s *Scanner) scanStringLiteral(tokenType sqldocument.TokenType) sqldocument.TokenType {
	return s.scanUntilSingleDoubleEscapes('\'', tokenType, UnterminatedVarcharLiteralErrorToken)
}

func (s *Scanner) scanQuotedIdentifier() sqldocument.TokenType {
	return s.scanUntilSingleDoubleEscapes(']', sqldocument.QuotedIdentifierToken, UnterminatedQuotedIdentifierErrorToken)
}

// scanIdentifier assumes first character of an identifier has been identified,
// and scans to the end
func (s *Scanner) scanIdentifier() {
	for i, r := range s.input[s.curIndex:] {
		if !(xid.Continue(r) || r == '$' || r == '#' || r == '@' || unicode.Is(unicode.Cf, r)) {
			s.curIndex += i
			return
		}
	}
	s.curIndex = len(s.input)
}

// DRY helper to handle both ” and ]] escapes
func (s *Scanner) scanUntilSingleDoubleEscapes(
	endmarker rune,
	tokenType sqldocument.TokenType,
	unterminatedTokenType sqldocument.TokenType,
) sqldocument.TokenType {
	skipnext := false
	for i, r := range s.input[s.curIndex:] {
		if skipnext {
			skipnext = false
			continue
		}
		if r == '\n' {
			s.bumpLine(i)
		}
		if r == endmarker {
			r2, _ := utf8.DecodeRuneInString(s.input[s.curIndex+i+1:]) // r2 may be RuneError if eof
			if r2 == endmarker {
				// we have a double endmarker; this is used as escape
				skipnext = true
			} else {
				// terminating '
				s.curIndex += i + 1
				return tokenType
			}
		}
	}
	s.curIndex = len(s.input)
	return unterminatedTokenType
}

var numberRegexp = regexp.MustCompile(`^[+-]?\d+\.?\d*([eE][+-]?\d*)?`)

func (s *Scanner) scanNumber() sqldocument.TokenType {
	// T-SQL seems to scan a number until the
	// end and then allowing a literal to start without whitespace or other things
	// in between...

	// https://docs.microsoft.com/en-us/sql/odbc/reference/appendixes/numeric-literal-syntax?view=sql-server-ver16

	loc := numberRegexp.FindStringIndex(s.input[s.curIndex:])
	if len(loc) == 0 {
		panic("should always have a match according to regex and conditions in caller")
	}
	s.curIndex += loc[1]
	return sqldocument.NumberToken
}

func (s *Scanner) scanWhitespace() sqldocument.TokenType {
	for i, r := range s.input[s.curIndex:] {
		if r == '\n' {
			s.bumpLine(i)
		}
		if !unicode.IsSpace(r) {
			s.curIndex += i
			return sqldocument.WhitespaceToken
		}
	}
	// eof
	s.curIndex = len(s.input)
	return sqldocument.WhitespaceToken
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
