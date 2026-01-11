package mssql

import (
	"testing"

	"github.com/vippsas/sqlcode/v2/sqlparser/sqldocument"
)

// Helper to collect all tokens from input
func collectTokens(input string) []struct {
	Type  sqldocument.TokenType
	Value string
} {
	s := NewScanner("test.sql", input)
	var tokens []struct {
		Type  sqldocument.TokenType
		Value string
	}
	for {
		tt := s.NextToken()
		tokens = append(tokens, struct {
			Type  sqldocument.TokenType
			Value string
		}{tt, s.Token()})
		if tt == sqldocument.EOFToken {
			break
		}
	}
	return tokens
}

func TestScanner_SimpleTokens(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []sqldocument.TokenType
	}{
		{
			name:  "parentheses and punctuation",
			input: "( ) ; = , .",
			expected: []sqldocument.TokenType{
				sqldocument.LeftParenToken,
				sqldocument.WhitespaceToken,
				sqldocument.RightParenToken,
				sqldocument.WhitespaceToken,
				sqldocument.SemicolonToken,
				sqldocument.WhitespaceToken,
				sqldocument.EqualToken,
				sqldocument.WhitespaceToken,
				sqldocument.CommaToken,
				sqldocument.WhitespaceToken,
				sqldocument.DotToken,
				sqldocument.EOFToken,
			},
		},
		{
			name:  "empty input",
			input: "",
			expected: []sqldocument.TokenType{
				sqldocument.EOFToken,
			},
		},
		{
			name:  "whitespace only",
			input: "   \t\n  ",
			expected: []sqldocument.TokenType{
				sqldocument.WhitespaceToken,
				sqldocument.EOFToken,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens := collectTokens(tt.input)
			if len(tokens) != len(tt.expected) {
				t.Fatalf("expected %d tokens, got %d", len(tt.expected), len(tokens))
			}
			for i, exp := range tt.expected {
				if tokens[i].Type != exp {
					t.Errorf("token %d: expected type %v, got %v (value: %q)",
						i, exp, tokens[i].Type, tokens[i].Value)
				}
			}
		})
	}
}

func TestScanner_StringLiterals(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedType  sqldocument.TokenType
		expectedValue string
	}{
		{
			name:          "simple varchar",
			input:         "'hello world'",
			expectedType:  VarcharLiteralToken,
			expectedValue: "'hello world'",
		},
		{
			name:          "varchar with escaped quote",
			input:         "'it''s working'",
			expectedType:  VarcharLiteralToken,
			expectedValue: "'it''s working'",
		},
		{
			name:          "empty varchar",
			input:         "''",
			expectedType:  VarcharLiteralToken,
			expectedValue: "''",
		},
		{
			name:          "simple nvarchar",
			input:         "N'unicode string'",
			expectedType:  NVarcharLiteralToken,
			expectedValue: "N'unicode string'",
		},
		{
			name:          "nvarchar with escaped quote",
			input:         "N'say ''hello'''",
			expectedType:  NVarcharLiteralToken,
			expectedValue: "N'say ''hello'''",
		},
		{
			name:          "multiline varchar",
			input:         "'line1\nline2\nline3'",
			expectedType:  VarcharLiteralToken,
			expectedValue: "'line1\nline2\nline3'",
		},
		{
			name:          "unterminated varchar",
			input:         "'unterminated",
			expectedType:  UnterminatedVarcharLiteralErrorToken,
			expectedValue: "'unterminated",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewScanner("test.sql", tt.input)
			s.NextToken()
			if s.TokenType() != tt.expectedType {
				t.Errorf("expected type %v, got %v", tt.expectedType, s.TokenType())
			}
			if s.Token() != tt.expectedValue {
				t.Errorf("expected value %q, got %q", tt.expectedValue, s.Token())
			}
		})
	}
}

func TestScanner_QuotedIdentifiers(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedType  sqldocument.TokenType
		expectedValue string
	}{
		{
			name:          "simple bracket identifier",
			input:         "[MyTable]",
			expectedType:  sqldocument.QuotedIdentifierToken,
			expectedValue: "[MyTable]",
		},
		{
			name:          "bracket identifier with space",
			input:         "[My Table Name]",
			expectedType:  sqldocument.QuotedIdentifierToken,
			expectedValue: "[My Table Name]",
		},
		{
			name:          "bracket identifier with escaped bracket",
			input:         "[My]]Table]",
			expectedType:  sqldocument.QuotedIdentifierToken,
			expectedValue: "[My]]Table]",
		},
		{
			name:          "code schema identifier",
			input:         "[code]",
			expectedType:  sqldocument.QuotedIdentifierToken,
			expectedValue: "[code]",
		},
		{
			name:          "unterminated bracket identifier",
			input:         "[unterminated",
			expectedType:  UnterminatedQuotedIdentifierErrorToken,
			expectedValue: "[unterminated",
		},
		{
			name:          "double quote error",
			input:         "\"identifier\"",
			expectedType:  DoubleQuoteErrorToken,
			expectedValue: "\"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewScanner("test.sql", tt.input)
			s.NextToken()
			if s.TokenType() != tt.expectedType {
				t.Errorf("expected type %v, got %v", tt.expectedType, s.TokenType())
			}
			if s.Token() != tt.expectedValue {
				t.Errorf("expected value %q, got %q", tt.expectedValue, s.Token())
			}
		})
	}
}

func TestScanner_Identifiers(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		expectedType sqldocument.TokenType
		expectedWord string // for reserved words
	}{
		{
			name:         "simple identifier",
			input:        "MyProc",
			expectedType: sqldocument.UnquotedIdentifierToken,
		},
		{
			name:         "identifier with underscore",
			input:        "my_procedure_name",
			expectedType: sqldocument.UnquotedIdentifierToken,
		},
		{
			name:         "identifier with numbers",
			input:        "Proc123",
			expectedType: sqldocument.UnquotedIdentifierToken,
		},
		{
			name:         "identifier starting with underscore",
			input:        "_private",
			expectedType: sqldocument.UnquotedIdentifierToken,
		},
		{
			name:         "identifier with hash (temp table)",
			input:        "#TempTable",
			expectedType: sqldocument.UnquotedIdentifierToken,
		},
		{
			name:         "global temp table",
			input:        "##GlobalTemp",
			expectedType: sqldocument.UnquotedIdentifierToken,
		},
		{
			name:         "variable identifier",
			input:        "@myVariable",
			expectedType: sqldocument.VariableIdentifierToken,
		},
		{
			name:         "system variable",
			input:        "@@ROWCOUNT",
			expectedType: sqldocument.VariableIdentifierToken,
		},
		{
			name:         "reserved word CREATE",
			input:        "CREATE",
			expectedType: sqldocument.ReservedWordToken,
			expectedWord: "create",
		},
		{
			name:         "reserved word lowercase",
			input:        "select",
			expectedType: sqldocument.ReservedWordToken,
			expectedWord: "select",
		},
		{
			name:         "reserved word mixed case",
			input:        "DeClaRe",
			expectedType: sqldocument.ReservedWordToken,
			expectedWord: "declare",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewScanner("test.sql", tt.input)
			s.NextToken()
			if s.TokenType() != tt.expectedType {
				t.Errorf("expected type %v, got %v", tt.expectedType, s.TokenType())
			}
			if tt.expectedWord != "" && s.ReservedWord() != tt.expectedWord {
				t.Errorf("expected reserved word %q, got %q", tt.expectedWord, s.ReservedWord())
			}
		})
	}
}

func TestScanner_Numbers(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedValue string
	}{
		{"integer", "42", "42"},
		{"negative integer", "-42", "-42"},
		{"positive integer", "+42", "+42"},
		{"decimal", "3.14159", "3.14159"},
		{"negative decimal", "-3.14", "-3.14"},
		{"scientific notation", "1.5e10", "1.5e10"},
		{"scientific negative exponent", "1.5e-10", "1.5e-10"},
		{"scientific positive exponent", "1.5e+10", "1.5e+10"},
		{"integer scientific", "1e5", "1e5"},
		{"leading decimal", "123.", "123."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewScanner("test.sql", tt.input)
			s.NextToken()
			if s.TokenType() != sqldocument.NumberToken {
				t.Errorf("expected NumberToken, got %v", s.TokenType())
			}
			if s.Token() != tt.expectedValue {
				t.Errorf("expected %q, got %q", tt.expectedValue, s.Token())
			}
		})
	}
}

func TestScanner_Comments(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedType  sqldocument.TokenType
		expectedValue string
	}{
		{
			name:          "single line comment",
			input:         "-- this is a comment",
			expectedType:  sqldocument.SinglelineCommentToken,
			expectedValue: "-- this is a comment",
		},
		{
			name:          "single line comment before newline",
			input:         "-- comment\ncode",
			expectedType:  sqldocument.SinglelineCommentToken,
			expectedValue: "-- comment",
		},
		{
			name:          "multiline comment",
			input:         "/* this is\na multiline\ncomment */",
			expectedType:  sqldocument.MultilineCommentToken,
			expectedValue: "/* this is\na multiline\ncomment */",
		},
		{
			name:          "multiline comment with asterisks",
			input:         "/* * * * */",
			expectedType:  sqldocument.MultilineCommentToken,
			expectedValue: "/* * * * */",
		},
		{
			name:          "pragma comment",
			input:         "--sqlcode:include-if foo",
			expectedType:  sqldocument.PragmaToken,
			expectedValue: "--sqlcode:include-if foo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewScanner("test.sql", tt.input)
			s.NextToken()
			if s.TokenType() != tt.expectedType {
				t.Errorf("expected type %v, got %v", tt.expectedType, s.TokenType())
			}
			if s.Token() != tt.expectedValue {
				t.Errorf("expected value %q, got %q", tt.expectedValue, s.Token())
			}
		})
	}
}

func TestScanner_Position(t *testing.T) {
	input := "SELECT\n  @var\n  FROM"
	s := NewScanner("test.sql", input)

	// SELECT
	s.NextToken()
	start := s.Start()
	if start.Line != 1 || start.Col != 1 {
		t.Errorf("SELECT start: expected (1,1), got (%d,%d)", start.Line, start.Col)
	}

	// whitespace (includes newline)
	s.NextToken()

	// @var
	s.NextToken()
	start = s.Start()
	if start.Line != 2 || start.Col != 3 {
		t.Errorf("@var start: expected (2,3), got (%d,%d)", start.Line, start.Col)
	}

	// whitespace
	s.NextToken()

	// FROM
	s.NextToken()
	start = s.Start()
	if start.Line != 3 || start.Col != 3 {
		t.Errorf("FROM start: expected (3,3), got (%d,%d)", start.Line, start.Col)
	}
}

func TestScanner_ComplexStatement(t *testing.T) {
	input := `CREATE PROCEDURE [code].[MyProc]
    @Param1 nvarchar(100),
    @Param2 int = 42
AS
BEGIN
    SELECT @Param1, @Param2
END`

	s := NewScanner("test.sql", input)

	// Verify we can tokenize the entire statement without errors
	tokenCount := 0
	for {
		tt := s.NextToken()
		tokenCount++
		if tt == sqldocument.EOFToken {
			break
		}
		if tt == sqldocument.NonUTF8ErrorToken {
			t.Fatalf("unexpected non-UTF8 error at token %d", tokenCount)
		}
	}

	if tokenCount < 30 {
		t.Errorf("expected at least 30 tokens, got %d", tokenCount)
	}
}

func TestScanner_Clone(t *testing.T) {
	input := "SELECT FROM WHERE"
	s := NewScanner("test.sql", input)

	s.NextToken() // SELECT
	s.NextToken() // whitespace

	clone := s.Clone()

	// Advance original
	s.NextToken() // FROM

	// Clone should still be at whitespace position
	if clone.Token() != " " {
		t.Errorf("clone should still be at whitespace, got %q", clone.Token())
	}

	// Advance clone independently
	clone.NextToken()
	if clone.Token() != "FROM" {
		t.Errorf("clone should now be at FROM, got %q", clone.Token())
	}
}

func TestScanner_SkipMethods(t *testing.T) {
	input := "SELECT  /* comment */  @var"
	s := NewScanner("test.sql", input)

	s.NextToken() // SELECT
	s.NextToken() // whitespace

	// SkipWhitespace should stop at comment
	s.SkipWhitespace()
	if s.TokenType() != sqldocument.MultilineCommentToken {
		t.Errorf("SkipWhitespace should stop at comment, got %v", s.TokenType())
	}

	// Reset and test SkipWhitespaceComments
	s = NewScanner("test.sql", input)
	s.NextToken() // SELECT
	tt := s.NextNonWhitespaceCommentToken()
	if tt != sqldocument.VariableIdentifierToken {
		t.Errorf("NextNonWhitespaceCommentToken should return @var token type, got %v", tt)
	}
	if s.Token() != "@var" {
		t.Errorf("should be at @var, got %q", s.Token())
	}
}
