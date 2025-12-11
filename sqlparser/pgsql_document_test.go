package sqlparser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDocument_PostgreSQL17_parseCreate(t *testing.T) {
	t.Run("parses PostgreSQL function with dollar quoting", func(t *testing.T) {
		doc := &TSqlDocument{}
		s := NewScanner("test.pgsql", "create function test_func() returns int as $$ begin return 1; end; $$ language plpgsql")
		s.NextToken()
		s.NextNonWhitespaceCommentToken()

		create := doc.parseCreate(s, 0)

		assert.Equal(t, "function", create.CreateType)
		assert.Equal(t, "test_func", create.QuotedName.Value)
	})

	t.Run("parses PostgreSQL procedure", func(t *testing.T) {
		doc := &TSqlDocument{}
		s := NewScanner("test.pgsql", "create procedure insert_data(a integer, b integer) language sql as $$ insert into tbl values (a, b); $$")
		s.NextToken()
		s.NextNonWhitespaceCommentToken()

		create := doc.parseCreate(s, 0)

		assert.Equal(t, "procedure", create.CreateType)
		assert.Equal(t, "insert_data", create.QuotedName.Value)
	})

	t.Run("parses CREATE OR REPLACE", func(t *testing.T) {
		doc := &TSqlDocument{}
		s := NewScanner("test.pgsql", "create or replace function test_func() returns int as $$ begin return 1; end; $$ language plpgsql")
		s.NextToken()
		s.NextNonWhitespaceCommentToken()

		create := doc.parseCreate(s, 0)

		assert.Equal(t, "function", create.CreateType)
	})

	t.Run("parses schema-qualified name", func(t *testing.T) {
		doc := &TSqlDocument{}
		s := NewScanner("test.pgsql", "create function public.test_func() returns int as $$ begin return 1; end; $$ language plpgsql")
		s.NextToken()
		s.NextNonWhitespaceCommentToken()

		create := doc.parseCreate(s, 0)

		assert.Contains(t, create.QuotedName.Value, "test_func")
	})

	t.Run("parses RETURNS TABLE", func(t *testing.T) {
		doc := &TSqlDocument{}
		s := NewScanner("test.pgsql", "create function get_users() returns table(id int, name text) as $$ select id, name from users; $$ language sql")
		s.NextToken()
		s.NextNonWhitespaceCommentToken()

		create := doc.parseCreate(s, 0)

		assert.Equal(t, "function", create.CreateType)
	})

	t.Run("tracks dependencies with schema prefix", func(t *testing.T) {
		doc := &TSqlDocument{}
		s := NewScanner("test.pgsql", "create function test() returns int as $$ select * from public.table1 join public.table2 on table1.id = table2.id; $$ language sql")
		s.NextToken()
		s.NextNonWhitespaceCommentToken()

		create := doc.parseCreate(s, 0)

		require.Len(t, create.DependsOn, 2)
	})

	t.Run("parses volatility categories", func(t *testing.T) {
		doc := &TSqlDocument{}
		s := NewScanner("test.pgsql", "create function test_func() returns int immutable as $$ begin return 1; end; $$ language plpgsql")
		s.NextToken()
		s.NextNonWhitespaceCommentToken()

		create := doc.parseCreate(s, 0)

		assert.Equal(t, "function", create.CreateType)
	})

	t.Run("parses PARALLEL SAFE", func(t *testing.T) {
		doc := &TSqlDocument{}
		s := NewScanner("test.pgsql", "create function test_func() returns int parallel safe as $$ begin return 1; end; $$ language plpgsql")
		s.NextToken()
		s.NextNonWhitespaceCommentToken()

		create := doc.parseCreate(s, 0)

		assert.Equal(t, "function", create.CreateType)
	})
}

func TestDocument_PostgreSQL17_Types(t *testing.T) {
	t.Run("parses composite type", func(t *testing.T) {
		doc := &TSqlDocument{}
		s := NewScanner("test.pgsql", "create type address_type as (street text, city text, zip varchar(10))")
		s.NextToken()
		s.NextNonWhitespaceCommentToken()

		create := doc.parseCreate(s, 0)

		assert.Equal(t, "type", create.CreateType)
	})

	t.Run("parses enum type", func(t *testing.T) {
		doc := &TSqlDocument{}
		s := NewScanner("test.pgsql", "create type mood as enum ('sad', 'ok', 'happy')")
		s.NextToken()
		s.NextNonWhitespaceCommentToken()

		create := doc.parseCreate(s, 0)

		assert.Equal(t, "type", create.CreateType)
	})

	t.Run("parses range type", func(t *testing.T) {
		doc := &TSqlDocument{}
		s := NewScanner("test.pgsql", "create type float_range as range (subtype = float8, subtype_diff = float8mi)")
		s.NextToken()
		s.NextNonWhitespaceCommentToken()

		create := doc.parseCreate(s, 0)

		assert.Equal(t, "type", create.CreateType)
	})
}

func TestDocument_PostgreSQL17_Extensions(t *testing.T) {
	t.Run("parses JSON functions PostgreSQL 17", func(t *testing.T) {
		doc := &TSqlDocument{}
		s := NewScanner("test.pgsql", "create function test() returns jsonb as $$ select json_serialize(data) from table1; $$ language sql")
		s.NextToken()
		s.NextNonWhitespaceCommentToken()

		create := doc.parseCreate(s, 0)

		assert.Equal(t, "function", create.CreateType)
	})

	t.Run("parses MERGE statement (PostgreSQL 15+)", func(t *testing.T) {
		doc := &TSqlDocument{}
		s := NewScanner("test.pgsql", "create function do_merge() returns void as $$ merge into target using source on target.id = source.id when matched then update set value = source.value; $$ language sql")
		s.NextToken()
		s.NextNonWhitespaceCommentToken()

		create := doc.parseCreate(s, 0)

		assert.Equal(t, "function", create.CreateType)
	})
}

func TestDocument_PostgreSQL17_Identifiers(t *testing.T) {
	t.Run("parses double-quoted identifiers", func(t *testing.T) {
		doc := &TSqlDocument{}
		s := NewScanner("test.pgsql", `create function "Test Func"() returns int as $$ begin return 1; end; $$ language plpgsql`)
		s.NextToken()
		s.NextNonWhitespaceCommentToken()

		create := doc.parseCreate(s, 0)

		assert.Contains(t, create.QuotedName.Value, "Test Func")
	})

	t.Run("parses case-sensitive identifiers", func(t *testing.T) {
		doc := &TSqlDocument{}
		s := NewScanner("test.pgsql", `create function "TestFunc"() returns int as $$ begin return 1; end; $$ language plpgsql`)
		s.NextToken()
		s.NextNonWhitespaceCommentToken()

		create := doc.parseCreate(s, 0)

		assert.Contains(t, create.QuotedName.Value, "TestFunc")
	})
}

func TestDocument_PostgreSQL17_Datatypes(t *testing.T) {
	t.Run("parses array types", func(t *testing.T) {
		doc := &TSqlDocument{}
		s := NewScanner("test.pgsql", "integer[]")
		s.NextToken()

		typ := doc.parseTypeExpression(s)

		assert.Equal(t, "integer[]", typ.BaseType)
	})

	t.Run("parses serial types", func(t *testing.T) {
		doc := &TSqlDocument{}
		s := NewScanner("test.pgsql", "serial")
		s.NextToken()

		typ := doc.parseTypeExpression(s)

		assert.Equal(t, "serial", typ.BaseType)
	})

	t.Run("parses text type", func(t *testing.T) {
		doc := &TSqlDocument{}
		s := NewScanner("test.pgsql", "text")
		s.NextToken()

		typ := doc.parseTypeExpression(s)

		assert.Equal(t, "text", typ.BaseType)
	})

	t.Run("parses jsonb type", func(t *testing.T) {
		doc := &TSqlDocument{}
		s := NewScanner("test.pgsql", "jsonb")
		s.NextToken()

		typ := doc.parseTypeExpression(s)

		assert.Equal(t, "jsonb", typ.BaseType)
	})

	t.Run("parses uuid type", func(t *testing.T) {
		doc := &TSqlDocument{}
		s := NewScanner("test.pgsql", "uuid")
		s.NextToken()

		typ := doc.parseTypeExpression(s)

		assert.Equal(t, "uuid", typ.BaseType)
	})
}

func TestDocument_PostgreSQL17_BatchSeparator(t *testing.T) {
	t.Run("PostgreSQL uses semicolon not GO", func(t *testing.T) {
		doc := &TSqlDocument{}
		s := NewScanner("test.pgsql", "create function test1() returns int as $$ begin return 1; end; $$ language plpgsql; create function test2() returns int as $$ begin return 2; end; $$ language plpgsql;")
		s.NextToken()
		s.NextNonWhitespaceCommentToken()

		create1 := doc.parseCreate(s, 0)
		assert.Equal(t, "test1", create1.QuotedName.Value)

		// Move to next statement
		s.NextNonWhitespaceCommentToken()
		s.NextNonWhitespaceCommentToken()

		create2 := doc.parseCreate(s, 1)
		assert.Equal(t, "test2", create2.QuotedName.Value)
	})
}
