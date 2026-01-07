package sqlparser

import (
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
		assert.Len(t, doc.Creates(), 2)
		assert.Len(t, doc.Declares(), 1)
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
		assert.Len(t, doc.Creates(), 1)
		assert.Equal(t, "[Included]", doc.Creates()[0].QuotedName.Value)
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
		assert.Len(t, doc.Creates(), 1)
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
		assert.Len(t, doc.Creates(), 1)
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
		assert.Len(t, doc.Creates(), 2)
		// Proc2 should come before Proc1 due to dependency
		assert.Equal(t, "[Proc2]", doc.Creates()[0].QuotedName.Value)
		assert.Equal(t, "[Proc1]", doc.Creates()[1].QuotedName.Value)
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
		require.NoError(t, err)          // filesystem error should be nil
		assert.NotEmpty(t, doc.Errors()) // but parsing errors should exist
		assert.Contains(t, doc.Errors()[0].Message, "Detected a dependency cycle")
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
		assert.Len(t, doc.Creates(), 2)
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
		assert.NotEmpty(t, doc.Errors())
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
		assert.Len(t, doc.Creates(), 1)
		assert.Equal(t, &stdlib.Driver{}, doc.Creates()[0].Driver)
	})

	t.Run("empty filesystem returns empty results", func(t *testing.T) {
		fsys := fstest.MapFS{}

		filenames, doc, err := ParseFilesystems([]fs.FS{fsys}, nil)
		require.NoError(t, err)
		assert.Empty(t, filenames)
		assert.Empty(t, doc.Creates())
		assert.Empty(t, doc.Declares())
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
