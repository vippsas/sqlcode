package sqltest

import (
	"context"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vippsas/sqlcode"
)

func Test_RowsAffected(t *testing.T) {
	fixture := NewFixture()
	defer fixture.Teardown()
	fixture.RunMigrationFile("../migrations/0001.sqlcode.sql")

	ctx := context.Background()

	require.NoError(t, SQL.EnsureUploaded(ctx, fixture.DB))
	patched := SQL.Patch(`[code].Test`)

	res, err := fixture.DB.ExecContext(ctx, patched)
	require.NoError(t, err)
	rowsAffected, err := res.RowsAffected()
	require.NoError(t, err)
	assert.Equal(t, int64(1), rowsAffected)

	schemas := SQL.ListUploaded(ctx, fixture.DB)
	require.Len(t, schemas, 1)
	require.Equal(t, 6, schemas[0].Objects)
	require.Equal(t, "5420c0269aaf", schemas[0].Suffix())
}

func TestEnsureUploadedPinsApplicationLockConnection(t *testing.T) {
	fixture := NewFixture()
	defer fixture.Teardown()
	fixture.RunMigrationFile("../migrations/0001.sqlcode.sql")

	const schemaSuffix = "ensure_uploaded_pinned_connection"
	_, err := fixture.DB.ExecContext(context.Background(), `
create trigger EnsureUploadedUsesLockConnection on database
for create_schema
as
begin
	if applock_mode('public', 'sqlcode.EnsureUploaded/ensure_uploaded_pinned_connection', 'Session') <> 'Shared'
		throw 50000, 'EnsureUploaded did not keep the application lock connection pinned', 1;
end
`)
	require.NoError(t, err)

	sqlFiles := fstest.MapFS{
		"pinned_connection.sql": &fstest.MapFile{Data: []byte(`
create procedure [code].PinnedConnectionTest as
begin
	select 1
end
`)},
	}
	deployable, err := sqlcode.Include(sqlcode.Options{}, sqlFiles)
	require.NoError(t, err)
	deployable = deployable.WithSchemaSuffix(schemaSuffix)

	require.NoError(t, deployable.EnsureUploaded(context.Background(), fixture.DB))
}
