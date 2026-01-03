package mssql

import (
	"testing"

	mssql "github.com/microsoft/go-mssqldb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vippsas/sqlcode/sqlparser/sqldocument"
)

func ParseString(t *testing.T, file, input string) *TSqlDocument {
	d := &TSqlDocument{}
	assert.NoError(t, d.Parse([]byte(input), sqldocument.FileRef(file)))
	return d
}

func TestParserSmokeTest(t *testing.T) {
	doc := ParseString(t, "test.sql", `
/* test is a test

declare @EnumFoo int = 2;

*/

declare/*comment*/@EnumBar1 varchar (max) = N'declare @EnumThisIsInString';
declare


    @EnumBar2 int = 20,
    @EnumBar3 int=21;

GO

declare @EnumNextBatch int = 3;

go
-- preceding comment 1
/* preceding comment 2

asdfasdf */create procedure [code].TestFunc as begin
  refers to [code].OtherFunc [code].HelloFunc;
  create table x ( int x not null );  -- should be ok
end;

/* trailing comment */
`)
	require.Equal(t, 1, len(doc.Creates()))
	c := doc.Creates()[0]
	require.Equal(t, &mssql.Driver{}, c.Driver)

	assert.Equal(t, "[TestFunc]", c.QuotedName.Value)
	assert.Equal(t, []string{"[HelloFunc]", "[OtherFunc]"}, c.DependsOnStrings())
	assert.Equal(t, `-- preceding comment 1
/* preceding comment 2

asdfasdf */create procedure [code].TestFunc as begin
  refers to [code].OtherFunc [code].HelloFunc;
  create table x ( int x not null );  -- should be ok
end;

/* trailing comment */
`, c.String())

	assert.Equal(t,
		[]sqldocument.Error{
			{
				Message: "'declare' statement only allowed in first batch",
			},
		}, doc.Errors())

	assert.Equal(t,
		[]sqldocument.Declare{
			{
				VariableName: "@EnumBar1",
				Datatype: sqldocument.Type{
					BaseType: "varchar",
					Args: []string{
						"max",
					},
				},
				Literal: sqldocument.Unparsed{
					Type:     NVarcharLiteralToken,
					RawValue: "N'declare @EnumThisIsInString'",
				},
			},
			{
				VariableName: "@EnumBar2",
				Datatype: sqldocument.Type{
					BaseType: "int",
				},
				Literal: sqldocument.Unparsed{
					Type:     sqldocument.NumberToken,
					RawValue: "20",
				},
			},
			{
				VariableName: "@EnumBar3",
				Datatype: sqldocument.Type{
					BaseType: "int",
				},
				Literal: sqldocument.Unparsed{
					Type:     sqldocument.NumberToken,
					RawValue: "21",
				},
			},
			{
				VariableName: "@EnumNextBatch",
				Datatype: sqldocument.Type{
					BaseType: "int",
				},
				Literal: sqldocument.Unparsed{
					Type:     sqldocument.NumberToken,
					RawValue: "3",
				},
			},
		},
		doc.Declares(),
	)
	//	repr.Println(doc)
}

// import (
// 	"fmt"
// 	"strings"
// 	"testing"

// 	"github.com/stretchr/testify/assert"
// 	"github.com/stretchr/testify/require"
// 	"github.com/vippsas/sqlcode/sqlparser/sqldocument"
// )

// // Helper to parse a document from input string
// func parseDocument(input string) *TSqlDocument {
// 	doc := &TSqlDocument{}
// 	s := NewScanner("test.sql", input)
// 	s.NextToken()
// 	doc.Parse(s)
// 	return doc
// }

// func TestTSqlDocument(t *testing.T) {
// 	t.Run("addError", func(t *testing.T) {
// 		t.Run("adds error with position", func(t *testing.T) {
// 			doc := &TSqlDocument{}
// 			s := NewScanner("test.sql", "select")
// 			s.NextToken()

// 			doc.addError(s, "test error message")
// 			require.True(t, doc.HasErrors())
// 			assert.Equal(t, "test error message", doc.errors[0].Message)
// 			assert.Equal(t, Pos{File: "test.sql", Line: 1, Col: 1}, doc.errors[0].Pos)
// 		})

// 		t.Run("accumulates multiple errors", func(t *testing.T) {
// 			doc := &TSqlDocument{}
// 			s := NewScanner("test.sql", "abc def")
// 			s.NextToken()
// 			doc.addError(s, "error 1")
// 			s.NextToken()
// 			doc.addError(s, "error 2")

// 			require.Len(t, doc.errors, 2)
// 			assert.Equal(t, "error 1", doc.errors[0].Message)
// 			assert.Equal(t, "error 2", doc.errors[1].Message)
// 		})

// 		t.Run("creates error with token text", func(t *testing.T) {
// 			doc := &TSqlDocument{}
// 			s := NewScanner("test.sql", "unexpected_token")
// 			s.NextToken()

// 			doc.unexpectedTokenError(s)

// 			require.Len(t, doc.errors, 1)
// 			assert.Equal(t, "Unexpected: unexpected_token", doc.errors[0].Message)
// 		})
// 	})

// 	t.Run("parseTypeExpression", func(t *testing.T) {
// 		t.Run("parses simple type without args", func(t *testing.T) {
// 			doc := &TSqlDocument{}
// 			s := NewScanner("test.sql", "int")
// 			s.NextToken()

// 			typ := doc.parseTypeExpression(s)

// 			assert.Equal(t, "int", typ.BaseType)
// 			assert.Empty(t, typ.Args)
// 		})

// 		t.Run("parses type with single arg", func(t *testing.T) {
// 			doc := &TSqlDocument{}
// 			s := NewScanner("test.sql", "varchar(50)")
// 			s.NextToken()

// 			typ := doc.parseTypeExpression(s)

// 			assert.Equal(t, "varchar", typ.BaseType)
// 			assert.Equal(t, []string{"50"}, typ.Args)
// 		})

// 		t.Run("parses type with multiple args", func(t *testing.T) {
// 			doc := &TSqlDocument{}
// 			s := NewScanner("test.sql", "decimal(10, 2)")
// 			s.NextToken()

// 			typ := doc.parseTypeExpression(s)

// 			assert.Equal(t, "decimal", typ.BaseType)
// 			assert.Equal(t, []string{"10", "2"}, typ.Args)
// 		})

// 		t.Run("parses type with max", func(t *testing.T) {
// 			doc := &TSqlDocument{}
// 			s := NewScanner("test.sql", "nvarchar(max)")
// 			s.NextToken()

// 			typ := doc.parseTypeExpression(s)

// 			assert.Equal(t, "nvarchar", typ.BaseType)
// 			assert.Equal(t, []string{"max"}, typ.Args)
// 		})

// 		t.Run("handles invalid arg", func(t *testing.T) {
// 			doc := &TSqlDocument{}
// 			s := NewScanner("test.sql", "varchar(invalid)")
// 			s.NextToken()

// 			typ := doc.parseTypeExpression(s)

// 			assert.Equal(t, "varchar", typ.BaseType)
// 			assert.NotEmpty(t, doc.errors)
// 		})

// 		t.Run("panics if not on identifier", func(t *testing.T) {
// 			doc := &TSqlDocument{}
// 			s := NewScanner("test.sql", "123")
// 			s.NextToken()

// 			assert.Panics(t, func() {
// 				doc.parseTypeExpression(s)
// 			})
// 		})
// 	})
// }

// func TestDocument_parseDeclare(t *testing.T) {
// 	t.Run("parses single enum declaration", func(t *testing.T) {
// 		doc := &TSqlDocument{}
// 		s := NewScanner("test.sql", "@EnumStatus int = 42")
// 		s.NextToken()

// 		declares := doc.parseDeclare(s)

// 		require.Len(t, declares, 1)
// 		assert.Equal(t, "@EnumStatus", declares[0].VariableName)
// 		assert.Equal(t, "int", declares[0].Datatype.BaseType)
// 		assert.Equal(t, "42", declares[0].Literal.RawValue)
// 	})

// 	t.Run("parses multiple declarations with comma", func(t *testing.T) {
// 		doc := &TSqlDocument{}
// 		s := NewScanner("test.sql", "@EnumA int = 1, @EnumB int = 2;")
// 		s.NextToken()

// 		declares := doc.parseDeclare(s)

// 		require.Len(t, declares, 2)
// 		assert.Equal(t, "@EnumA", declares[0].VariableName)
// 		assert.Equal(t, "@EnumB", declares[1].VariableName)
// 	})

// 	t.Run("parses string literal", func(t *testing.T) {
// 		doc := &TSqlDocument{}
// 		s := NewScanner("test.sql", "@EnumName nvarchar(50) = N'test'")
// 		s.NextToken()

// 		declares := doc.parseDeclare(s)

// 		require.Len(t, declares, 1)
// 		assert.Equal(t, "N'test'", declares[0].Literal.RawValue)
// 	})

// 	t.Run("errors on invalid variable name", func(t *testing.T) {
// 		doc := &TSqlDocument{}
// 		s := NewScanner("test.sql", "@InvalidName int = 1")
// 		s.NextToken()

// 		declares := doc.parseDeclare(s)

// 		// in this case when we detect the missing prefix,
// 		// we add an error and continue parsing the declaration.
// 		// this results with it being added
// 		require.Len(t, declares, 1)
// 		assert.NotEmpty(t, doc.errors)
// 		assert.Contains(t, doc.errors[0].Message, "@InvalidName")
// 	})

// 	t.Run("errors on missing type", func(t *testing.T) {
// 		doc := &TSqlDocument{}
// 		s := NewScanner("test.sql", "@EnumTest = 42")
// 		s.NextToken()

// 		declares := doc.parseDeclare(s)

// 		require.Len(t, declares, 0)
// 		assert.NotEmpty(t, doc.errors)
// 		assert.Contains(t, doc.errors[0].Message, "type declared explicitly")
// 	})

// 	t.Run("errors on missing assignment", func(t *testing.T) {
// 		doc := &TSqlDocument{}
// 		s := NewScanner("test.sql", "@EnumTest int")
// 		s.NextToken()

// 		declares := doc.parseDeclare(s)

// 		require.Len(t, declares, 0)
// 		assert.NotEmpty(t, doc.errors)
// 		assert.Contains(t, doc.errors[0].Message, "needs to be assigned")
// 	})

// 	t.Run("accepts @Global prefix", func(t *testing.T) {
// 		doc := &TSqlDocument{}
// 		s := NewScanner("test.sql", "@GlobalSetting int = 100")
// 		s.NextToken()

// 		declares := doc.parseDeclare(s)

// 		require.Len(t, declares, 1)
// 		assert.Equal(t, "@GlobalSetting", declares[0].VariableName)
// 		assert.Empty(t, doc.errors)
// 	})

// 	t.Run("accepts @Const prefix", func(t *testing.T) {
// 		doc := &TSqlDocument{}
// 		s := NewScanner("test.sql", "@ConstValue int = 200")
// 		s.NextToken()

// 		declares := doc.parseDeclare(s)

// 		require.Len(t, declares, 1)
// 		assert.Equal(t, "@ConstValue", declares[0].VariableName)
// 		assert.Empty(t, doc.errors)
// 	})
// }

// func TestDocument_parseBatchSeparator(t *testing.T) {
// 	t.Run("parses valid go separator", func(t *testing.T) {
// 		doc := &TSqlDocument{}
// 		s := NewScanner("test.sql", "go\n")
// 		s.NextToken()

// 		doc.parseBatchSeparator(s)

// 		assert.Empty(t, doc.errors)
// 	})

// 	t.Run("errors on malformed separator", func(t *testing.T) {
// 		doc := &TSqlDocument{}
// 		s := NewScanner("test.sql", "go -- comment")
// 		tt := s.NextToken()
// 		fmt.Printf("%#v %#v\n", s, tt)
// 		doc.parseBatchSeparator(s)
// 		fmt.Printf("%#v\n", s)

// 		assert.NotEmpty(t, doc.errors)
// 		assert.Contains(t, doc.errors[0].Message, "should be alone")
// 	})
// }

// func TestDocument_parseCreate(t *testing.T) {
// 	t.Run("parses simple procedure", func(t *testing.T) {
// 		doc := &TSqlDocument{}
// 		s := NewScanner("test.sql", "create procedure [code].TestProc as begin end")
// 		s.NextToken()
// 		s.NextNonWhitespaceCommentToken()

// 		create := doc.parseCreate(s, 0)

// 		assert.Equal(t, "procedure", create.CreateType)
// 		assert.Equal(t, "[TestProc]", create.QuotedName.Value)
// 		assert.NotEmpty(t, create.Body)
// 	})

// 	t.Run("parses function", func(t *testing.T) {
// 		doc := &TSqlDocument{}
// 		s := NewScanner("test.sql", "create function [code].TestFunc() returns int as begin return 1 end")
// 		s.NextToken()
// 		s.NextNonWhitespaceCommentToken()

// 		create := doc.parseCreate(s, 0)

// 		assert.Equal(t, "function", create.CreateType)
// 		assert.Equal(t, "[TestFunc]", create.QuotedName.Value)
// 	})

// 	t.Run("parses type", func(t *testing.T) {
// 		doc := &TSqlDocument{}
// 		s := NewScanner("test.sql", "create type [code].TestType as table (id int)")
// 		s.NextToken()
// 		s.NextNonWhitespaceCommentToken()

// 		create := doc.parseCreate(s, 0)

// 		assert.Equal(t, "type", create.CreateType)
// 		assert.Equal(t, "[TestType]", create.QuotedName.Value)
// 	})

// 	t.Run("tracks dependencies", func(t *testing.T) {
// 		doc := &TSqlDocument{}
// 		s := NewScanner("test.sql", "create procedure [code].Proc1 as begin select * from [code].Table1 join [code].Table2 end")
// 		s.NextToken()
// 		s.NextNonWhitespaceCommentToken()

// 		create := doc.parseCreate(s, 0)

// 		require.Len(t, create.DependsOn, 2)
// 		assert.Equal(t, "[Table1]", create.DependsOn[0].Value)
// 		assert.Equal(t, "[Table2]", create.DependsOn[1].Value)
// 	})

// 	t.Run("deduplicates dependencies", func(t *testing.T) {
// 		doc := &TSqlDocument{}
// 		s := NewScanner("test.sql", "create procedure [code].Proc1 as begin select * from [code].Table1; select * from [code].Table1 end")
// 		s.NextToken()
// 		s.NextNonWhitespaceCommentToken()

// 		create := doc.parseCreate(s, 0)

// 		require.Len(t, create.DependsOn, 1)
// 		assert.Equal(t, "[Table1]", create.DependsOn[0].Value)
// 	})

// 	t.Run("errors on unsupported create type", func(t *testing.T) {
// 		doc := &TSqlDocument{}
// 		s := NewScanner("test.sql", "create table [code].TestTable (id int)")
// 		s.NextToken()
// 		s.NextNonWhitespaceCommentToken()

// 		doc.parseCreate(s, 0)

// 		assert.NotEmpty(t, doc.errors)
// 		assert.Contains(t, doc.errors[0].Message, "only supports creating procedures")
// 	})

// 	t.Run("errors on multiple procedures in batch", func(t *testing.T) {
// 		doc := &TSqlDocument{}
// 		s := NewScanner("test.sql", "create procedure [code].Proc2 as begin end")
// 		s.NextToken()
// 		s.NextNonWhitespaceCommentToken()

// 		doc.parseCreate(s, 1)

// 		assert.NotEmpty(t, doc.errors)
// 		assert.Contains(t, doc.errors[0].Message, "must be alone in a batch")
// 	})

// 	t.Run("errors on missing code schema", func(t *testing.T) {
// 		doc := &TSqlDocument{}
// 		s := NewScanner("test.sql", "create procedure dbo.TestProc as begin end")
// 		s.NextToken()
// 		s.NextNonWhitespaceCommentToken()

// 		doc.parseCreate(s, 0)

// 		assert.NotEmpty(t, doc.errors)
// 		assert.Contains(t, doc.errors[0].Message, "must be followed by [code]")
// 	})

// 	t.Run("allows create index inside procedure", func(t *testing.T) {
// 		doc := &TSqlDocument{}
// 		s := NewScanner("test.sql", "create procedure [code].Proc as begin create index IX_Test on #temp(id) end")
// 		s.NextToken()
// 		s.NextNonWhitespaceCommentToken()

// 		create := doc.parseCreate(s, 0)

// 		assert.Equal(t, "procedure", create.CreateType)
// 		assert.Empty(t, doc.errors)
// 	})

// 	t.Run("stops at batch separator", func(t *testing.T) {
// 		doc := &TSqlDocument{}
// 		s := NewScanner("test.sql", "create procedure [code].Proc as begin end\ngo")
// 		s.NextToken()
// 		s.NextNonWhitespaceCommentToken()

// 		create := doc.parseCreate(s, 0)

// 		assert.Equal(t, "[Proc]", create.QuotedName.Value)
// 		assert.Equal(t, BatchSeparatorToken, s.TokenType())
// 	})

// 	t.Run("panics if not on create token", func(t *testing.T) {
// 		doc := &TSqlDocument{}
// 		s := NewScanner("test.sql", "procedure")
// 		s.NextToken()

// 		assert.Panics(t, func() {
// 			doc.parseCreate(s, 0)
// 		})
// 	})
// }

// func TestNextTokenCopyingWhitespace(t *testing.T) {
// 	t.Run("copies whitespace tokens", func(t *testing.T) {
// 		s := NewScanner("test.sql", "   \n\t  token")
// 		var target []Unparsed

// 		NextTokenCopyingWhitespace(s, &target)

// 		assert.NotEmpty(t, target)
// 		assert.Equal(t, UnquotedIdentifierToken, s.TokenType())
// 	})

// 	t.Run("copies comments", func(t *testing.T) {
// 		s := NewScanner("test.sql", "/* comment */ -- line\ntoken")
// 		var target []Unparsed

// 		NextTokenCopyingWhitespace(s, &target)

// 		assert.True(t, len(target) >= 2)
// 		assert.Equal(t, UnquotedIdentifierToken, s.TokenType())
// 	})

// 	t.Run("stops at EOF", func(t *testing.T) {
// 		s := NewScanner("test.sql", "   ")
// 		var target []Unparsed

// 		NextTokenCopyingWhitespace(s, &target)

// 		assert.Equal(t, EOFToken, s.TokenType())
// 	})
// }

// func TestCreateUnparsed(t *testing.T) {
// 	t.Run("creates unparsed from scanner", func(t *testing.T) {
// 		s := NewScanner("test.sql", "select")
// 		s.NextToken()

// 		unparsed := CreateUnparsed(s)

// 		assert.Equal(t, ReservedWordToken, unparsed.Type)
// 		assert.Equal(t, "select", unparsed.RawValue)
// 		assert.Equal(t, Pos{File: "test.sql", Line: 1, Col: 1}, unparsed.Start)
// 	})
// }

// func TestDocument_CreateProcedure(t *testing.T) {
// 	input := `
// CREATE PROCEDURE [code].[MyProc]
//     @Param1 nvarchar(100)
// AS
// BEGIN
//     SELECT @Param1
// END
// go
// `
// 	doc := parseDocument(input)

// 	if doc.HasErrors() {
// 		t.Fatalf("unexpected errors: %+v", doc.Errors())
// 	}

// 	creates := doc.Creates()
// 	if len(creates) != 1 {
// 		t.Fatalf("expected 1 create, got %d", len(creates))
// 	}

// 	c := creates[0]
// 	if c.CreateType != "procedure" {
// 		t.Errorf("expected createType 'procedure', got %q", c.CreateType)
// 	}
// 	if c.QuotedName.Value != "[MyProc]" {
// 		t.Errorf("expected name '[MyProc]', got %q", c.QuotedName.Value)
// 	}
// }

// func TestDocument_CreateFunction(t *testing.T) {
// 	input := `
// CREATE FUNCTION [code].[GetValue]
// (
//     @Input int
// )
// RETURNS int
// AS
// BEGIN
//     RETURN @Input * 2
// END
// go
// `
// 	doc := parseDocument(input)

// 	if doc.HasErrors() {
// 		t.Fatalf("unexpected errors: %+v", doc.Errors())
// 	}

// 	creates := doc.Creates()
// 	if len(creates) != 1 {
// 		t.Fatalf("expected 1 create, got %d", len(creates))
// 	}

// 	c := creates[0]
// 	if c.CreateType != "function" {
// 		t.Errorf("expected createType 'function', got %q", c.CreateType)
// 	}
// 	if c.QuotedName.Value != "[GetValue]" {
// 		t.Errorf("expected name '[GetValue]', got %q", c.QuotedName.Value)
// 	}
// }

// func TestDocument_CreateType(t *testing.T) {
// 	input := `
// CREATE TYPE [code].[MyTableType] AS TABLE
// (
//     Id int,
//     Name nvarchar(100)
// )
// go
// `
// 	doc := parseDocument(input)

// 	if doc.HasErrors() {
// 		t.Fatalf("unexpected errors: %+v", doc.Errors())
// 	}

// 	creates := doc.Creates()
// 	if len(creates) != 1 {
// 		t.Fatalf("expected 1 create, got %d", len(creates))
// 	}

// 	c := creates[0]
// 	if c.CreateType != "type" {
// 		t.Errorf("expected createType 'type', got %q", c.CreateType)
// 	}
// }

// func TestDocument_DeclareConstants(t *testing.T) {
// 	input := `
// DECLARE @EnumStatus int = 1,
//         @GlobalTimeout int = 30,
//         @ConstName nvarchar(50) = N'TestValue';
// go

// CREATE PROCEDURE [code].[MyProc]
// AS
// BEGIN
//     SELECT 1
// END
// go
// `
// 	doc := parseDocument(input)

// 	if doc.HasErrors() {
// 		t.Fatalf("unexpected errors: %+v", doc.Errors())
// 	}

// 	declares := doc.Declares()
// 	if len(declares) != 3 {
// 		t.Fatalf("expected 3 declares, got %d", len(declares))
// 	}

// 	// Check first declare
// 	if declares[0].VariableName != "@EnumStatus" {
// 		t.Errorf("expected '@EnumStatus', got %q", declares[0].VariableName)
// 	}
// 	if declares[0].Datatype.BaseType != "int" {
// 		t.Errorf("expected type 'int', got %q", declares[0].Datatype.BaseType)
// 	}

// 	// Check third declare with nvarchar type
// 	if declares[2].VariableName != "@ConstName" {
// 		t.Errorf("expected '@ConstName', got %q", declares[2].VariableName)
// 	}
// 	if declares[2].Datatype.BaseType != "nvarchar" {
// 		t.Errorf("expected type 'nvarchar', got %q", declares[2].Datatype.BaseType)
// 	}
// 	if len(declares[2].Datatype.Args) != 1 || declares[2].Datatype.Args[0] != "50" {
// 		t.Errorf("expected type args ['50'], got %v", declares[2].Datatype.Args)
// 	}
// }

// func TestDocument_Dependencies(t *testing.T) {
// 	input := `
// CREATE PROCEDURE [code].[ProcA]
// AS
// BEGIN
//     EXEC [code].[ProcB]
//     SELECT * FROM [code].[TableFunc]()
//     EXEC [code].[ProcB] -- duplicate reference
// END
// go
// `
// 	doc := parseDocument(input)

// 	if doc.HasErrors() {
// 		t.Fatalf("unexpected errors: %+v", doc.Errors())
// 	}

// 	creates := doc.Creates()
// 	if len(creates) != 1 {
// 		t.Fatalf("expected 1 create, got %d", len(creates))
// 	}

// 	deps := creates[0].DependsOn
// 	if len(deps) != 2 {
// 		t.Fatalf("expected 2 unique dependencies, got %d: %+v", len(deps), deps)
// 	}

// 	// Dependencies should be sorted
// 	if deps[0].Value != "[ProcB]" {
// 		t.Errorf("expected first dep '[ProcB]', got %q", deps[0].Value)
// 	}
// 	if deps[1].Value != "[TableFunc]" {
// 		t.Errorf("expected second dep '[TableFunc]', got %q", deps[1].Value)
// 	}
// }

// func TestDocument_MultipleBatches(t *testing.T) {
// 	input := `
// CREATE PROCEDURE [code].[ProcA]
// AS
// BEGIN
//     SELECT 1
// END
// go

// CREATE PROCEDURE [code].[ProcB]
// AS
// BEGIN
//     EXEC [code].[ProcA]
// END
// go

// CREATE FUNCTION [code].[FuncC]()
// RETURNS int
// AS
// BEGIN
//     RETURN 42
// END
// go
// `
// 	doc := parseDocument(input)

// 	if doc.HasErrors() {
// 		t.Fatalf("unexpected errors: %+v", doc.Errors())
// 	}

// 	creates := doc.Creates()
// 	if len(creates) != 3 {
// 		t.Fatalf("expected 3 creates, got %d", len(creates))
// 	}

// 	// Verify order preserved
// 	if creates[0].QuotedName.Value != "[ProcA]" {
// 		t.Errorf("expected first create '[ProcA]', got %q", creates[0].QuotedName.Value)
// 	}
// 	if creates[1].QuotedName.Value != "[ProcB]" {
// 		t.Errorf("expected second create '[ProcB]', got %q", creates[1].QuotedName.Value)
// 	}
// 	if creates[2].QuotedName.Value != "[FuncC]" {
// 		t.Errorf("expected third create '[FuncC]', got %q", creates[2].QuotedName.Value)
// 	}

// 	// ProcB should depend on ProcA
// 	if len(creates[1].DependsOn) != 1 || creates[1].DependsOn[0].Value != "[ProcA]" {
// 		t.Errorf("ProcB should depend on ProcA, got %+v", creates[1].DependsOn)
// 	}
// }

// func TestDocument_MultipleTypesInBatch(t *testing.T) {
// 	input := `
// CREATE TYPE [code].[Type1] AS TABLE (Id int)
// CREATE TYPE [code].[Type2] AS TABLE (Name nvarchar(50))
// CREATE TYPE [code].[Type3] AS TABLE (Value decimal(10,2))
// go
// `
// 	doc := parseDocument(input)

// 	if doc.HasErrors() {
// 		t.Fatalf("unexpected errors: %+v", doc.Errors())
// 	}

// 	creates := doc.Creates()
// 	if len(creates) != 3 {
// 		t.Fatalf("expected 3 type creates, got %d", len(creates))
// 	}

// 	for i, c := range creates {
// 		if c.CreateType != "type" {
// 			t.Errorf("create %d: expected type 'type', got %q", i, c.CreateType)
// 		}
// 	}
// }

// func TestDocument_ErrorInvalidDeclarePrefix(t *testing.T) {
// 	input := `
// DECLARE @InvalidName int = 1;
// go
// `
// 	doc := parseDocument(input)

// 	if !doc.HasErrors() {
// 		t.Fatal("expected error for invalid variable name prefix")
// 	}

// 	errors := doc.Errors()
// 	found := false
// 	for _, err := range errors {
// 		if strings.Contains(err.Message, "@Enum") || strings.Contains(err.Message, "@Global") || strings.Contains(err.Message, "@Const") {
// 			found = true
// 			break
// 		}
// 	}
// 	if !found {
// 		t.Errorf("expected error about variable name prefix, got: %+v", errors)
// 	}
// }

// func TestDocument_ErrorMissingCodeSchema(t *testing.T) {
// 	input := `
// CREATE PROCEDURE [dbo].[MyProc]
// AS
// BEGIN
//     SELECT 1
// END
// go
// `
// 	doc := parseDocument(input)

// 	if !doc.HasErrors() {
// 		t.Fatal("expected error for missing [code] schema")
// 	}

// 	errors := doc.Errors()
// 	found := false
// 	for _, err := range errors {
// 		if strings.Contains(err.Message, "[code]") {
// 			found = true
// 			break
// 		}
// 	}
// 	if !found {
// 		t.Errorf("expected error about [code] schema, got: %+v", errors)
// 	}
// }

// func TestDocument_ErrorProcedureNotAloneInBatch(t *testing.T) {
// 	input := `
// CREATE PROCEDURE [code].[ProcA]
// AS
// BEGIN
//     SELECT 1
// END

// CREATE PROCEDURE [code].[ProcB]
// AS
// BEGIN
//     SELECT 2
// END
// go
// `
// 	doc := parseDocument(input)

// 	if !doc.HasErrors() {
// 		t.Fatal("expected error for multiple procedures in one batch")
// 	}

// 	errors := doc.Errors()
// 	found := false
// 	for _, err := range errors {
// 		if strings.Contains(err.Message, "alone in a batch") {
// 			found = true
// 			break
// 		}
// 	}
// 	if !found {
// 		t.Errorf("expected error about procedure alone in batch, got: %+v", errors)
// 	}
// }

// func TestDocument_ErrorDeclareInSecondBatch(t *testing.T) {
// 	input := `
// CREATE PROCEDURE [code].[MyProc]
// AS
// BEGIN
//     SELECT 1
// END
// go

// DECLARE @EnumValue int = 1;
// go
// `
// 	doc := parseDocument(input)

// 	if !doc.HasErrors() {
// 		t.Fatal("expected error for declare in second batch")
// 	}

// 	errors := doc.Errors()
// 	found := false
// 	for _, err := range errors {
// 		if strings.Contains(err.Message, "first batch") {
// 			found = true
// 			break
// 		}
// 	}
// 	if !found {
// 		t.Errorf("expected error about first batch, got: %+v", errors)
// 	}
// }

// func TestDocument_Pragma(t *testing.T) {
// 	input := `--sqlcode:include-if feature-flag
// CREATE PROCEDURE [code].[MyProc]
// AS
// BEGIN
//     SELECT 1
// END
// go
// `
// 	doc := parseDocument(input)

// 	if doc.HasErrors() {
// 		t.Fatalf("unexpected errors: %+v", doc.Errors())
// 	}

// 	creates := doc.Creates()
// 	if len(creates) != 1 {
// 		t.Fatalf("expected 1 create, got %d", len(creates))
// 	}
// }

// func TestDocument_ComplexProcedure(t *testing.T) {
// 	input := `
// -- This procedure demonstrates complex T-SQL features
// CREATE PROCEDURE [code].[ComplexProc]
//     @TableInput [code].[MyTableType] READONLY,
//     @Status int OUTPUT,
//     @Message nvarchar(max) OUTPUT
// AS
// BEGIN
//     SET NOCOUNT ON;

//     BEGIN TRY
//         BEGIN TRANSACTION;

//         -- Use CTE
//         WITH CTE AS (
//             SELECT Id, Name, ROW_NUMBER() OVER (ORDER BY Id) AS RowNum
//             FROM @TableInput
//         )
//         INSERT INTO SomeTable (Id, Name)
//         SELECT Id, Name FROM CTE WHERE RowNum <= 100;

//         -- Call another procedure
//         EXEC [code].[HelperProc] @Status OUTPUT;

//         -- Use table-valued function
//         SELECT * FROM [code].[GetItems](@Status);

//         COMMIT TRANSACTION;
//         SET @Message = N'Success';
//     END TRY
//     BEGIN CATCH
//         IF @@TRANCOUNT > 0
//             ROLLBACK TRANSACTION;

//         SET @Status = ERROR_NUMBER();
//         SET @Message = ERROR_MESSAGE();
//     END CATCH
// END
// go
// `
// 	doc := parseDocument(input)

// 	if doc.HasErrors() {
// 		t.Fatalf("unexpected errors: %+v", doc.Errors())
// 	}

// 	creates := doc.Creates()
// 	if len(creates) != 1 {
// 		t.Fatalf("expected 1 create, got %d", len(creates))
// 	}

// 	c := creates[0]
// 	if c.CreateType != "procedure" {
// 		t.Errorf("expected 'procedure', got %q", c.CreateType)
// 	}

// 	// Should have dependencies on HelperProc, GetItems, and MyTableType
// 	if len(c.DependsOn) != 3 {
// 		t.Errorf("expected 3 dependencies, got %d: %+v", c.DependsOn)
// 	}

// 	depNames := make(map[string]bool)
// 	for _, dep := range c.DependsOn {
// 		depNames[dep.Value] = true
// 	}

// 	expectedDeps := []string{"[GetItems]", "[HelperProc]", "[MyTableType]"}
// 	for _, exp := range expectedDeps {
// 		if !depNames[exp] {
// 			t.Errorf("missing expected dependency %s", exp)
// 		}
// 	}
// }

// func TestDocument_WithoutPos(t *testing.T) {
// 	input := `
// DECLARE @EnumValue int = 1;
// go
// CREATE PROCEDURE [code].[MyProc]
// AS
// SELECT 1
// go
// `
// 	doc := parseDocument(input)
// 	docWithoutPos := doc.WithoutPos().(*TSqlDocument)

// 	// Verify positions are zeroed
// 	for _, c := range docWithoutPos.Creates() {
// 		if c.QuotedName.Pos.Line != 0 {
// 			t.Error("expected zero position in WithoutPos result")
// 		}
// 	}
// 	for _, d := range docWithoutPos.Declares() {
// 		if d.Start.Line != 0 {
// 			t.Error("expected zero position in WithoutPos result")
// 		}
// 	}
// }

// func TestDocument_Include(t *testing.T) {
// 	input1 := `
// CREATE PROCEDURE [code].[ProcA]
// AS
// SELECT 1
// go
// `
// 	input2 := `
// CREATE PROCEDURE [code].[ProcB]
// AS
// SELECT 2
// go
// `
// 	doc1 := parseDocument(input1)
// 	doc2 := parseDocument(input2)

// 	doc1.Include(doc2)

// 	creates := doc1.Creates()
// 	if len(creates) != 2 {
// 		t.Fatalf("expected 2 creates after Include, got %d", len(creates))
// 	}
// }

// func TestDocument_Sort(t *testing.T) {
// 	// ProcB depends on ProcA, so ProcA should come first after sort
// 	input := `
// CREATE PROCEDURE [code].[ProcB]
// AS
// BEGIN
//     EXEC [code].[ProcA]
// END
// go

// CREATE PROCEDURE [code].[ProcA]
// AS
// SELECT 1
// go
// `
// 	doc := parseDocument(input)
// 	doc.Sort()

// 	if doc.HasErrors() {
// 		t.Fatalf("unexpected errors after sort: %+v", doc.Errors())
// 	}

// 	creates := doc.Creates()
// 	if len(creates) != 2 {
// 		t.Fatalf("expected 2 creates, got %d", len(creates))
// 	}

// 	// After topological sort, ProcA should come before ProcB
// 	if creates[0].QuotedName.Value != "[ProcA]" {
// 		t.Errorf("expected ProcA first after sort, got %q", creates[0].QuotedName.Value)
// 	}
// 	if creates[1].QuotedName.Value != "[ProcB]" {
// 		t.Errorf("expected ProcB second after sort, got %q", creates[1].QuotedName.Value)
// 	}
// }

// func TestDocument_SortCyclicDependency(t *testing.T) {
// 	// Create a cycle: A -> B -> C -> A
// 	input := `
// CREATE PROCEDURE [code].[ProcA]
// AS
// EXEC [code].[ProcC]
// go

// CREATE PROCEDURE [code].[ProcB]
// AS
// EXEC [code].[ProcA]
// go

// CREATE PROCEDURE [code].[ProcC]
// AS
// EXEC [code].[ProcB]
// go
// `
// 	doc := parseDocument(input)
// 	doc.Sort()

// 	// Should have an error about cyclic dependency
// 	if !doc.HasErrors() {
// 		t.Fatal("expected error for cyclic dependency")
// 	}

// 	found := false
// 	for _, err := range doc.Errors() {
// 		if strings.Contains(strings.ToLower(err.Message), "cycle") ||
// 			strings.Contains(strings.ToLower(err.Message), "circular") {
// 			found = true
// 			break
// 		}
// 	}
// 	if !found {
// 		t.Errorf("expected cycle-related error, got: %+v", doc.Errors())
// 	}
// }

// func TestDocument_NewScanner(t *testing.T) {
// 	doc := &TSqlDocument{}
// 	input := "SELECT 1"
// 	file := sqldocument.FileRef("test.sql")

// 	scanner := doc.NewScanner(input, file)

// 	if scanner == nil {
// 		t.Fatal("NewScanner returned nil")
// 	}

// 	scanner.NextToken()
// 	if scanner.TokenType() != sqldocument.ReservedWordToken {
// 		t.Errorf("expected ReservedWordToken for SELECT, got %v", scanner.TokenType())
// 	}
// }

// func TestDocument_Empty(t *testing.T) {
// 	emptyDoc := &TSqlDocument{}
// 	if !emptyDoc.Empty() {
// 		t.Error("empty document should report Empty() = true")
// 	}

// 	input := `
// CREATE PROCEDURE [code].[MyProc]
// AS
// SELECT 1
// go
// `
// 	doc := parseDocument(input)
// 	// Note: Empty() returns true if EITHER creates or declares is empty (uses ||)
// 	// This might be a bug in the original code - should probably be &&
// 	// Testing current behavior
// 	if doc.Empty() {
// 		// Has creates but no declares, so with || it would be true
// 		// Let's check the actual logic
// 		t.Log("Empty() behavior may need review - currently uses || instead of &&")
// 	}
// }

// func TestDocument_UnicodeIdentifiers(t *testing.T) {
// 	input := `
// CREATE PROCEDURE [code].[日本語プロシージャ]
//     @パラメータ nvarchar(100)
// AS
// BEGIN
//     SELECT @パラメータ
// END
// go
// `
// 	doc := parseDocument(input)

// 	if doc.HasErrors() {
// 		t.Fatalf("unexpected errors: %+v", doc.Errors())
// 	}

// 	creates := doc.Creates()
// 	if len(creates) != 1 {
// 		t.Fatalf("expected 1 create, got %d", len(creates))
// 	}

// 	if creates[0].QuotedName.Value != "[日本語プロシージャ]" {
// 		t.Errorf("expected Unicode name, got %q", creates[0].QuotedName.Value)
// 	}
// }

// func TestDocument_NestedCreateStatements(t *testing.T) {
// 	// Procedure containing CREATE TABLE (should be allowed)
// 	input := `
// CREATE PROCEDURE [code].[MyProc]
// AS
// BEGIN
//     CREATE TABLE #TempTable (Id int)
//     INSERT INTO #TempTable SELECT 1
//     SELECT * FROM #TempTable
//     DROP TABLE #TempTable
// END
// go
// `
// 	doc := parseDocument(input)

// 	if doc.HasErrors() {
// 		t.Fatalf("unexpected errors: %+v", doc.Errors())
// 	}

// 	creates := doc.Creates()
// 	if len(creates) != 1 {
// 		t.Fatalf("expected 1 create (procedure only), got %d", len(creates))
// 	}
// }

// func TestDocument_TypeWithMaxArg(t *testing.T) {
// 	input := `
// DECLARE @ConstValue nvarchar(max) = N'test';
// go
// `
// 	doc := parseDocument(input)

// 	if doc.HasErrors() {
// 		t.Fatalf("unexpected errors: %+v", doc.Errors())
// 	}

// 	declares := doc.Declares()
// 	if len(declares) != 1 {
// 		t.Fatalf("expected 1 declare, got %d", len(declares))
// 	}

// 	if declares[0].Datatype.BaseType != "nvarchar" {
// 		t.Errorf("expected 'nvarchar', got %q", declares[0].Datatype.BaseType)
// 	}
// 	if len(declares[0].Datatype.Args) != 1 || declares[0].Datatype.Args[0] != "max" {
// 		t.Errorf("expected ['max'], got %v", declares[0].Datatype.Args)
// 	}
// }

// func TestDocument_TypeWithMultipleArgs(t *testing.T) {
// 	input := `
// DECLARE @ConstValue decimal(18,4) = 123.4567;
// go
// `
// 	doc := parseDocument(input)

// 	if doc.HasErrors() {
// 		t.Fatalf("unexpected errors: %+v", doc.Errors())
// 	}

// 	declares := doc.Declares()
// 	if len(declares) != 1 {
// 		t.Fatalf("expected 1 declare, got %d", len(declares))
// 	}

// 	dt := declares[0].Datatype
// 	if dt.BaseType != "decimal" {
// 		t.Errorf("expected 'decimal', got %q", dt.BaseType)
// 	}
// 	if len(dt.Args) != 2 || dt.Args[0] != "18" || dt.Args[1] != "4" {
// 		t.Errorf("expected ['18', '4'], got %v", dt.Args)
// 	}
// }
