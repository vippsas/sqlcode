package pgsql

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vippsas/sqlcode/v2/sqlparser/sqldocument"
)

func ParseString(t *testing.T, file, input string) *PGSqlDocument {
	d := &PGSqlDocument{}
	err := d.Parse([]byte(input), sqldocument.FileRef(file))
	require.NoError(t, err)
	return d
}

func TestDocument_AddError(t *testing.T) {
	d := PGSqlDocument{}
	s := NewScanner(sqldocument.FileRef(""), "")
	assert.False(t, d.HasErrors())
	d.addError(s, "test error")
	assert.True(t, d.HasErrors())
}

func TestDocument_Parse_Pragma(t *testing.T) {
	doc := ParseString(t, "test.pgsql", `--sqlcode:include-if feature1,feature2
--sqlcode:include-if feature3

-- Some comment
SELECT 1;
`)

	assert.False(t, doc.HasErrors())
	assert.Equal(t, []string{"feature1", "feature2", "feature3"}, doc.PragmaIncludeIf())
}

func TestDocument_Parse_InvalidPragma(t *testing.T) {
	doc := ParseString(t, "test.pgsql", `--sqlcode:invalid-pragma
SELECT 1;
`)

	assert.True(t, doc.HasErrors())
	assert.Contains(t, doc.Errors()[0].Message, "Illegal pragma")
}

func TestDocument_Parse_SimpleFunction(t *testing.T) {
	input := `CREATE FUNCTION add_numbers(a INTEGER, b INTEGER)
RETURNS INTEGER
LANGUAGE plpgsql
AS $$
BEGIN
    RETURN a + b;
END;
$$;`

	doc := ParseString(t, "test.pgsql", input)

	// Currently the parser only handles pragmas, so no creates yet
	assert.False(t, doc.HasErrors())
}

func TestDocument_Parse_SimpleProcedure(t *testing.T) {
	input := `CREATE PROCEDURE insert_data(value TEXT)
LANGUAGE plpgsql
AS $proc$
BEGIN
    INSERT INTO my_table (data) VALUES (value);
END;
$proc$;`

	doc := ParseString(t, "test.pgsql", input)

	assert.False(t, doc.HasErrors())
}

func TestDocument_Parse_PragmaBeforeFunction(t *testing.T) {
	input := `--sqlcode:include-if production

CREATE FUNCTION my_func()
RETURNS INTEGER
LANGUAGE sql
AS $$
    SELECT 1;
$$;`

	doc := ParseString(t, "test.pgsql", input)

	assert.False(t, doc.HasErrors())
	assert.Equal(t, []string{"production"}, doc.PragmaIncludeIf())
}

func TestDocument_Parse_MultiplePragmas(t *testing.T) {
	input := `--sqlcode:include-if feature-a
--sqlcode:include-if feature-b,feature-c

SELECT 1;`

	doc := ParseString(t, "test.pgsql", input)

	assert.False(t, doc.HasErrors())
	assert.Equal(t, []string{"feature-a", "feature-b", "feature-c"}, doc.PragmaIncludeIf())
}

func TestDocument_Declares(t *testing.T) {
	doc := ParseString(t, "test.pgsql", "SELECT 1;")

	// PostgreSQL doesn't use DECLARE statements like T-SQL
	assert.Nil(t, doc.Declares())
}

func TestDocument_Parse_DollarQuotedFunction(t *testing.T) {
	input := `CREATE OR REPLACE FUNCTION complex_func(
    p_id INTEGER,
    p_name TEXT DEFAULT 'default'
)
RETURNS TABLE (
    id INTEGER,
    name TEXT,
    created_at TIMESTAMP
)
LANGUAGE plpgsql
AS $func$
DECLARE
    v_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO v_count FROM my_table WHERE id = p_id;
    
    IF v_count > 0 THEN
        RETURN QUERY
        SELECT t.id, t.name, t.created_at
        FROM my_table t
        WHERE t.id = p_id;
    END IF;
END;
$func$;`

	doc := ParseString(t, "test.pgsql", input)

	assert.False(t, doc.HasErrors())
}

func TestDocument_Parse_TriggerFunction(t *testing.T) {
	input := `CREATE FUNCTION audit_trigger()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        INSERT INTO audit_log (action, new_data)
        VALUES ('INSERT', row_to_json(NEW));
    ELSIF TG_OP = 'UPDATE' THEN
        INSERT INTO audit_log (action, old_data, new_data)
        VALUES ('UPDATE', row_to_json(OLD), row_to_json(NEW));
    ELSIF TG_OP = 'DELETE' THEN
        INSERT INTO audit_log (action, old_data)
        VALUES ('DELETE', row_to_json(OLD));
    END IF;
    RETURN NEW;
END;
$$;`

	doc := ParseString(t, "test.pgsql", input)

	assert.False(t, doc.HasErrors())
}

func TestDocument_Parse_MultipleStatements(t *testing.T) {
	input := `--sqlcode:include-if feature1

CREATE FUNCTION func1()
RETURNS INTEGER
LANGUAGE sql
AS $$ SELECT 1; $$;

CREATE FUNCTION func2()
RETURNS INTEGER
LANGUAGE sql
AS $$ SELECT 2; $$;

CREATE PROCEDURE proc1()
LANGUAGE plpgsql
AS $$
BEGIN
    PERFORM func1();
END;
$$;`

	doc := ParseString(t, "test.pgsql", input)

	assert.False(t, doc.HasErrors())
	assert.Equal(t, []string{"feature1"}, doc.PragmaIncludeIf())
}
