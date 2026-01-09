package mssql

import (
	"strings"
	"testing"

	mssql "github.com/microsoft/go-mssqldb"
	"github.com/simukka/sqlcode/sqlparser/sqldocument"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	assert.Equal(t, 1, len(doc.Creates()))

	c := doc.Creates()[0]
	assert.Equal(t, &mssql.Driver{}, c.Driver)
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

	assert.Equal(t, "'declare' statement only allowed in first batch", doc.Errors()[0].Message)

	assert.Equal(t, "@EnumBar1", doc.Declares()[0].VariableName)
	assert.Equal(t, "N'declare @EnumThisIsInString'", doc.Declares()[0].Literal.RawValue)
	assert.Equal(t, NVarcharLiteralToken, doc.Declares()[0].Literal.Type)
	assert.Equal(t, "varchar", doc.Declares()[0].Datatype.BaseType)
	assert.Equal(t, "max", doc.Declares()[0].Datatype.Args[0])

	assert.Equal(t, "@EnumNextBatch", doc.Declares()[3].VariableName)
	assert.Equal(t, "int", doc.Declares()[3].Datatype.BaseType)
	assert.Equal(t, sqldocument.NumberToken, doc.Declares()[3].Literal.Type)
	assert.Equal(t, "3", doc.Declares()[3].Literal.RawValue)
}

func TestParserDisallowMultipleCreates(t *testing.T) {
	// Test that we get an error if we create two things in same batch;
	// the test above tests that it is still OK to create a table within
	// a procedure..
	doc := ParseString(t, "test.sql", `
create function [code].One();
-- the following should give an error; not that One() depends on Two()...
-- (we don't parse body start/end yet)
create function [code].Two();
`)

	assert.Equal(t, "a procedure/function must be alone in a batch; use 'go' to split batches", doc.Errors()[0].Message)
}

func TestBuggyDeclare(t *testing.T) {
	// this caused parses to infinitely loop; regression test...
	doc := ParseString(t, "test.sql", `declare    @EnumA int = 4    @EnumB tinyint = 5    @ENUM_C bigint = 435;`)
	assert.Equal(t, 1, len(doc.Errors()))
	assert.Equal(t, "Unexpected: @EnumB", doc.Errors()[0].Message)
}

func TestCreateType(t *testing.T) {
	doc := ParseString(t, "test.sql", `create type [code].MyType as table (x int not null primary key);`)
	assert.Equal(t, 1, len(doc.Creates()))
	assert.Equal(t, sqldocument.SQLType, doc.Creates()[0].CreateType)
	assert.Equal(t, "[MyType]", doc.Creates()[0].QuotedName.Value)
}

func TestPragma(t *testing.T) {
	doc := ParseString(t, "test.sql", `--sqlcode:include-if one,two
--sqlcode:include-if three

create procedure [code].ProcedureShouldAlsoHavePragmasAnnotated()
`)
	assert.Equal(t, []string{"one", "two", "three"}, doc.PragmaIncludeIf())
}

func TestInfiniteLoopRegression(t *testing.T) {
	// success if we terminate!...
	doc := ParseString(t, "test.sql", `@declare`)
	assert.Equal(t, 1, len(doc.Errors()))
}

func TestDeclareSeparation(t *testing.T) {
	// Trying out many possible ways to separate declare statements:
	// Comman, semicolon, simply starting a new declare with or without
	// whitespace in between.
	// Yes, ='hello'declare @EnumThird really does parse as T-SQL
	doc := ParseString(t, "test.sql", `
declare @EnumFirst int = 3, @EnumSecond varchar(max) = 'hello'declare @EnumThird int=3 declare @EnumFourth int=4;declare @EnumFifth int =5
`)
	assert.Len(t, doc.Declares(), 5)
	assert.Equal(t, "@EnumFirst", doc.Declares()[0].VariableName)
	assert.Equal(t, "int", doc.Declares()[0].Datatype.BaseType)
	assert.Equal(t, sqldocument.NumberToken, doc.Declares()[0].Literal.Type)
	assert.Equal(t, "3", doc.Declares()[0].Literal.RawValue)

}

func TestBatchDivisionsAndCreateStatements(t *testing.T) {
	// Had a bug where comments where repeated on each create statement in different batches, discovery & regression
	// (The bug was that a token too much was consumed in parseCreate, consuming the `go` token..)
	doc := ParseString(t, "test.sql", `
create type [code].Batch1 as table (x int);
go
-- a comment in 2nd batch
create procedure [code].Batch2 as table (x int);
go
create type [code].Batch3 as table (x int);
`)
	commentCount := 0
	for _, c := range doc.Creates() {
		for _, b := range c.Body {
			if strings.Contains(b.RawValue, "2nd") {
				commentCount++
			}
			assert.NotEqual(t, "go", b.RawValue)
		}
	}
	assert.Equal(t, 1, commentCount)
}

func TestCreateTypes(t *testing.T) {
	// Apparently there can be several 'create type' per batch, but only one function/procedure...
	// Check we catch all 3 types
	doc := ParseString(t, "test.sql", `
create type [code].Type1 as table (x int);
create type [code].Type2 as table (x int);
create type [code].Type3 as table (x int);
`)
	assert.Len(t, doc.Creates(), 3)
	assert.Equal(t, "[Type1]", doc.Creates()[0].QuotedName.Value)
	assert.Equal(t, "[Type3]", doc.Creates()[2].QuotedName.Value)
	// There was a bug that the last item in the body would be the 'create'
	// of the next statement; regression test..
	assert.Equal(t, "\n", doc.Creates()[0].Body[len(doc.Creates()[0].Body)-1].RawValue)
	assert.Equal(t, "create", doc.Creates()[1].Body[0].RawValue)
}

func TestCreateProcs(t *testing.T) {
	// Apparently there can be several 'create type' per batch, but only one function/procedure...
	// Check that we get an error for all further create statements in the same batch
	doc := ParseString(t, "test.sql", `
create procedure [code].FirstProc as table (x int)
create function [code].MyFunction ()
create type [code].MyType ()
create procedure [code].MyProcedure ()
`)

	// First function and last procedure triggers errors.
	assert.Len(t, doc.Errors(), 2)
	emsg := "a procedure/function must be alone in a batch; use 'go' to split batches"
	assert.Equal(t, emsg, doc.Errors()[0].Message)
	assert.Equal(t, emsg, doc.Errors()[1].Message)

}

func TestCreateProcs2(t *testing.T) {
	// Create type first, then create proc... should give an error still..
	doc := ParseString(t, "test.sql", `
create type [code].MyType ()
create procedure [code].FirstProc as table (x int)
`)
	// Code above was mainly to be able to step through parser in a given way.
	// First function triggers an error. Then create type is parsed which is
	// fine sharing a batch with others.
	require.Equal(t, 1, len(doc.Errors()))
	emsg := "a procedure/function must be alone in a batch; use 'go' to split batches"
	assert.Equal(t, emsg, doc.Errors()[0].Message)
}

func TestGoWithoutNewline(t *testing.T) {
	doc := ParseString(t, "test.sql", `
create procedure [code].Foo() as begin
end;
go create function [code].Bar() returns int as begin
end
`)
	// Code above was mainly to be able to step through parser in a given way.
	// First function triggers an error. Then create type is parsed which is
	// fine sharing a batch with others.
	require.Equal(t, 1, len(doc.Errors()))
	assert.Equal(t, "`go` should be alone on a line without any comments", doc.Errors()[0].Message)
}

func TestCreateAnnotationHappyDay(t *testing.T) {
	// Comment / annotations on create statements
	doc := ParseString(t, "test.sql", `
-- Not part of annotation
--! key4: 1

-- This is part of annotation
--! key1: a
--! key2: b
--! key3: [1,2,3]
create procedure [code].Foo as begin end

`)
	assert.Equal(t,
		"-- This is part of annotation\n--! key1: a\n--! key2: b\n--! key3: [1,2,3]",
		doc.Creates()[0].DocstringAsString())
	s, err := doc.Creates()[0].DocstringYamldoc()
	assert.NoError(t, err)
	assert.Equal(t,
		"key1: a\nkey2: b\nkey3: [1,2,3]",
		s)

	var x struct {
		Key1 string `yaml:"key1"`
	}
	require.NoError(t, doc.Creates()[0].ParseYamlInDocstring(&x))
	assert.Equal(t, "a", x.Key1)
}

func TestCreateAnnotationAfterPragma(t *testing.T) {
	// Comment / annotations on create statement, with pragma at start of file
	doc := ParseString(t, "test.sql", `
--sqlcode: include-if foo

-- docstring here
create procedure [code].Foo as begin end

`)
	assert.Equal(t,
		"-- docstring here",
		doc.Creates()[0].DocstringAsString())
}

func TestCreateAnnotationErrors(t *testing.T) {
	// Multiple embedded yaml documents ..
	doc := ParseString(t, "test.sql", `
--! key4: 1
-- This comment after yamldoc is illegal; this also prevents multiple embedded YAML documents
create procedure [code].Foo as begin end
`)
	_, err := doc.Creates()[0].DocstringYamldoc()
	assert.Equal(t, "test.sql:3:1 once embedded yaml document is started (lines prefixed with `--!`), it must continue until create statement",
		err.Error())

	// No whitespace after !
	doc = ParseString(t, "test.sql", `
-- Docstring here
--!key4: 1
create procedure [code].Foo as begin end
`)
	_, err = doc.Creates()[0].DocstringYamldoc()
	assert.Equal(t, "test.sql:3:1 YAML document in docstring; missing space after `--!`",
		err.Error())

}
