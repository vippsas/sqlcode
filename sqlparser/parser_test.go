package sqlparser

import (
	"fmt"
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/jackc/pgx/v5/stdlib"
	mssql "github.com/microsoft/go-mssqldb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPostgresqlCreate(t *testing.T) {
	doc := ParseString("test.pgsql", `
create procedure [code].test()
language plpgsql
as $$
begin
    perform 1;
end;
$$;
	`)

	require.Len(t, doc.Creates, 1)
	require.Equal(t, &stdlib.Driver{}, doc.Creates[0].Driver)
}

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

func TestCreateProcsAndCheckForRoutineName(t *testing.T) {
	t.Skip() // Routine name is disabled for now
	testcases := []struct {
		name             string
		doc              Document
		expectedProcName string
		expectedIndex    int
	}{
		{
			name:             "Test simple proc",
			expectedProcName: "FirstProc",
			doc: ParseString("test.sql", `
create procedure [code].FirstProc as
begin
end
`),
			expectedIndex: 10,
		},
		{
			name:             "Test proc with args",
			expectedProcName: "transform:safeguarding.Calculation/HEAD",
			doc: ParseString("test.sql", `
create procedure [code].[transform:safeguarding.Calculation/HEAD](@now datetime2,
@count bigint output) as
`),
			expectedIndex: 22,
		},
	}
	for _, tc := range testcases {
		require.Equal(t, 0, len(tc.doc.Errors))
		assert.Len(t, tc.doc.Creates, 1)
		assert.Greater(t, len(tc.doc.Creates[0].Body), tc.expectedIndex)
		assert.Equal(t,
			fmt.Sprintf(templateRoutineName, tc.expectedProcName),
			tc.doc.Creates[0].Body[tc.expectedIndex].RawValue,
		)
	}
}

func TestGoWithoutNewline(t *testing.T) {
	doc := ParseString("test.sql", `
create procedure [code].Foo() as begin
end;
go create function [code].Bar() returns int as begin
end
`)
	// Code above was mainly to be able to step through parser in a given way.
	// First function triggers an error. Then create type is parsed which is
	// fine sharing a batch with others.
	require.Equal(t, 2, len(doc.Errors))
	assert.Equal(t, "`go` should be alone on a line without any comments", doc.Errors[0].Message)
	assert.Equal(t, "Expected 'declare' or 'create', got: end", doc.Errors[1].Message)
}

func TestCreateAnnotationHappyDay(t *testing.T) {
	// Comment / annotations on create statements
	doc := ParseString("test.sql", `
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
		doc.Creates[0].DocstringAsString())
	s, err := doc.Creates[0].DocstringYamldoc()
	assert.NoError(t, err)
	assert.Equal(t,
		"key1: a\nkey2: b\nkey3: [1,2,3]",
		s)

	var x struct {
		Key1 string `yaml:"key1"`
	}
	require.NoError(t, doc.Creates[0].ParseYamlInDocstring(&x))
	assert.Equal(t, "a", x.Key1)
}

func TestCreateAnnotationAfterPragma(t *testing.T) {
	// Comment / annotations on create statement, with pragma at start of file
	doc := ParseString("test.sql", `
--sqlcode: include-if foo

-- docstring here
create procedure [code].Foo as begin end

`)
	assert.Equal(t,
		"-- docstring here",
		doc.Creates[0].DocstringAsString())
}

func TestCreateAnnotationErrors(t *testing.T) {
	// Multiple embedded yaml documents ..
	doc := ParseString("test.sql", `
--! key4: 1
-- This comment after yamldoc is illegal; this also prevents multiple embedded YAML documents
create procedure [code].Foo as begin end
`)
	_, err := doc.Creates[0].DocstringYamldoc()
	assert.Equal(t, "test.sql:3:1 once embedded yaml document is started (lines prefixed with `--!`), it must continue until create statement",
		err.Error())

	// No whitespace after !
	doc = ParseString("test.sql", `
-- Docstring here
--!key4: 1
create procedure [code].Foo as begin end
`)
	_, err = doc.Creates[0].DocstringYamldoc()
	assert.Equal(t, "test.sql:3:1 YAML document in docstring; missing space after `--!`",
		err.Error())

}

func TestParseFilesystems(t *testing.T) {
	t.Run("basic parsing of sql files", func(t *testing.T) {
		fsys := fstest.MapFS{
			"test1.sql": &fstest.MapFile{
				Data: []byte(`
declare @EnumFoo int = 1;
go
create procedure [code].Proc1 as begin end
`),
			},
			"test2.sql": &fstest.MapFile{
				Data: []byte(`
create function [code].Func1() returns int as begin return 1 end
`),
			},
		}

		filenames, doc, err := ParseFilesystems([]fs.FS{fsys}, nil)
		require.NoError(t, err)
		assert.Len(t, filenames, 2)
		assert.Len(t, doc.Creates, 2)
		assert.Len(t, doc.Declares, 1)
	})

	t.Run("filters by include tags", func(t *testing.T) {
		fsys := fstest.MapFS{
			"included.sql": &fstest.MapFile{
				Data: []byte(`--sqlcode:include-if foo,bar
create procedure [code].Included as begin end
`),
			},
			"excluded.sql": &fstest.MapFile{
				Data: []byte(`--sqlcode:include-if baz
create procedure [code].Excluded as begin end
`),
			},
		}

		filenames, doc, err := ParseFilesystems([]fs.FS{fsys}, []string{"foo", "bar"})
		require.NoError(t, err)
		assert.Len(t, filenames, 1)
		assert.Contains(t, filenames[0], "included.sql")
		assert.Len(t, doc.Creates, 1)
		assert.Equal(t, "[Included]", doc.Creates[0].QuotedName.Value)
	})

	t.Run("detects duplicate files with same hash", func(t *testing.T) {
		contents := []byte(`create procedure [code].Test as begin end`)

		fs1 := fstest.MapFS{
			"test.sql": &fstest.MapFile{Data: contents},
		}
		fs2 := fstest.MapFS{
			"test.sql": &fstest.MapFile{Data: contents},
		}

		_, _, err := ParseFilesystems([]fs.FS{fs1, fs2}, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "exact same contents")
	})

	t.Run("skips non-sqlcode files", func(t *testing.T) {
		fsys := fstest.MapFS{
			"regular.sql": &fstest.MapFile{
				Data: []byte(`select * from table1`),
			},
			"sqlcode.sql": &fstest.MapFile{
				Data: []byte(`create procedure [code].Test as begin end`),
			},
		}

		filenames, doc, err := ParseFilesystems([]fs.FS{fsys}, nil)
		require.NoError(t, err)
		assert.Len(t, filenames, 1)
		assert.Contains(t, filenames[0], "sqlcode.sql")
		assert.Len(t, doc.Creates, 1)
	})

	t.Run("skips hidden directories", func(t *testing.T) {
		fsys := fstest.MapFS{
			"visible.sql": &fstest.MapFile{
				Data: []byte(`create procedure [code].Visible as begin end`),
			},
			".hidden/test.sql": &fstest.MapFile{
				Data: []byte(`create procedure [code].Hidden as begin end`),
			},
			"dir/.git/test.sql": &fstest.MapFile{
				Data: []byte(`create procedure [code].Git as begin end`),
			},
		}

		filenames, doc, err := ParseFilesystems([]fs.FS{fsys}, nil)
		require.NoError(t, err)
		assert.Len(t, filenames, 1)
		assert.Contains(t, filenames[0], "visible.sql")
		assert.Len(t, doc.Creates, 1)
	})

	t.Run("handles dependencies and topological sort", func(t *testing.T) {
		fsys := fstest.MapFS{
			"proc1.sql": &fstest.MapFile{
				Data: []byte(`create procedure [code].Proc1 as begin exec [code].Proc2 end`),
			},
			"proc2.sql": &fstest.MapFile{
				Data: []byte(`create procedure [code].Proc2 as begin select 1 end`),
			},
		}

		filenames, doc, err := ParseFilesystems([]fs.FS{fsys}, nil)
		require.NoError(t, err)
		assert.Len(t, filenames, 2)
		assert.Len(t, doc.Creates, 2)
		// Proc2 should come before Proc1 due to dependency
		assert.Equal(t, "[Proc2]", doc.Creates[0].QuotedName.Value)
		assert.Equal(t, "[Proc1]", doc.Creates[1].QuotedName.Value)
	})

	t.Run("reports topological sort errors", func(t *testing.T) {
		fsys := fstest.MapFS{
			"circular1.sql": &fstest.MapFile{
				Data: []byte(`create procedure [code].A as begin exec [code].B end`),
			},
			"circular2.sql": &fstest.MapFile{
				Data: []byte(`create procedure [code].B as begin exec [code].A end`),
			},
		}

		_, doc, err := ParseFilesystems([]fs.FS{fsys}, nil)
		require.NoError(t, err)        // filesystem error should be nil
		assert.NotEmpty(t, doc.Errors) // but parsing errors should exist
		assert.Contains(t, doc.Errors[0].Message, "Detected a dependency cycle")
	})

	t.Run("handles multiple filesystems", func(t *testing.T) {
		fs1 := fstest.MapFS{
			"test1.sql": &fstest.MapFile{
				Data: []byte(`create procedure [code].Proc1 as begin end`),
			},
		}
		fs2 := fstest.MapFS{
			"test2.sql": &fstest.MapFile{
				Data: []byte(`create procedure [code].Proc2 as begin end`),
			},
		}

		filenames, doc, err := ParseFilesystems([]fs.FS{fs1, fs2}, nil)
		require.NoError(t, err)
		assert.Len(t, filenames, 2)
		assert.Contains(t, filenames[0], "fs[0]:")
		assert.Contains(t, filenames[1], "fs[1]:")
		assert.Len(t, doc.Creates, 2)
	})

	t.Run("detects sqlcode files by pragma header", func(t *testing.T) {
		fsys := fstest.MapFS{
			"test.sql": &fstest.MapFile{
				Data: []byte(`--sqlcode:include-if foo
create procedure NotInCodeSchema.Test as begin end`),
			},
		}

		filenames, doc, err := ParseFilesystems([]fs.FS{fsys}, []string{"foo"})
		require.NoError(t, err)
		assert.Len(t, filenames, 1)
		// Should still parse even though it will have errors (not in [code] schema)
		assert.NotEmpty(t, doc.Errors)
	})

	t.Run("handles pgsql extension", func(t *testing.T) {
		fsys := fstest.MapFS{
			"test.pgsql": &fstest.MapFile{
				Data: []byte(`
create procedure [code].test()
language plpgsql
as $$
begin
    perform 1;
end;
$$;
`),
			},
		}

		filenames, doc, err := ParseFilesystems([]fs.FS{fsys}, nil)
		require.NoError(t, err)
		assert.Len(t, filenames, 1)
		assert.Len(t, doc.Creates, 1)
		assert.Equal(t, &stdlib.Driver{}, doc.Creates[0].Driver)
	})

	t.Run("empty filesystem returns empty results", func(t *testing.T) {
		fsys := fstest.MapFS{}

		filenames, doc, err := ParseFilesystems([]fs.FS{fsys}, nil)
		require.NoError(t, err)
		assert.Empty(t, filenames)
		assert.Empty(t, doc.Creates)
		assert.Empty(t, doc.Declares)
	})
}

func TestMatchesIncludeTags(t *testing.T) {
	t.Run("empty requirements matches anything", func(t *testing.T) {
		assert.True(t, matchesIncludeTags([]string{}, []string{}))
		assert.True(t, matchesIncludeTags([]string{}, []string{"foo"}))
	})

	t.Run("all requirements must be met", func(t *testing.T) {
		assert.True(t, matchesIncludeTags([]string{"foo", "bar"}, []string{"foo", "bar", "baz"}))
		assert.False(t, matchesIncludeTags([]string{"foo", "bar"}, []string{"foo"}))
		assert.False(t, matchesIncludeTags([]string{"foo", "bar"}, []string{"bar"}))
	})

	t.Run("exact match", func(t *testing.T) {
		assert.True(t, matchesIncludeTags([]string{"foo"}, []string{"foo"}))
		assert.False(t, matchesIncludeTags([]string{"foo"}, []string{"bar"}))
	})
}

func TestIsSqlcodeConstVariable(t *testing.T) {
	testCases := []struct {
		name     string
		varname  string
		expected bool
	}{
		{"@Enum prefix", "@EnumFoo", true},
		{"@ENUM_ prefix", "@ENUM_FOO", true},
		{"@enum_ prefix", "@enum_foo", true},
		{"@Const prefix", "@ConstFoo", true},
		{"@CONST_ prefix", "@CONST_FOO", true},
		{"@const_ prefix", "@const_foo", true},
		{"regular variable", "@MyVariable", false},
		{"@Global prefix", "@GlobalVar", false},
		{"no @ prefix", "EnumFoo", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, IsSqlcodeConstVariable(tc.varname))
		})
	}
}
