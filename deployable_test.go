package sqlcode

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"testing/fstest"
)

func TestDeployable(t *testing.T) {
	fs := make(fstest.MapFS)
	fs["test.sql"] = &fstest.MapFile{
		Data: []byte(`--sqlcode:
declare @EnumInt int = 1, @EnumString varchar(max) = 'hello';
`),
	}

	d, err := Include(Options{}, fs)
	require.NoError(t, err)
	n, err := d.IntConst("@EnumInt")
	require.NoError(t, err)
	assert.Equal(t, 1, n)

}
