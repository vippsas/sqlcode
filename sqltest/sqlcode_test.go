package sqltest

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Patch(t *testing.T) {
	fixture := NewFixture()
	ctx := context.Background()
	defer fixture.Teardown()

	if fixture.IsSqlServer() {
		fixture.RunMigrationFile("../migrations/0001.sqlcode.sql")
	}

	if fixture.IsPostgresql() {
		fixture.RunMigrationFile("../migrations/0001.sqlcode.pgsql")
		_, err := fixture.DB.Exec(
			fmt.Sprintf(`grant create on database "%s" to "sqlcode-definer-role"`, fixture.DBName))
		require.NoError(t, err)
	}

	require.NoError(t, SQL.EnsureUploaded(ctx, fixture.DB))

	fixture.RunIfMssql(t, "mssql", func(t *testing.T) {
		patched := SQL.CodePatch(fixture.DB, `[code].Test`)
		res, err := fixture.DB.ExecContext(ctx, patched)
		require.NoError(t, err)

		rowsAffected, err := res.RowsAffected()
		require.NoError(t, err)
		assert.Equal(t, int64(1), rowsAffected)
	})

	fixture.RunIfPostgres(t, "pgsql", func(t *testing.T) {
		patched := SQL.CodePatch(fixture.DB, `call [code].Test()`)
		res, err := fixture.DB.ExecContext(ctx, patched)
		require.NoError(t, err)

		// postgresql perform does not result with affected rows
		rowsAffected, err := res.RowsAffected()
		require.NoError(t, err)
		assert.Equal(t, int64(0), rowsAffected)
	})

}

func Test_EnsureUploaded(t *testing.T) {
	f := NewFixture()
	defer f.Teardown()
	ctx := context.Background()

	f.RunIfMssql(t, "mssql", func(t *testing.T) {
		f.RunMigrationFile("../migrations/0001.sqlcode.sql")
		require.NoError(t, SQL.EnsureUploaded(ctx, f.DB))
		schemas, err := SQL.ListUploaded(ctx, f.DB)
		require.NoError(t, err)
		require.Len(t, schemas, 1)

	})

	f.RunIfPostgres(t, "pgsql", func(t *testing.T) {
		f.RunMigrationFile("../migrations/0001.sqlcode.pgsql")

		_, err := f.DB.Exec(
			fmt.Sprintf(`grant create on database "%s" to "sqlcode-definer-role"`, f.DBName))
		require.NoError(t, err)

		require.NoError(t, SQL.EnsureUploaded(ctx, f.DB))
		schemas, err := SQL.ListUploaded(ctx, f.DB)
		require.NoError(t, err)
		require.Len(t, schemas, 1)
	})
}
