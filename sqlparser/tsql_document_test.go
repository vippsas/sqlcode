package sqlparser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDocument_addError(t *testing.T) {
	t.Run("adds error with position", func(t *testing.T) {
		doc := &TSqlDocument{}
		s := NewScanner("test.sql", "select")
		s.NextToken()

		doc.addError(s, "test error message")
		require.True(t, doc.HasErrors())
		assert.Equal(t, "test error message", doc.errors[0].Message)
		assert.Equal(t, Pos{File: "test.sql", Line: 1, Col: 1}, doc.errors[0].Pos)
	})

	t.Run("accumulates multiple errors", func(t *testing.T) {
		doc := &TSqlDocument{}
		s := NewScanner("test.sql", "abc def")
		s.NextToken()
		doc.addError(s, "error 1")
		s.NextToken()
		doc.addError(s, "error 2")

		require.Len(t, doc.errors, 2)
		assert.Equal(t, "error 1", doc.errors[0].Message)
		assert.Equal(t, "error 2", doc.errors[1].Message)
	})

	t.Run("creates error with token text", func(t *testing.T) {
		doc := &TSqlDocument{}
		s := NewScanner("test.sql", "unexpected_token")
		s.NextToken()

		doc.unexpectedTokenError(s)

		require.Len(t, doc.errors, 1)
		assert.Equal(t, "Unexpected: unexpected_token", doc.errors[0].Message)
	})
}

func TestDocument_parseTypeExpression(t *testing.T) {
	t.Run("parses simple type without args", func(t *testing.T) {
		doc := &TSqlDocument{}
		s := NewScanner("test.sql", "int")
		s.NextToken()

		typ := doc.parseTypeExpression(s)

		assert.Equal(t, "int", typ.BaseType)
		assert.Empty(t, typ.Args)
	})

	t.Run("parses type with single arg", func(t *testing.T) {
		doc := &TSqlDocument{}
		s := NewScanner("test.sql", "varchar(50)")
		s.NextToken()

		typ := doc.parseTypeExpression(s)

		assert.Equal(t, "varchar", typ.BaseType)
		assert.Equal(t, []string{"50"}, typ.Args)
	})

	t.Run("parses type with multiple args", func(t *testing.T) {
		doc := &TSqlDocument{}
		s := NewScanner("test.sql", "decimal(10, 2)")
		s.NextToken()

		typ := doc.parseTypeExpression(s)

		assert.Equal(t, "decimal", typ.BaseType)
		assert.Equal(t, []string{"10", "2"}, typ.Args)
	})

	t.Run("parses type with max", func(t *testing.T) {
		doc := &TSqlDocument{}
		s := NewScanner("test.sql", "nvarchar(max)")
		s.NextToken()

		typ := doc.parseTypeExpression(s)

		assert.Equal(t, "nvarchar", typ.BaseType)
		assert.Equal(t, []string{"max"}, typ.Args)
	})

	t.Run("handles invalid arg", func(t *testing.T) {
		doc := &TSqlDocument{}
		s := NewScanner("test.sql", "varchar(invalid)")
		s.NextToken()

		typ := doc.parseTypeExpression(s)

		assert.Equal(t, "varchar", typ.BaseType)
		assert.NotEmpty(t, doc.errors)
	})

	t.Run("panics if not on identifier", func(t *testing.T) {
		doc := &TSqlDocument{}
		s := NewScanner("test.sql", "123")
		s.NextToken()

		assert.Panics(t, func() {
			doc.parseTypeExpression(s)
		})
	})
}

func TestDocument_parseDeclare(t *testing.T) {
	t.Run("parses single enum declaration", func(t *testing.T) {
		doc := &TSqlDocument{}
		s := NewScanner("test.sql", "@EnumStatus int = 42")
		s.NextToken()

		declares := doc.parseDeclare(s)

		require.Len(t, declares, 1)
		assert.Equal(t, "@EnumStatus", declares[0].VariableName)
		assert.Equal(t, "int", declares[0].Datatype.BaseType)
		assert.Equal(t, "42", declares[0].Literal.RawValue)
	})

	t.Run("parses multiple declarations with comma", func(t *testing.T) {
		doc := &TSqlDocument{}
		s := NewScanner("test.sql", "@EnumA int = 1, @EnumB int = 2;")
		s.NextToken()

		declares := doc.parseDeclare(s)

		require.Len(t, declares, 2)
		assert.Equal(t, "@EnumA", declares[0].VariableName)
		assert.Equal(t, "@EnumB", declares[1].VariableName)
	})

	t.Run("parses string literal", func(t *testing.T) {
		doc := &TSqlDocument{}
		s := NewScanner("test.sql", "@EnumName nvarchar(50) = N'test'")
		s.NextToken()

		declares := doc.parseDeclare(s)

		require.Len(t, declares, 1)
		assert.Equal(t, "N'test'", declares[0].Literal.RawValue)
	})

	t.Run("errors on invalid variable name", func(t *testing.T) {
		doc := &TSqlDocument{}
		s := NewScanner("test.sql", "@InvalidName int = 1")
		s.NextToken()

		declares := doc.parseDeclare(s)

		require.Len(t, declares, 1)
		assert.NotEmpty(t, doc.errors)
		assert.Contains(t, doc.errors[0].Message, "@InvalidName")
	})

	t.Run("errors on missing type", func(t *testing.T) {
		doc := &TSqlDocument{}
		s := NewScanner("test.sql", "@EnumTest = 42")
		s.NextToken()

		declares := doc.parseDeclare(s)

		require.Len(t, declares, 1)
		assert.NotEmpty(t, doc.errors)
		assert.Contains(t, doc.errors[0].Message, "type declared explicitly")
	})

	t.Run("errors on missing assignment", func(t *testing.T) {
		doc := &TSqlDocument{}
		s := NewScanner("test.sql", "@EnumTest int")
		s.NextToken()

		doc.parseDeclare(s)

		assert.NotEmpty(t, doc.errors)
		assert.Contains(t, doc.errors[0].Message, "needs to be assigned")
	})

	t.Run("accepts @Global prefix", func(t *testing.T) {
		doc := &TSqlDocument{}
		s := NewScanner("test.sql", "@GlobalSetting int = 100")
		s.NextToken()

		declares := doc.parseDeclare(s)

		require.Len(t, declares, 1)
		assert.Equal(t, "@GlobalSetting", declares[0].VariableName)
		assert.Empty(t, doc.errors)
	})

	t.Run("accepts @Const prefix", func(t *testing.T) {
		doc := &TSqlDocument{}
		s := NewScanner("test.sql", "@ConstValue int = 200")
		s.NextToken()

		declares := doc.parseDeclare(s)

		require.Len(t, declares, 1)
		assert.Equal(t, "@ConstValue", declares[0].VariableName)
		assert.Empty(t, doc.errors)
	})
}

func TestDocument_parseBatchSeparator(t *testing.T) {
	t.Run("parses valid go separator", func(t *testing.T) {
		doc := &TSqlDocument{}
		s := NewScanner("test.sql", "go\n")
		s.NextToken()

		doc.parseBatchSeparator(s)

		assert.Empty(t, doc.errors)
	})

	t.Run("errors on malformed separator", func(t *testing.T) {
		doc := &TSqlDocument{}
		s := NewScanner("test.sql", "go -- comment")
		s.NextToken()

		doc.parseBatchSeparator(s)

		assert.NotEmpty(t, doc.errors)
		assert.Contains(t, doc.errors[0].Message, "should be alone")
	})
}

func TestDocument_parseCodeschemaName(t *testing.T) {
	t.Run("parses unquoted identifier", func(t *testing.T) {
		doc := &TSqlDocument{}
		s := NewScanner("test.sql", "[code].TestProc")
		s.NextToken()
		var target []Unparsed

		result := doc.parseCodeschemaName(s, &target)

		assert.Equal(t, "[TestProc]", result.Value)
		assert.NotEmpty(t, target)
	})

	t.Run("parses quoted identifier", func(t *testing.T) {
		doc := &TSqlDocument{}
		s := NewScanner("test.sql", "[code].[Test Proc]")
		s.NextToken()
		var target []Unparsed

		result := doc.parseCodeschemaName(s, &target)

		assert.Equal(t, "[Test Proc]", result.Value)
	})

	t.Run("errors on missing dot", func(t *testing.T) {
		doc := &TSqlDocument{}
		s := NewScanner("test.sql", "[code] TestProc")
		s.NextToken()
		var target []Unparsed

		result := doc.parseCodeschemaName(s, &target)

		assert.Equal(t, "", result.Value)
		assert.NotEmpty(t, doc.errors)
		assert.Contains(t, doc.errors[0].Message, "must be followed by '.'")
	})

	t.Run("errors on missing identifier", func(t *testing.T) {
		doc := &TSqlDocument{}
		s := NewScanner("test.sql", "[code].123")
		s.NextToken()
		var target []Unparsed

		result := doc.parseCodeschemaName(s, &target)

		assert.Equal(t, "", result.Value)
		assert.NotEmpty(t, doc.errors)
		assert.Contains(t, doc.errors[0].Message, "must be followed an identifier")
	})
}

func TestDocument_parseCreate(t *testing.T) {
	t.Run("parses simple procedure", func(t *testing.T) {
		doc := &TSqlDocument{}
		s := NewScanner("test.sql", "create procedure [code].TestProc as begin end")
		s.NextToken()
		s.NextNonWhitespaceCommentToken()

		create := doc.parseCreate(s, 0)

		assert.Equal(t, "procedure", create.CreateType)
		assert.Equal(t, "[TestProc]", create.QuotedName.Value)
		assert.NotEmpty(t, create.Body)
	})

	t.Run("parses function", func(t *testing.T) {
		doc := &TSqlDocument{}
		s := NewScanner("test.sql", "create function [code].TestFunc() returns int as begin return 1 end")
		s.NextToken()
		s.NextNonWhitespaceCommentToken()

		create := doc.parseCreate(s, 0)

		assert.Equal(t, "function", create.CreateType)
		assert.Equal(t, "[TestFunc]", create.QuotedName.Value)
	})

	t.Run("parses type", func(t *testing.T) {
		doc := &TSqlDocument{}
		s := NewScanner("test.sql", "create type [code].TestType as table (id int)")
		s.NextToken()
		s.NextNonWhitespaceCommentToken()

		create := doc.parseCreate(s, 0)

		assert.Equal(t, "type", create.CreateType)
		assert.Equal(t, "[TestType]", create.QuotedName.Value)
	})

	t.Run("tracks dependencies", func(t *testing.T) {
		doc := &TSqlDocument{}
		s := NewScanner("test.sql", "create procedure [code].Proc1 as begin select * from [code].Table1 join [code].Table2 end")
		s.NextToken()
		s.NextNonWhitespaceCommentToken()

		create := doc.parseCreate(s, 0)

		require.Len(t, create.DependsOn, 2)
		assert.Equal(t, "[Table1]", create.DependsOn[0].Value)
		assert.Equal(t, "[Table2]", create.DependsOn[1].Value)
	})

	t.Run("deduplicates dependencies", func(t *testing.T) {
		doc := &TSqlDocument{}
		s := NewScanner("test.sql", "create procedure [code].Proc1 as begin select * from [code].Table1; select * from [code].Table1 end")
		s.NextToken()
		s.NextNonWhitespaceCommentToken()

		create := doc.parseCreate(s, 0)

		require.Len(t, create.DependsOn, 1)
		assert.Equal(t, "[Table1]", create.DependsOn[0].Value)
	})

	t.Run("errors on unsupported create type", func(t *testing.T) {
		doc := &TSqlDocument{}
		s := NewScanner("test.sql", "create table [code].TestTable (id int)")
		s.NextToken()
		s.NextNonWhitespaceCommentToken()

		doc.parseCreate(s, 0)

		assert.NotEmpty(t, doc.errors)
		assert.Contains(t, doc.errors[0].Message, "only supports creating procedures")
	})

	t.Run("errors on multiple procedures in batch", func(t *testing.T) {
		doc := &TSqlDocument{}
		s := NewScanner("test.sql", "create procedure [code].Proc2 as begin end")
		s.NextToken()
		s.NextNonWhitespaceCommentToken()

		doc.parseCreate(s, 1)

		assert.NotEmpty(t, doc.errors)
		assert.Contains(t, doc.errors[0].Message, "must be alone in a batch")
	})

	t.Run("errors on missing code schema", func(t *testing.T) {
		doc := &TSqlDocument{}
		s := NewScanner("test.sql", "create procedure dbo.TestProc as begin end")
		s.NextToken()
		s.NextNonWhitespaceCommentToken()

		doc.parseCreate(s, 0)

		assert.NotEmpty(t, doc.errors)
		assert.Contains(t, doc.errors[0].Message, "must be followed by [code]")
	})

	t.Run("allows create index inside procedure", func(t *testing.T) {
		doc := &TSqlDocument{}
		s := NewScanner("test.sql", "create procedure [code].Proc as begin create index IX_Test on #temp(id) end")
		s.NextToken()
		s.NextNonWhitespaceCommentToken()

		create := doc.parseCreate(s, 0)

		assert.Equal(t, "procedure", create.CreateType)
		assert.Empty(t, doc.errors)
	})

	t.Run("stops at batch separator", func(t *testing.T) {
		doc := &TSqlDocument{}
		s := NewScanner("test.sql", "create procedure [code].Proc as begin end\ngo")
		s.NextToken()
		s.NextNonWhitespaceCommentToken()

		create := doc.parseCreate(s, 0)

		assert.Equal(t, "[Proc]", create.QuotedName.Value)
		assert.Equal(t, BatchSeparatorToken, s.TokenType())
	})

	t.Run("panics if not on create token", func(t *testing.T) {
		doc := &TSqlDocument{}
		s := NewScanner("test.sql", "procedure")
		s.NextToken()

		assert.Panics(t, func() {
			doc.parseCreate(s, 0)
		})
	})
}

func TestNextTokenCopyingWhitespace(t *testing.T) {
	t.Run("copies whitespace tokens", func(t *testing.T) {
		s := NewScanner("test.sql", "   \n\t  token")
		var target []Unparsed

		NextTokenCopyingWhitespace(s, &target)

		assert.NotEmpty(t, target)
		assert.Equal(t, UnquotedIdentifierToken, s.TokenType())
	})

	t.Run("copies comments", func(t *testing.T) {
		s := NewScanner("test.sql", "/* comment */ -- line\ntoken")
		var target []Unparsed

		NextTokenCopyingWhitespace(s, &target)

		assert.True(t, len(target) >= 2)
		assert.Equal(t, UnquotedIdentifierToken, s.TokenType())
	})

	t.Run("stops at EOF", func(t *testing.T) {
		s := NewScanner("test.sql", "   ")
		var target []Unparsed

		NextTokenCopyingWhitespace(s, &target)

		assert.Equal(t, EOFToken, s.TokenType())
	})
}

func TestCreateUnparsed(t *testing.T) {
	t.Run("creates unparsed from scanner", func(t *testing.T) {
		s := NewScanner("test.sql", "select")
		s.NextToken()

		unparsed := CreateUnparsed(s)

		assert.Equal(t, ReservedWordToken, unparsed.Type)
		assert.Equal(t, "select", unparsed.RawValue)
		assert.Equal(t, Pos{File: "test.sql", Line: 1, Col: 1}, unparsed.Start)
	})
}

func TestDocument_recoverToNextStatement(t *testing.T) {
	t.Run("recovers to declare", func(t *testing.T) {
		doc := &TSqlDocument{}
		s := NewScanner("test.sql", "invalid tokens here declare @x int = 1")
		s.NextToken()

		doc.recoverToNextStatement(s)

		assert.Equal(t, ReservedWordToken, s.TokenType())
		assert.Equal(t, "declare", s.ReservedWord())
	})

	t.Run("recovers to create", func(t *testing.T) {
		doc := &TSqlDocument{}
		s := NewScanner("test.sql", "bad stuff create procedure")
		s.NextToken()

		doc.recoverToNextStatement(s)

		assert.Equal(t, "create", s.ReservedWord())
	})

	t.Run("recovers to go", func(t *testing.T) {
		doc := &TSqlDocument{}
		s := NewScanner("test.sql", "error error go")
		s.NextToken()

		doc.recoverToNextStatement(s)

		assert.Equal(t, "go", s.ReservedWord())
	})

	t.Run("stops at EOF", func(t *testing.T) {
		doc := &TSqlDocument{}
		s := NewScanner("test.sql", "no keywords")
		s.NextToken()

		doc.recoverToNextStatement(s)

		assert.Equal(t, EOFToken, s.TokenType())
	})
}

func TestDocument_recoverToNextStatementCopying(t *testing.T) {
	t.Run("copies tokens while recovering", func(t *testing.T) {
		doc := &TSqlDocument{}
		s := NewScanner("test.sql", "bad token declare")
		s.NextToken()
		var target []Unparsed

		doc.recoverToNextStatementCopying(s, &target)

		assert.NotEmpty(t, target)
		assert.Equal(t, "declare", s.ReservedWord())
	})
}

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

// func TestDocument_PostgreSQL17_Types(t *testing.T) {
// 	t.Run("parses composite type", func(t *testing.T) {
// 		doc := &TSqlDocument{}
// 		s := NewScanner("test.pgsql", "create type address_type as (street text, city text, zip varchar(10))")
// 		s.NextToken()
// 		s.NextNonWhitespaceCommentToken()

// 		create := doc.parseCreate(s, 0)

// 		assert.Equal(t, "type", create.CreateType)
// 	})

// 	t.Run("parses enum type", func(t *testing.T) {
// 		doc := &TSqlDocument{}
// 		s := NewScanner("test.pgsql", "create type mood as enum ('sad', 'ok', 'happy')")
// 		s.NextToken()
// 		s.NextNonWhitespaceCommentToken()

// 		create := doc.parseCreate(s, 0)

// 		assert.Equal(t, "type", create.CreateType)
// 	})

// 	t.Run("parses range type", func(t *testing.T) {
// 		doc := &TSqlDocument{}
// 		s := NewScanner("test.pgsql", "create type float_range as range (subtype = float8, subtype_diff = float8mi)")
// 		s.NextToken()
// 		s.NextNonWhitespaceCommentToken()

// 		create := doc.parseCreate(s, 0)

// 		assert.Equal(t, "type", create.CreateType)
// 	})
// }

// func TestDocument_PostgreSQL17_Extensions(t *testing.T) {
// 	t.Run("parses JSON functions PostgreSQL 17", func(t *testing.T) {
// 		doc := &TSqlDocument{}
// 		s := NewScanner("test.pgsql", "create function test() returns jsonb as $$ select json_serialize(data) from table1; $$ language sql")
// 		s.NextToken()
// 		s.NextNonWhitespaceCommentToken()

// 		create := doc.parseCreate(s, 0)

// 		assert.Equal(t, "function", create.CreateType)
// 	})

// 	t.Run("parses MERGE statement (PostgreSQL 15+)", func(t *testing.T) {
// 		doc := &TSqlDocument{}
// 		s := NewScanner("test.pgsql", "create function do_merge() returns void as $$ merge into target using source on target.id = source.id when matched then update set value = source.value; $$ language sql")
// 		s.NextToken()
// 		s.NextNonWhitespaceCommentToken()

// 		create := doc.parseCreate(s, 0)

// 		assert.Equal(t, "function", create.CreateType)
// 	})
// }

// func TestDocument_PostgreSQL17_Identifiers(t *testing.T) {
// 	t.Run("parses double-quoted identifiers", func(t *testing.T) {
// 		doc := &TSqlDocument{}
// 		s := NewScanner("test.pgsql", `create function "Test Func"() returns int as $$ begin return 1; end; $$ language plpgsql`)
// 		s.NextToken()
// 		s.NextNonWhitespaceCommentToken()

// 		create := doc.parseCreate(s, 0)

// 		assert.Contains(t, create.QuotedName.Value, "Test Func")
// 	})

// 	t.Run("parses case-sensitive identifiers", func(t *testing.T) {
// 		doc := &TSqlDocument{}
// 		s := NewScanner("test.pgsql", `create function "TestFunc"() returns int as $$ begin return 1; end; $$ language plpgsql`)
// 		s.NextToken()
// 		s.NextNonWhitespaceCommentToken()

// 		create := doc.parseCreate(s, 0)

// 		assert.Contains(t, create.QuotedName.Value, "TestFunc")
// 	})
// }

// func TestDocument_PostgreSQL17_Datatypes(t *testing.T) {
// 	t.Run("parses array types", func(t *testing.T) {
// 		doc := &TSqlDocument{}
// 		s := NewScanner("test.pgsql", "integer[]")
// 		s.NextToken()

// 		typ := doc.parseTypeExpression(s)

// 		assert.Equal(t, "integer[]", typ.BaseType)
// 	})

// 	t.Run("parses serial types", func(t *testing.T) {
// 		doc := &TSqlDocument{}
// 		s := NewScanner("test.pgsql", "serial")
// 		s.NextToken()

// 		typ := doc.parseTypeExpression(s)

// 		assert.Equal(t, "serial", typ.BaseType)
// 	})

// 	t.Run("parses text type", func(t *testing.T) {
// 		doc := &TSqlDocument{}
// 		s := NewScanner("test.pgsql", "text")
// 		s.NextToken()

// 		typ := doc.parseTypeExpression(s)

// 		assert.Equal(t, "text", typ.BaseType)
// 	})

// 	t.Run("parses jsonb type", func(t *testing.T) {
// 		doc := &TSqlDocument{}
// 		s := NewScanner("test.pgsql", "jsonb")
// 		s.NextToken()

// 		typ := doc.parseTypeExpression(s)

// 		assert.Equal(t, "jsonb", typ.BaseType)
// 	})

// 	t.Run("parses uuid type", func(t *testing.T) {
// 		doc := &TSqlDocument{}
// 		s := NewScanner("test.pgsql", "uuid")
// 		s.NextToken()

// 		typ := doc.parseTypeExpression(s)

// 		assert.Equal(t, "uuid", typ.BaseType)
// 	})
// }

// func TestDocument_PostgreSQL17_BatchSeparator(t *testing.T) {
// 	t.Run("PostgreSQL uses semicolon not GO", func(t *testing.T) {
// 		doc := &TSqlDocument{}
// 		s := NewScanner("test.pgsql", "create function test1() returns int as $$ begin return 1; end; $$ language plpgsql; create function test2() returns int as $$ begin return 2; end; $$ language plpgsql;")
// 		s.NextToken()
// 		s.NextNonWhitespaceCommentToken()

// 		create1 := doc.parseCreate(s, 0)
// 		assert.Equal(t, "test1", create1.QuotedName.Value)

// 		// Move to next statement
// 		s.NextNonWhitespaceCommentToken()
// 		s.NextNonWhitespaceCommentToken()

// 		create2 := doc.parseCreate(s, 1)
// 		assert.Equal(t, "test2", create2.QuotedName.Value)
// 	})
// }
