package sqlparser

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

func TestParserSmokeTest(t *testing.T) {
	doc := ParseString("test.sql", `
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
	docNoPos := doc.WithoutPos()

	require.Equal(t, 1, len(doc.Creates))
	c := doc.Creates[0]

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
		[]Error{
			{
				Message: "'declare' statement only allowed in first batch",
			},
		}, docNoPos.Errors)

	assert.Equal(t,
		[]Declare{
			{
				VariableName: "@EnumBar1",
				Datatype: Type{
					BaseType: "varchar",
					Args: []string{
						"max",
					},
				},
				Literal: Unparsed{
					Type:     NVarcharLiteralToken,
					RawValue: "N'declare @EnumThisIsInString'",
				},
			},
			{
				VariableName: "@EnumBar2",
				Datatype: Type{
					BaseType: "int",
				},
				Literal: Unparsed{
					Type:     NumberToken,
					RawValue: "20",
				},
			},
			{
				VariableName: "@EnumBar3",
				Datatype: Type{
					BaseType: "int",
				},
				Literal: Unparsed{
					Type:     NumberToken,
					RawValue: "21",
				},
			},
			{
				VariableName: "@EnumNextBatch",
				Datatype: Type{
					BaseType: "int",
				},
				Literal: Unparsed{
					Type:     NumberToken,
					RawValue: "3",
				},
			},
		},
		docNoPos.Declares,
	)
	//	repr.Println(doc)
}

func TestParserDisallowMultipleCreates(t *testing.T) {
	// Test that we get an error if we create two things in same batch;
	// the test above tests that it is still OK to create a table within
	// a procedure..
	doc := ParseString("test.sql", `
create function [code].One();
-- the following should give an error; not that One() depends on Two()...
-- (we don't parse body start/end yet)
create function [code].Two();
`).WithoutPos()

	assert.Equal(t, []Error{
		{
			Message: "a procedure/function must be alone in a batch; use 'go' to split batches",
		},
	}, doc.Errors)
}

func TestBuggyDeclare(t *testing.T) {
	// this caused parses to infinitely loop; regression test...
	doc := ParseString("test.sql", `declare    @EnumA int = 4    @EnumB tinyint = 5    @ENUM_C bigint = 435;`)
	assert.Equal(t, 1, len(doc.Errors))
	assert.Equal(t, "Unexpected: @EnumB", doc.Errors[0].Message)
}

func TestCreateType(t *testing.T) {
	doc := ParseString("test.sql", `create type [code].MyType as table (x int not null primary key);`)
	assert.Equal(t, 1, len(doc.Creates))
	assert.Equal(t, "type", doc.Creates[0].CreateType)
	assert.Equal(t, "[MyType]", doc.Creates[0].QuotedName.Value)
}

func TestPragma(t *testing.T) {
	doc := ParseString("test.sql", `--sqlcode:include-if one,two
--sqlcode:include-if three

create procedure [code].ProcedureShouldAlsoHavePragmasAnnotated()
`)
	assert.Equal(t, []string{"one", "two", "three"}, doc.PragmaIncludeIf)
}

func TestInfiniteLoopRegression(t *testing.T) {
	// success if we terminate!...
	doc := ParseString("test.sql", `@declare`)
	assert.Equal(t, 1, len(doc.Errors))
}

func TestDeclareSeparation(t *testing.T) {
	// Trying out many possible ways to separate declare statements:
	// Comman, semicolon, simply starting a new declare with or without
	// whitespace in between.
	// Yes, ='hello'declare @EnumThird really does parse as T-SQL
	doc := ParseString("test.sql", `
declare @EnumFirst int = 3, @EnumSecond varchar(max) = 'hello'declare @EnumThird int=3 declare @EnumFourth int=4;declare @EnumFifth int =5
`)
	//repr.Println(doc.Declares)
	require.Equal(t, []Declare{
		{
			VariableName: "@EnumFirst",
			Datatype:     Type{BaseType: "int"},
			Literal:      Unparsed{Type: NumberToken, RawValue: "3"},
		},
		{
			VariableName: "@EnumSecond",
			Datatype:     Type{BaseType: "varchar", Args: []string{"max"}},
			Literal:      Unparsed{Type: VarcharLiteralToken, RawValue: "'hello'"},
		},
		{
			VariableName: "@EnumThird",
			Datatype:     Type{BaseType: "int"},
			Literal:      Unparsed{Type: NumberToken, RawValue: "3"},
		},
		{
			VariableName: "@EnumFourth",
			Datatype:     Type{BaseType: "int"},
			Literal:      Unparsed{Type: NumberToken, RawValue: "4"},
		},
		{
			VariableName: "@EnumFifth",
			Datatype:     Type{BaseType: "int"},
			Literal:      Unparsed{Type: NumberToken, RawValue: "5"},
		},
	}, doc.WithoutPos().Declares)
}

func TestBatchDivisionsAndCreateStatements(t *testing.T) {
	// Had a bug where comments where repeated on each create statement in different batches, discovery & regression
	// (The bug was that a token too much was consumed in parseCreate, consuming the `go` token..)
	doc := ParseString("test.sql", `
create type [code].Batch1 as table (x int);
go
-- a comment in 2nd batch
create procedure [code].Batch2 as table (x int);
go
create type [code].Batch3 as table (x int);
`)
	commentCount := 0
	for _, c := range doc.Creates {
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
	doc := ParseString("test.sql", `
create type [code].Type1 as table (x int);
create type [code].Type2 as table (x int);
create type [code].Type3 as table (x int);
`)
	require.Equal(t, 3, len(doc.Creates))
	assert.Equal(t, "[Type1]", doc.Creates[0].QuotedName.Value)
	assert.Equal(t, "[Type3]", doc.Creates[2].QuotedName.Value)
	// There was a bug that the last item in the body would be the 'create'
	// of the next statement; regression test..
	assert.Equal(t, "\n", doc.Creates[0].Body[len(doc.Creates[0].Body)-1].RawValue)
	assert.Equal(t, "create", doc.Creates[1].Body[0].RawValue)
}

func TestCreateProcs(t *testing.T) {
	// Apparently there can be several 'create type' per batch, but only one function/procedure...
	// Check that we get an error for all further create statements in the same batch
	doc := ParseString("test.sql", `
create procedure [code].FirstProc as table (x int)
create function [code].MyFunction ()
create type [code].MyType ()
create procedure [code].MyProcedure ()
`)
	// First function and last procedure triggers errors.
	require.Equal(t, 2, len(doc.Errors))
	emsg := "a procedure/function must be alone in a batch; use 'go' to split batches"
	assert.Equal(t, emsg, doc.Errors[0].Message)
	assert.Equal(t, emsg, doc.Errors[1].Message)

}

func TestCreateProcs2(t *testing.T) {
	// Create type first, then create proc... should give an error still..
	doc := ParseString("test.sql", `
create type [code].MyType ()
create procedure [code].FirstProc as table (x int)
`)
	//repr.Println(doc.Errors)

	// Code above was mainly to be able to step through parser in a given way.
	// First function triggers an error. Then create type is parsed which is
	// fine sharing a batch with others.
	require.Equal(t, 1, len(doc.Errors))
	emsg := "a procedure/function must be alone in a batch; use 'go' to split batches"
	assert.Equal(t, emsg, doc.Errors[0].Message)
}
