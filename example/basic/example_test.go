//go:build examples
// +build examples

package example

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vippsas/sqlcode/sqltest"
)

func TestPreprocess(t *testing.T) {
	patched := SQL.Patch(`select [code].AddTwoNumbers(2, 3)`)
	assert.Equal(t,
		// e.g.. select [code@775c0f272ae4].AddTwoNumbers(2, 3)
		fmt.Sprintf(`select [code@%s].AddTwoNumbers(2, 3)`, SQL.SchemaSuffix),
		patched)
}

var sqlPatchedBeforeUpload = SQL.Patch(`select [code].AddTwoNumbers(2, 3)`)

func TestCallSqlCode(t *testing.T) {
	fixture := sqltest.NewFixture()
	defer fixture.Teardown()
	fixture.RunMigrationFile("../migrations/0001.sqlcode.sql")

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	require.NoError(t, SQL.EnsureUploaded(ctx, fixture.DB))

	sqlPatchedAfterUpload := SQL.Patch(`select [code].AddTwoNumbers(2, 3)`)

	var x int
	require.NoError(t, fixture.DB.QueryRowContext(ctx, sqlPatchedBeforeUpload).Scan(&x))
	assert.Equal(t, 5, x)
	require.NoError(t, fixture.DB.QueryRowContext(ctx, sqlPatchedAfterUpload).Scan(&x))
	assert.Equal(t, 5, x)
}
