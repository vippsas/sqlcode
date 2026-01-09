package sqlcode

import (
	"strings"
	"testing"

	mssql "github.com/microsoft/go-mssqldb"
	"github.com/simukka/sqlcode/sqlparser"
	mssql17 "github.com/simukka/sqlcode/sqlparser/mssql"
	"github.com/simukka/sqlcode/sqlparser/sqldocument"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ParseString(t *testing.T, file, input string) *mssql17.TSqlDocument {
	d := &mssql17.TSqlDocument{}
	assert.NoError(t, d.Parse([]byte(input), sqldocument.FileRef(file)))
	return d
}

func TestLineNumberInInput(t *testing.T) {

	// Scenario:
	// line 5 in input was transformed to 3 lines in output
	// line 7 in input was transformed to 2 lines in output
	// line 8 in input was transformed to 2 lines in output
	p := Batch{
		lineNumberCorrections: []lineNumberCorrection{
			{5, 2},
			{7, 1},
			{8, 1},
		},
	}
	expectedInputLineNumbers := []int{
		1,
		2,
		3,
		4,

		5,
		5,
		5,

		6,

		7,
		7,

		8,
		8,

		9,
		10,
		11,
		12,
	}

	var inputlines [17]int
	for lineno := 1; lineno <= 16; lineno++ {
		inputlines[lineno] = p.RelativeLineNumberInInput(lineno)
		//fmt.Println(p.RelativeLineNumberInInput(lineno), lineno)
	}
	assert.Equal(t, expectedInputLineNumbers, inputlines[1:])
}

func TestSchemaSuffixFromHash(t *testing.T) {
	t.Run("returns a unique hash", func(t *testing.T) {
		doc := sqlparser.NewDocumentFromExtension(".sql")
		value := SchemaSuffixFromHash(doc)
		require.Equal(t, value, SchemaSuffixFromHash(doc))
	})

	t.Run("returns consistent hash", func(t *testing.T) {
		doc := ParseString(t, "test.sql", `
declare @EnumFoo int = 1;
go
create procedure [code].Test as begin end
`)

		suffix1 := SchemaSuffixFromHash(doc)
		suffix2 := SchemaSuffixFromHash(doc)

		assert.Equal(t, suffix1, suffix2)
		assert.Len(t, suffix1, 12) // 6 bytes = 12 hex chars
	})

	t.Run("different content yields different hash", func(t *testing.T) {
		doc1 := ParseString(t, "test.sql", `
declare @EnumFoo int = 1;
go
create procedure [code].Test1 as begin end
`)
		doc2 := ParseString(t, "test.sql", `
declare @EnumFoo int = 2;
go
create procedure [code].Test2 as begin end
`)

		suffix1 := SchemaSuffixFromHash(doc1)
		suffix2 := SchemaSuffixFromHash(doc2)

		assert.NotEqual(t, suffix1, suffix2)
	})
}

func TestSchemaName(t *testing.T) {
	assert.Equal(t, "code@abc123", SchemaName("abc123"))
	assert.Equal(t, "code@", SchemaName(""))
}

func TestBatchLineNumberInInput(t *testing.T) {
	t.Run("no corrections", func(t *testing.T) {
		b := Batch{
			StartPos: sqldocument.Pos{Line: 10, Col: 1},
			Lines:    "line1\nline2\nline3",
		}

		assert.Equal(t, 10, b.LineNumberInInput(1))
		assert.Equal(t, 11, b.LineNumberInInput(2))
		assert.Equal(t, 12, b.LineNumberInInput(3))
	})

	t.Run("with corrections", func(t *testing.T) {
		b := Batch{
			StartPos: sqldocument.Pos{Line: 10, Col: 1},
			Lines:    "line1\nline2\nextra1\nextra2\nline3",
			lineNumberCorrections: []lineNumberCorrection{
				{inputLineNumber: 2, extraLinesInOutput: 2}, // line 2 became 3 lines
			},
		}

		assert.Equal(t, 10, b.LineNumberInInput(1)) // line 1 -> input line 10
		assert.Equal(t, 11, b.LineNumberInInput(2)) // line 2 -> input line 11
		assert.Equal(t, 11, b.LineNumberInInput(3)) // extra line -> still input line 11
		assert.Equal(t, 11, b.LineNumberInInput(4)) // extra line -> still input line 11
		assert.Equal(t, 12, b.LineNumberInInput(5)) // line 3 -> input line 12
	})
}

func TestBatchRelativeLineNumberInInput(t *testing.T) {
	t.Run("simple case with no corrections", func(t *testing.T) {
		b := Batch{
			lineNumberCorrections: []lineNumberCorrection{},
		}

		assert.Equal(t, 1, b.RelativeLineNumberInInput(1))
		assert.Equal(t, 5, b.RelativeLineNumberInInput(5))
	})

	t.Run("single correction", func(t *testing.T) {
		b := Batch{
			lineNumberCorrections: []lineNumberCorrection{
				{inputLineNumber: 3, extraLinesInOutput: 2},
			},
		}

		assert.Equal(t, 1, b.RelativeLineNumberInInput(1))
		assert.Equal(t, 2, b.RelativeLineNumberInInput(2))
		assert.Equal(t, 3, b.RelativeLineNumberInInput(3))
		assert.Equal(t, 3, b.RelativeLineNumberInInput(4)) // extra line
		assert.Equal(t, 3, b.RelativeLineNumberInInput(5)) // extra line
		assert.Equal(t, 4, b.RelativeLineNumberInInput(6))
	})

	t.Run("multiple corrections", func(t *testing.T) {
		b := Batch{
			lineNumberCorrections: []lineNumberCorrection{
				{inputLineNumber: 2, extraLinesInOutput: 1},
				{inputLineNumber: 5, extraLinesInOutput: 3},
			},
		}

		assert.Equal(t, 1, b.RelativeLineNumberInInput(1))
		assert.Equal(t, 2, b.RelativeLineNumberInInput(2))
		assert.Equal(t, 2, b.RelativeLineNumberInInput(3)) // extra from line 2
		assert.Equal(t, 3, b.RelativeLineNumberInInput(4))
		assert.Equal(t, 4, b.RelativeLineNumberInInput(5))
		assert.Equal(t, 5, b.RelativeLineNumberInInput(6))
		assert.Equal(t, 5, b.RelativeLineNumberInInput(7)) // extra from line 5
		assert.Equal(t, 5, b.RelativeLineNumberInInput(8)) // extra from line 5
		assert.Equal(t, 5, b.RelativeLineNumberInInput(9)) // extra from line 5
		assert.Equal(t, 6, b.RelativeLineNumberInInput(10))
	})
}

func TestPreprocess(t *testing.T) {
	t.Run("basic procedure with schema replacement", func(t *testing.T) {
		doc := ParseString(t, "test.sql", `
create procedure [code].Test as
begin
    select 1
end
`)

		result, err := Preprocess(doc, "abc123", &mssql.Driver{})
		require.NoError(t, err)
		require.Len(t, result.Batches, 1)

		assert.Contains(t, result.Batches[0].Lines, "[code@abc123].")
		assert.NotContains(t, result.Batches[0].Lines, "[code].")
	})

	t.Run("replaces enum constants", func(t *testing.T) {
		doc := ParseString(t, "test.sql", `
declare @EnumStatus int = 42;
go
create procedure [code].Test as
begin
    select @EnumStatus
end
`)
		result, err := Preprocess(doc, "abc123", &mssql.Driver{})
		require.NoError(t, err)
		require.Len(t, result.Batches, 1)

		batch := result.Batches[0].Lines
		assert.Contains(t, batch, "42/*=@EnumStatus*/")
		assert.NotContains(t, batch, "@EnumStatus\n") // shouldn't have bare reference
	})

	t.Run("handles multiline string constants", func(t *testing.T) {
		doc := ParseString(t, "test.sql", `
declare @EnumMulti nvarchar(max) = N'line1
line2
line3';
go
create procedure [code].Test as
begin
    select @EnumMulti
end
`)
		result, err := Preprocess(doc, "abc123", &mssql.Driver{})
		require.NoError(t, err)
		require.Len(t, result.Batches, 1)

		batch := result.Batches[0]
		assert.Contains(t, batch.Lines, "N'line1\nline2\nline3'/*=@EnumMulti*/")
		// Should have line number corrections for the 2 extra lines
		assert.Len(t, batch.lineNumberCorrections, 1)
		assert.Equal(t, 2, batch.lineNumberCorrections[0].extraLinesInOutput)
	})

	t.Run("error on undeclared constant", func(t *testing.T) {
		doc := ParseString(t, "test.sql", `
create procedure [code].Test as
begin
    select @EnumUndeclared
end
`)
		_, err := Preprocess(doc, "abc123", &mssql.Driver{})
		require.Error(t, err)

		var preprocErr PreprocessorError
		require.ErrorAs(t, err, &preprocErr)
		assert.Contains(t, preprocErr.Message, "@EnumUndeclared")
		assert.Contains(t, preprocErr.Message, "not declared")
	})

	t.Run("error on schema suffix with bracket", func(t *testing.T) {
		doc := ParseString(t, "test.sql", `
create procedure [code].Test as begin end
`)
		_, err := Preprocess(doc, "abc]123", &mssql.Driver{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "schemasuffix cannot contain")
	})

	t.Run("handles multiple creates", func(t *testing.T) {
		doc := ParseString(t, "test.sql", `
create procedure [code].Proc1 as begin select 1 end
go
create procedure [code].Proc2 as begin select 2 end
`)
		result, err := Preprocess(doc, "abc123", &mssql.Driver{})
		require.NoError(t, err)
		assert.Len(t, result.Batches, 2)

		assert.Contains(t, result.Batches[0].Lines, "Proc1")
		assert.Contains(t, result.Batches[1].Lines, "Proc2")
	})

	t.Run("handles multiple constants in same procedure", func(t *testing.T) {
		doc := ParseString(t, "test.sql", `
declare @EnumA int = 1, @EnumB int = 2;
go
create procedure [code].Test as
begin
    select @EnumA, @EnumB
end
`)
		result, err := Preprocess(doc, "abc123", &mssql.Driver{})
		require.NoError(t, err)
		require.Len(t, result.Batches, 1)

		batch := result.Batches[0].Lines
		assert.Contains(t, batch, "1/*=@EnumA*/")
		assert.Contains(t, batch, "2/*=@EnumB*/")
	})

	t.Run("preserves comments and formatting", func(t *testing.T) {
		doc := ParseString(t, "test.sql", `
-- This is a test procedure
create procedure [code].Test as
begin
    /* multi
       line
       comment */
    select 1
end
`)
		result, err := Preprocess(doc, "abc123", &mssql.Driver{})
		require.NoError(t, err)
		require.Len(t, result.Batches, 1)

		batch := result.Batches[0].Lines
		assert.Contains(t, batch, "-- This is a test procedure")
		assert.Contains(t, batch, "/* multi")
	})

	t.Run("handles const and global prefixes", func(t *testing.T) {
		doc := ParseString(t, "test.sql", `
declare @ConstValue int = 100; 
declare @GlobalSetting nvarchar(50) = N'test';
go
create procedure [code].Test as
begin
    select @ConstValue, @GlobalSetting
end
`)
		result, err := Preprocess(doc, "abc123", &mssql.Driver{})
		require.NoError(t, err)
		require.Len(t, result.Batches, 1)

		batch := result.Batches[0].Lines
		assert.Contains(t, batch, "100/*=@ConstValue*/")
		assert.NotContains(t, batch, "N'test'/*=@GlobalSetting*/")
	})
}

func TestPreprocessString(t *testing.T) {
	t.Run("replaces code schema", func(t *testing.T) {
		result := preprocessString("abc123", "select * from [code].Table")
		assert.Equal(t, "select * from [code@abc123].Table", result)
	})

	t.Run("case insensitive replacement", func(t *testing.T) {
		result := preprocessString("abc123", "select * from [CODE].Table and [Code].Other")
		assert.Contains(t, result, "[code@abc123].Table")
		assert.Contains(t, result, "[code@abc123].Other")
	})

	t.Run("multiple occurrences", func(t *testing.T) {
		sql := `
            select * from [code].A
            join [code].B on A.id = B.id
            where exists (select 1 from [code].C)
        `
		result := preprocessString("abc123", sql)
		assert.Equal(t, 3, strings.Count(result, "[code@abc123]"))
		assert.NotContains(t, result, "[code].")
	})

	t.Run("no replacement needed", func(t *testing.T) {
		sql := "select * from dbo.Table"
		result := preprocessString("abc123", sql)
		assert.Equal(t, sql, result)
	})
}

func TestPreprocessorError(t *testing.T) {
	t.Run("formats error message", func(t *testing.T) {
		err := PreprocessorError{
			Pos:     sqldocument.Pos{File: "test.sql", Line: 10, Col: 5},
			Message: "something went wrong",
		}

		assert.Equal(t, "test.sql:10:5: something went wrong", err.Error())
	})
}
