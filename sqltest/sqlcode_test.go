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

	fixture.RunIfMssql(t, "returns 1 affected row", func(t *testing.T) {
		patched := SQL.CodePatch(fixture.DB, `[code].Test`)
		res, err := fixture.DB.ExecContext(ctx, patched)
		require.NoError(t, err)

		rowsAffected, err := res.RowsAffected()
		require.NoError(t, err)
		assert.Equal(t, int64(1), rowsAffected)
	})

	// TODO: instrument a test table to perform an update operation
	fixture.RunIfPostgres(t, "returns 0 affected rows", func(t *testing.T) {
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

	f.RunIfMssql(t, "uploads schema", func(t *testing.T) {
		f.RunMigrationFile("../migrations/0001.sqlcode.sql")
		require.NoError(t, SQL.EnsureUploaded(ctx, f.DB))
		schemas, err := SQL.ListUploaded(ctx, f.DB)
		require.NoError(t, err)
		require.Len(t, schemas, 1)

	})

	f.RunIfPostgres(t, "uploads schema", func(t *testing.T) {
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

func Test_DropAndUpload(t *testing.T) {
	f := NewFixture()
	defer f.Teardown()
	ctx := context.Background()

	f.RunIfMssql(t, "drop and upload", func(t *testing.T) {
		f.RunMigrationFile("../migrations/0001.sqlcode.sql")
		require.NoError(t, SQL.EnsureUploaded(ctx, f.DB))
		require.NoError(t, SQL.DropAndUpload(ctx, f.DB))
	})

	f.RunIfPostgres(t, "drop and upload", func(t *testing.T) {
		f.RunMigrationFile("../migrations/0001.sqlcode.pgsql")

		_, err := f.DB.Exec(
			fmt.Sprintf(`grant create on database "%s" to "sqlcode-definer-role"`, f.DBName))
		require.NoError(t, err)

		require.NoError(t, SQL.EnsureUploaded(ctx, f.DB))
		require.NoError(t, SQL.DropAndUpload(ctx, f.DB))
	})
}
