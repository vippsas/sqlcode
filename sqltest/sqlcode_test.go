package sqltest

import (
	"context"
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

	require.NoError(t, SQL.EnsureUploaded(ctx, fixture.DB))

	fixture.RunIfMssql(t, "returns 1 affected row", func(t *testing.T) {
		patched := SQL.CodePatch(fixture.DB, `[code].Test`)
		res, err := fixture.DB.ExecContext(ctx, patched)
		require.NoError(t, err)

		rowsAffected, err := res.RowsAffected()
		require.NoError(t, err)
		assert.Equal(t, int64(1), rowsAffected)
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
}
