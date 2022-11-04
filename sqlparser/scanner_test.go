package sqlparser

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

func TestNextToken(t *testing.T) {
	// just check that regexp should return nil if we didn't start to match...
	assert.Equal(t, []int(nil), numberRegexp.FindStringIndex("a123"))

	testExt := func(startOfLine bool, prefix, input string, expectedTokenType TokenType, expected string, extraAssertion ...func(s *Scanner)) func(*testing.T) {
		return func(t *testing.T) {
			s := &Scanner{input: prefix + input, curIndex: len(prefix), startOfLine: startOfLine}
			tt := s.NextToken()
			assert.Equal(t, expectedTokenType, tt)
			assert.Equal(t, expected, s.Token())
			for _, a := range extraAssertion {
				a(s)
			}
		}
	}

	test := func(input string, expectedTokenType TokenType, expected string, extraAssertion ...func(s *Scanner)) func(*testing.T) {
		return testExt(false, "abcd", input, expectedTokenType, expected, extraAssertion...)
	}

	t.Run("", test("    ", WhitespaceToken, "    "))
	t.Run("", test("     a   ", WhitespaceToken, "     "))
	t.Run("", test(" \t\t\n\n  \t \nasdf", WhitespaceToken, " \t\t\n\n  \t \n"))

	t.Run("", test("123", NumberToken, "123"))
	t.Run("", test("123;\n", NumberToken, "123"))
	t.Run("", test("123\n", NumberToken, "123"))
	t.Run("", test("123 ", NumberToken, "123"))
	t.Run("", test("+123.e-3_asdf", NumberToken, "+123.e-3"))
	t.Run("", test("-123.e+3+a", NumberToken, "-123.e+3"))
	t.Run("", test("-123.12e3+a", NumberToken, "-123.12e3"))
	t.Run("", test("-123.12e-35+a", NumberToken, "-123.12e-35"))
	t.Run("", test("-123.12ea", NumberToken, "-123.12e"))
	t.Run("", test("-123.12;\n", NumberToken, "-123.12"))

	t.Run("", test("'hello world'", VarcharLiteralToken, "'hello world'"))
	t.Run("", test("'hello world'after", VarcharLiteralToken, "'hello world'"))
	t.Run("", test("'hello '' world'after", VarcharLiteralToken, "'hello '' world'"))
	t.Run("", test("''''", VarcharLiteralToken, "''''"))
	t.Run("", test("''", VarcharLiteralToken, "''"))

	t.Run("", test("N'hello world'after", NVarcharLiteralToken, "N'hello world'"))
	t.Run("", test("N''", NVarcharLiteralToken, "N''"))

	t.Run("", test("'''hello", UnterminatedVarcharLiteralErrorToken, "'''hello"))
	t.Run("", test("N'''hello", UnterminatedVarcharLiteralErrorToken, "N'''hello"))

	t.Run("", test("[ quote \n quote]] hi]asdf", QuotedIdentifierToken, "[ quote \n quote]] hi]"))
	t.Run("", test("[][]", QuotedIdentifierToken, "[]"))
	t.Run("", test("[]]]", QuotedIdentifierToken, "[]]]"))
	t.Run("", test("[]]test", UnterminatedQuotedIdentifierErrorToken, "[]]test"))

	t.Run("", test("/* comment\n\n */asdf", MultilineCommentToken, "/* comment\n\n */"))
	t.Run("", test("/* comment\n\n ****/asdf", MultilineCommentToken, "/* comment\n\n ****/"))
	// unterminated multiline comment is treated like a comment
	t.Run("", test("/* comment\n\n asdf", MultilineCommentToken, "/* comment\n\n asdf"))

	// single stopLine comment .. trailing \n is not considered part of token
	t.Run("", test("-- test\nhello", SinglelineCommentToken, "-- test"))
	t.Run("", test("-- test", SinglelineCommentToken, "-- test"))

	t.Run("", test(`"asdf`, DoubleQuoteErrorToken, `"`))

	t.Run("", test(``, EOFToken, ``))

	t.Run("", test("abc", UnquotedIdentifierToken, "abc"))
	t.Run("", test("@a#$$__bc a", VariableIdentifierToken, "@a#$$__bc"))
	// identifier starting with N is special branch
	t.Run("", test("N@a#$$__bc a", UnquotedIdentifierToken, "N@a#$$__bc"))

	t.Run("", test("<select from", OtherToken, "<"))

	t.Run("", test("with,", ReservedWordToken, "with"))                 // branch 1
	t.Run("", test("nonclustered,", ReservedWordToken, "nonclustered")) // branch 1
	t.Run("", test("WItH,", ReservedWordToken, "WItH", func(s *Scanner) {
		assert.Equal(t, "with", s.ReservedWord())
	})) // branch 1
	t.Run("", test("NONclustered,", ReservedWordToken, "NONclustered", func(s *Scanner) {
		assert.Equal(t, "nonclustered", s.ReservedWord())
	})) // branch 2

	t.Run("", test("(", LeftParenToken, "("))
	t.Run("", test("(", LeftParenToken, "("))
	t.Run("", test(")", RightParenToken, ")"))
	t.Run("", test("))", RightParenToken, ")"))
	t.Run("", test(";", SemicolonToken, ";"))
	t.Run("", test(";;", SemicolonToken, ";"))
	t.Run("", test("=", EqualToken, "="))
	t.Run("", test("==", EqualToken, "="))
	t.Run("", test(",", CommaToken, ","))
	t.Run("", test(",,", CommaToken, ","))
	t.Run("", test(".", DotToken, "."))
	t.Run("", test("..", DotToken, "."))

	t.Run("", test("--sqlcode:include-if a\nnext line", PragmaToken, "--sqlcode:include-if a"))

	t.Run("", testExt(true, `\n  /*test*/`, "go asdfas", BatchSeparatorToken, `go`))

}

func TestBatchSeparator(t *testing.T) {
	// These rules are a bit simpler than what sqlcmd does;
	// we don't allow comments on the same line
	s := &Scanner{file: "test.sql", input: `this is
-- (0) Y
go

-- (1) N
prefix go

-- (2) Y+error
go trailer /*trailer2*/

-- (3) N
/*this is not a batch sep:*/ go

-- (4) N
not a batch sep: go

-- (5) N
this [is not
go
a batch sep]

-- (6) N
neither '
is this
go
a batch sep'                                            

-- (7) N
neither /* is
this a
go
batch sep */

-- (8) Y+error
go -- comment after
`}
	type Token struct {
		tokenType TokenType
		value     string
	}

	expected := []Token{
		{ // (0) Y
			tokenType: BatchSeparatorToken,
			value:     "go",
		},
		{ // (1) N
			tokenType: UnquotedIdentifierToken,
			value:     "go",
		},
		{ // (2) Y..
			tokenType: BatchSeparatorToken,
			value:     "go",
		},
		{ // (2) +error
			tokenType: MalformedBatchSeparatorToken,
			value:     "trailer",
		},
		{ // (2) +error
			tokenType: MalformedBatchSeparatorToken,
			value:     "/*trailer2*/",
		},
		{ // (3) N
			tokenType: UnquotedIdentifierToken,
			value:     "go",
		},
		{ // (4) N
			tokenType: UnquotedIdentifierToken,
			value:     "go",
		},
		{ // (5) N
			tokenType: QuotedIdentifierToken,
			value:     "[is not\ngo\na batch sep]",
		},
		{ // (6) N
			tokenType: VarcharLiteralToken,
			value:     "'\nis this\ngo\na batch sep'",
		},
		{ // (7) N
			tokenType: MultilineCommentToken,
			value:     "/* is\nthis a\ngo\nbatch sep */",
		},
		{ // (8) Y+...
			tokenType: BatchSeparatorToken,
			value:     "go",
		},
		{ // (8) +error
			tokenType: MalformedBatchSeparatorToken,
			value:     "-- comment after",
		},
	}

	// NOTE: Double quoted names also span 'go', but we don't support double
	// quotes in this parser

	var tokens []Token
	for {
		tt := s.NextToken()
		if tt == EOFToken {
			break
		}
		// keep all tokens containing 'go' for inspection..
		if strings.Contains(s.Token(), "go") || s.tokenType == MalformedBatchSeparatorToken {
			tokens = append(tokens, Token{tt, s.Token()})
		}
	}
	assert.Equal(t, expected, tokens)
}

func TestLineNumberAndColumn(t *testing.T) {
	s := &Scanner{file: "test.sql", input: `this is

-- a test, line 3

  declare @foo = 'of line numbers, this is line 5


and this is : '; /*line 8*/

      select [a name on line 10
  with newlines in it]   /*  11

multiline comment   line=13

*/  exec foo @line = 15
`}
	type typeAndLine struct {
		tokenType   TokenType
		start, stop Pos
		value       string
	}
	var tokens []typeAndLine
	for {
		tt := s.NextToken()
		if tt == EOFToken {
			break
		}
		tokens = append(tokens, typeAndLine{tt, s.Start(), s.Stop(), s.Token()})

	}
	// KEEP THIS COMMENT FOR GENERATING ASSERTION
	//for _, t := range tokens {
	//	fmt.Println(fmt.Sprintf("{%s, Pos{\"%s\", %d, %d}, Pos{\"%s\", %d, %d}, %s},", t.tokenType.GoString(), t.start.File, t.start.Line, t.start.Col, t.stop.File, t.stop.Line, t.stop.Col, repr.String(t.value)))
	//}
	require.Equal(t, []typeAndLine{
		{UnquotedIdentifierToken, Pos{"test.sql", 1, 1}, Pos{"test.sql", 1, 5}, "this"},
		{WhitespaceToken, Pos{"test.sql", 1, 5}, Pos{"test.sql", 1, 6}, " "},
		{ReservedWordToken, Pos{"test.sql", 1, 6}, Pos{"test.sql", 1, 8}, "is"},
		{WhitespaceToken, Pos{"test.sql", 1, 8}, Pos{"test.sql", 3, 1}, "\n\n"},
		{SinglelineCommentToken, Pos{"test.sql", 3, 1}, Pos{"test.sql", 3, 18}, "-- a test, line 3"},
		{WhitespaceToken, Pos{"test.sql", 3, 18}, Pos{"test.sql", 5, 3}, "\n\n  "},
		{ReservedWordToken, Pos{"test.sql", 5, 3}, Pos{"test.sql", 5, 10}, "declare"},
		{WhitespaceToken, Pos{"test.sql", 5, 10}, Pos{"test.sql", 5, 11}, " "},
		{VariableIdentifierToken, Pos{"test.sql", 5, 11}, Pos{"test.sql", 5, 15}, "@foo"},
		{WhitespaceToken, Pos{"test.sql", 5, 15}, Pos{"test.sql", 5, 16}, " "},
		{EqualToken, Pos{"test.sql", 5, 16}, Pos{"test.sql", 5, 17}, "="},
		{WhitespaceToken, Pos{"test.sql", 5, 17}, Pos{"test.sql", 5, 18}, " "},
		{VarcharLiteralToken, Pos{"test.sql", 5, 18}, Pos{"test.sql", 8, 16}, "'of line numbers, this is line 5\n\n\nand this is : '"},
		{SemicolonToken, Pos{"test.sql", 8, 16}, Pos{"test.sql", 8, 17}, ";"},
		{WhitespaceToken, Pos{"test.sql", 8, 17}, Pos{"test.sql", 8, 18}, " "},
		{MultilineCommentToken, Pos{"test.sql", 8, 18}, Pos{"test.sql", 8, 28}, "/*line 8*/"},
		{WhitespaceToken, Pos{"test.sql", 8, 28}, Pos{"test.sql", 10, 7}, "\n\n      "},
		{ReservedWordToken, Pos{"test.sql", 10, 7}, Pos{"test.sql", 10, 13}, "select"},
		{WhitespaceToken, Pos{"test.sql", 10, 13}, Pos{"test.sql", 10, 14}, " "},
		{QuotedIdentifierToken, Pos{"test.sql", 10, 14}, Pos{"test.sql", 11, 23}, "[a name on line 10\n  with newlines in it]"},
		{WhitespaceToken, Pos{"test.sql", 11, 23}, Pos{"test.sql", 11, 26}, "   "},
		{MultilineCommentToken, Pos{"test.sql", 11, 26}, Pos{"test.sql", 15, 3}, "/*  11\n\nmultiline comment   line=13\n\n*/"},
		{WhitespaceToken, Pos{"test.sql", 15, 3}, Pos{"test.sql", 15, 5}, "  "},
		{ReservedWordToken, Pos{"test.sql", 15, 5}, Pos{"test.sql", 15, 9}, "exec"},
		{WhitespaceToken, Pos{"test.sql", 15, 9}, Pos{"test.sql", 15, 10}, " "},
		{UnquotedIdentifierToken, Pos{"test.sql", 15, 10}, Pos{"test.sql", 15, 13}, "foo"},
		{WhitespaceToken, Pos{"test.sql", 15, 13}, Pos{"test.sql", 15, 14}, " "},
		{VariableIdentifierToken, Pos{"test.sql", 15, 14}, Pos{"test.sql", 15, 19}, "@line"},
		{WhitespaceToken, Pos{"test.sql", 15, 19}, Pos{"test.sql", 15, 20}, " "},
		{EqualToken, Pos{"test.sql", 15, 20}, Pos{"test.sql", 15, 21}, "="},
		{WhitespaceToken, Pos{"test.sql", 15, 21}, Pos{"test.sql", 15, 22}, " "},
		{NumberToken, Pos{"test.sql", 15, 22}, Pos{"test.sql", 15, 24}, "15"},
		{WhitespaceToken, Pos{"test.sql", 15, 24}, Pos{"test.sql", 16, 1}, "\n"},
	}, tokens)
}
