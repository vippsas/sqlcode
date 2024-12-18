package sqltest

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
}