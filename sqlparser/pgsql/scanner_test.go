package pgsql

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vippsas/sqlcode/v2/sqlparser/sqldocument"
)

func TestScanner_BasicTokens(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected sqldocument.TokenType
		token    string
	}{
		{"left paren", "(", sqldocument.LeftParenToken, "("},
		{"right paren", ")", sqldocument.RightParenToken, ")"},
		{"semicolon", ";", sqldocument.SemicolonToken, ";"},
		{"equal", "=", sqldocument.EqualToken, "="},
		{"comma", ",", sqldocument.CommaToken, ","},
		{"dot", ".", sqldocument.DotToken, "."},
		{"EOF", "", sqldocument.EOFToken, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewScanner("test.sql", tt.input)
			tokenType := s.NextToken()
			assert.Equal(t, tt.expected, tokenType)
			assert.Equal(t, tt.token, s.Token())
		})
	}
}

func TestScanner_Whitespace(t *testing.T) {
	s := NewScanner("test.sql", "   \n\t  ")
	tokenType := s.NextToken()
	assert.Equal(t, sqldocument.WhitespaceToken, tokenType)
}

func TestScanner_SingleLineComment(t *testing.T) {
	s := NewScanner("test.sql", "-- this is a comment\nSELECT")

	tokenType := s.NextToken()
	assert.Equal(t, sqldocument.SinglelineCommentToken, tokenType)
	assert.Equal(t, "-- this is a comment", s.Token())

	s.NextToken() // whitespace
	tokenType = s.NextToken()
	assert.Equal(t, sqldocument.ReservedWordToken, tokenType)
	assert.Equal(t, "SELECT", s.Token())
}

func TestScanner_MultiLineComment(t *testing.T) {
	s := NewScanner("test.sql", "/* multi\nline\ncomment */SELECT")

	tokenType := s.NextToken()
	assert.Equal(t, sqldocument.MultilineCommentToken, tokenType)
	assert.Equal(t, "/* multi\nline\ncomment */", s.Token())

	tokenType = s.NextToken()
	assert.Equal(t, sqldocument.ReservedWordToken, tokenType)
}

func TestScanner_StringLiteral(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple string", "'hello'", "'hello'"},
		{"escaped quote", "'it''s'", "'it''s'"},
		{"empty string", "''", "''"},
		{"multiline string", "'line1\nline2'", "'line1\nline2'"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewScanner("test.sql", tt.input)
			tokenType := s.NextToken()
			assert.Equal(t, StringLiteralToken, tokenType)
			assert.Equal(t, tt.expected, s.Token())
		})
	}
}

func TestScanner_EscapeStringLiteral(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple escape string", "E'hello'", "E'hello'"},
		{"backslash escape", "E'it\\'s'", "E'it\\'s'"},
		{"newline escape", "E'line1\\nline2'", "E'line1\\nline2'"},
		{"lowercase e", "e'hello'", "e'hello'"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewScanner("test.sql", tt.input)
			tokenType := s.NextToken()
			assert.Equal(t, StringLiteralToken, tokenType)
			assert.Equal(t, tt.expected, s.Token())
		})
	}
}

func TestScanner_DollarQuotedString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple dollar quote", "$$hello$$", "$$hello$$"},
		{"tagged dollar quote", "$body$hello$body$", "$body$hello$body$"},
		{"multiline dollar quote", "$$line1\nline2$$", "$$line1\nline2$$"},
		{"nested quotes in dollar", "$$it's a 'test'$$", "$$it's a 'test'$$"},
		{"function body", "$func$\nBEGIN\n  RETURN 1;\nEND;\n$func$", "$func$\nBEGIN\n  RETURN 1;\nEND;\n$func$"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewScanner("test.sql", tt.input)
			tokenType := s.NextToken()
			assert.Equal(t, DollarQuotedStringToken, tokenType)
			assert.Equal(t, tt.expected, s.Token())
		})
	}
}

func TestScanner_PositionalParameter(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"param 1", "$1", "$1"},
		{"param 10", "$10", "$10"},
		{"param 123", "$123", "$123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewScanner("test.sql", tt.input)
			tokenType := s.NextToken()
			assert.Equal(t, PositionalParameterToken, tokenType)
			assert.Equal(t, tt.expected, s.Token())
		})
	}
}

func TestScanner_QuotedIdentifier(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple quoted", "\"MyTable\"", "\"MyTable\""},
		{"escaped quote", "\"My\"\"Table\"", "\"My\"\"Table\""},
		{"with spaces", "\"My Table\"", "\"My Table\""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewScanner("test.sql", tt.input)
			tokenType := s.NextToken()
			assert.Equal(t, sqldocument.QuotedIdentifierToken, tokenType)
			assert.Equal(t, tt.expected, s.Token())
		})
	}
}

func TestScanner_BitStringLiteral(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"bit string", "B'101010'", "B'101010'"},
		{"lowercase b", "b'1100'", "b'1100'"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewScanner("test.sql", tt.input)
			tokenType := s.NextToken()
			assert.Equal(t, BitStringLiteralToken, tokenType)
			assert.Equal(t, tt.expected, s.Token())
		})
	}
}

func TestScanner_HexStringLiteral(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"hex string", "X'1A2B'", "X'1A2B'"},
		{"lowercase x", "x'deadbeef'", "x'deadbeef'"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewScanner("test.sql", tt.input)
			tokenType := s.NextToken()
			assert.Equal(t, HexStringLiteralToken, tokenType)
			assert.Equal(t, tt.expected, s.Token())
		})
	}
}

func TestScanner_Number(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"integer", "123", "123"},
		{"decimal", "123.456", "123.456"},
		{"negative", "-123", "-123"},
		{"positive", "+123", "+123"},
		{"scientific", "1.23e10", "1.23e10"},
		{"scientific negative exp", "1.23e-10", "1.23e-10"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewScanner("test.sql", tt.input)
			tokenType := s.NextToken()
			assert.Equal(t, sqldocument.NumberToken, tokenType)
			assert.Equal(t, tt.expected, s.Token())
		})
	}
}

func TestScanner_ReservedWords(t *testing.T) {
	reservedWordTests := []string{
		"SELECT", "FROM", "WHERE", "CREATE",
		"END", "AS", "AND", "OR", "NOT", "NULL",
		"TABLE", "JOIN", "LEFT", "RIGHT",
		"INNER", "OUTER", "ON", "IN", "IS", "CASE", "WHEN", "THEN", "ELSE",
	}

	for _, word := range reservedWordTests {
		t.Run(word, func(t *testing.T) {
			s := NewScanner("test.sql", word)
			tokenType := s.NextToken()
			assert.Equal(t, sqldocument.ReservedWordToken, tokenType)
			assert.Equal(t, word, s.Token())
		})
	}
}

func TestScanner_Identifier(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple identifier", "my_table", "my_table"},
		{"with numbers", "table1", "table1"},
		{"underscore start", "_private", "_private"},
		{"mixed case", "MyTable", "MyTable"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewScanner("test.sql", tt.input)
			tokenType := s.NextToken()
			assert.Equal(t, sqldocument.UnquotedIdentifierToken, tokenType)
			assert.Equal(t, tt.expected, s.Token())
		})
	}
}

func TestScanner_TypeCastOperator(t *testing.T) {
	s := NewScanner("test.sql", "value::integer")

	tokenType := s.NextToken()
	assert.Equal(t, sqldocument.UnquotedIdentifierToken, tokenType)
	assert.Equal(t, "value", s.Token())

	tokenType = s.NextToken()
	assert.Equal(t, sqldocument.OtherToken, tokenType)
	assert.Equal(t, "::", s.Token())

	tokenType = s.NextToken()
	assert.Equal(t, sqldocument.UnquotedIdentifierToken, tokenType)
	assert.Equal(t, "integer", s.Token())
}

func TestScanner_CreateFunction(t *testing.T) {
	input := `CREATE FUNCTION add_numbers(a INTEGER, b INTEGER)
RETURNS INTEGER
LANGUAGE plpgsql
AS $$
BEGIN
    RETURN a + b;
END;
$$;`

	s := NewScanner("test.sql", input)

	tokens := []struct {
		tokenType sqldocument.TokenType
		token     string
	}{
		{sqldocument.ReservedWordToken, "CREATE"},
		{sqldocument.WhitespaceToken, " "},
		{sqldocument.UnquotedIdentifierToken, "FUNCTION"},
		{sqldocument.WhitespaceToken, " "},
		{sqldocument.UnquotedIdentifierToken, "add_numbers"},
		{sqldocument.LeftParenToken, "("},
		{sqldocument.UnquotedIdentifierToken, "a"},
		{sqldocument.WhitespaceToken, " "},
		{sqldocument.UnquotedIdentifierToken, "INTEGER"},
		{sqldocument.CommaToken, ","},
		{sqldocument.WhitespaceToken, " "},
		{sqldocument.UnquotedIdentifierToken, "b"},
		{sqldocument.WhitespaceToken, " "},
		{sqldocument.UnquotedIdentifierToken, "INTEGER"},
		{sqldocument.RightParenToken, ")"},
	}

	for i, expected := range tokens {
		tokenType := s.NextToken()
		assert.Equal(t, expected.tokenType, tokenType, "token %d type mismatch", i)
		assert.Equal(t, expected.token, s.Token(), "token %d value mismatch", i)
	}
}

func TestScanner_CreateProcedure(t *testing.T) {
	input := `CREATE PROCEDURE insert_data(value TEXT)
LANGUAGE plpgsql
AS $proc$
BEGIN
    INSERT INTO my_table (data) VALUES (value);
END;
$proc$;`

	s := NewScanner("test.sql", input)

	// Scan through and find the procedure body
	var dollarQuotedBody string
	for {
		tokenType := s.NextToken()
		if tokenType == sqldocument.EOFToken {
			break
		}
		if tokenType == DollarQuotedStringToken {
			dollarQuotedBody = s.Token()
			break
		}
	}

	require.NotEmpty(t, dollarQuotedBody)
	assert.Contains(t, dollarQuotedBody, "BEGIN")
	assert.Contains(t, dollarQuotedBody, "INSERT INTO")
	assert.Contains(t, dollarQuotedBody, "END;")
}

func TestScanner_ComplexFunction(t *testing.T) {
	input := `CREATE OR REPLACE FUNCTION get_user_orders(
    p_user_id INTEGER,
    p_status TEXT DEFAULT 'active'
)
RETURNS TABLE (
    order_id INTEGER,
    order_date TIMESTAMP,
    total_amount NUMERIC(10,2)
)
LANGUAGE plpgsql
AS $func$
DECLARE
    v_count INTEGER;
BEGIN
    -- Count matching orders
    SELECT COUNT(*) INTO v_count
    FROM orders
    WHERE user_id = p_user_id AND status = p_status;
    
    IF v_count > 0 THEN
        RETURN QUERY
        SELECT o.id, o.created_at, o.total
        FROM orders o
        WHERE o.user_id = p_user_id AND o.status = p_status;
    END IF;
END;
$func$;`

	s := NewScanner("test.sql", input)

	// Count different token types
	tokenCounts := make(map[sqldocument.TokenType]int)
	for {
		tokenType := s.NextToken()
		if tokenType == sqldocument.EOFToken {
			break
		}
		tokenCounts[tokenType]++
	}

	// Verify we found expected token types
	assert.Greater(t, tokenCounts[sqldocument.ReservedWordToken], 0, "should have reserved words")
	assert.Greater(t, tokenCounts[sqldocument.UnquotedIdentifierToken], 0, "should have identifiers")
	assert.Greater(t, tokenCounts[sqldocument.LeftParenToken], 0, "should have left parens")
	assert.Greater(t, tokenCounts[sqldocument.RightParenToken], 0, "should have right parens")
	assert.Equal(t, 1, tokenCounts[DollarQuotedStringToken], "should have exactly one dollar-quoted body")
}

func TestScanner_TriggerFunction(t *testing.T) {
	input := `CREATE FUNCTION audit_trigger()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        INSERT INTO audit_log (action, table_name, new_data)
        VALUES ('INSERT', TG_TABLE_NAME, row_to_json(NEW));
    ELSIF TG_OP = 'UPDATE' THEN
        INSERT INTO audit_log (action, table_name, old_data, new_data)
        VALUES ('UPDATE', TG_TABLE_NAME, row_to_json(OLD), row_to_json(NEW));
    ELSIF TG_OP = 'DELETE' THEN
        INSERT INTO audit_log (action, table_name, old_data)
        VALUES ('DELETE', TG_TABLE_NAME, row_to_json(OLD));
    END IF;
    RETURN NEW;
END;
$$;`

	s := NewScanner("test.sql", input)

	// Find and verify the dollar-quoted body
	var bodyToken string
	for {
		tokenType := s.NextToken()
		if tokenType == sqldocument.EOFToken {
			break
		}
		if tokenType == DollarQuotedStringToken {
			bodyToken = s.Token()
			break
		}
	}

	require.NotEmpty(t, bodyToken)
	assert.True(t, len(bodyToken) > 50, "body should contain substantial content")
	assert.Contains(t, bodyToken, "TG_OP")
	assert.Contains(t, bodyToken, "INSERT")
	assert.Contains(t, bodyToken, "UPDATE")
	assert.Contains(t, bodyToken, "DELETE")
}

func TestScanner_FunctionWithParameterDefaults(t *testing.T) {
	input := `CREATE FUNCTION paginate(
    p_limit INTEGER DEFAULT 10,
    p_offset INTEGER DEFAULT 0,
    p_order_by TEXT DEFAULT 'id'
)
RETURNS SETOF my_table
LANGUAGE sql
AS $$
    SELECT * FROM my_table
    ORDER BY p_order_by
    LIMIT p_limit OFFSET p_offset;
$$;`

	s := NewScanner("test.sql", input)

	// Count DEFAULT keywords
	defaultCount := 0
	for {
		tokenType := s.NextToken()
		if tokenType == sqldocument.EOFToken {
			break
		}
		if tokenType == sqldocument.ReservedWordToken && s.ReservedWord() == "default" {
			defaultCount++
		}
	}

	assert.Equal(t, 3, defaultCount, "should have 3 DEFAULT keywords")
}

func TestScanner_UnterminatedStrings(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected sqldocument.TokenType
	}{
		{"unterminated string", "'hello", UnterminatedStringLiteralErrorToken},
		{"unterminated quoted identifier", "\"MyTable", UnterminatedQuotedIdentifierErrorToken},
		{"unterminated dollar quote", "$$hello", UnterminatedStringLiteralErrorToken},
		{"unterminated escape string", "E'hello", UnterminatedStringLiteralErrorToken},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewScanner("test.sql", tt.input)
			tokenType := s.NextToken()
			assert.Equal(t, tt.expected, tokenType)
		})
	}
}

func TestScanner_UnicodeString(t *testing.T) {
	input := `U&'d\0061t\+000061'`
	s := NewScanner("test.sql", input)
	tokenType := s.NextToken()
	assert.Equal(t, StringLiteralToken, tokenType)
}

func TestScanner_UnicodeIdentifier(t *testing.T) {
	input := `U&"d\0061t\+000061"`
	s := NewScanner("test.sql", input)
	tokenType := s.NextToken()
	assert.Equal(t, sqldocument.QuotedIdentifierToken, tokenType)
}

func TestScanner_Position(t *testing.T) {
	input := "SELECT\nFROM"
	s := NewScanner("test.sql", input)

	s.NextToken() // SELECT
	start := s.Start()
	assert.Equal(t, 1, start.Line)
	assert.Equal(t, 1, start.Col)

	s.NextToken() // \n
	s.NextToken() // FROM
	start = s.Start()
	assert.Equal(t, 2, start.Line)
	assert.Equal(t, 1, start.Col)
}

func TestScanner_ComparisonOperators(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"<>", "<>"},
		{">=", ">="},
		{"<=", "<="},
		{"!=", "!="},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			s := NewScanner("test.sql", tt.input)
			tokenType := s.NextToken()
			assert.Equal(t, sqldocument.OtherToken, tokenType)
			assert.Equal(t, tt.expected, s.Token())
		})
	}
}
