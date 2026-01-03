package sqldocument

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createNames(lst []Create) (result []PosString) {
	for _, c := range lst {
		result = append(result, c.QuotedName)
	}
	return
}

func shuffleCreate(input []Create) []Create {
	dest := make([]Create, len(input))
	perm := rand.Perm(len(input))
	for i, v := range perm {
		dest[v] = input[i]
	}
	return dest
}

func TestTopologicalSort_HappyDay(t *testing.T) {
	input := []Create{
		{
			QuotedName: PosString{Value: "c"},
		},
		{
			QuotedName: PosString{Value: "b"},
			DependsOn:  []PosString{{Value: "c"}},
		},
		{
			QuotedName: PosString{Value: "a"},
			DependsOn:  []PosString{{Value: "b"}, {Value: "c"}},
		},
		{
			QuotedName: PosString{Value: "d"},
			DependsOn:  []PosString{{Value: "a"}},
		},
	}
	for i := 0; i != 5; i++ {
		output, _, err := TopologicalSort(shuffleCreate(input))
		require.NoError(t, err)
		assert.Equal(t, createNames(input), createNames(output))
	}
}

func TestTopologicalSort_Cycle(t *testing.T) {
	input := []Create{
		{
			QuotedName: PosString{Value: "a"},
			DependsOn:  []PosString{{Pos{Line: 1}, "b"}},
		},
		{
			QuotedName: PosString{Value: "b"},
			DependsOn:  []PosString{{Pos{Line: 1}, "c"}},
		},
		{
			QuotedName: PosString{Value: "c"},
			DependsOn:  []PosString{{Pos{Line: 1}, "a"}},
		},
	}
	_, errpos, err := TopologicalSort(input)
	require.Equal(t, CycleError, err)
	require.Equal(t, 1, errpos.Line) // note that 1 is used for the allowed spots above
}

func TestTopologicalSort_NotFound(t *testing.T) {
	input := []Create{
		{
			QuotedName: PosString{Pos{Line: 1}, "a"},
			DependsOn:  []PosString{{Value: "b"}},
		},
		{
			QuotedName: PosString{Value: "b"},
			DependsOn:  []PosString{{Pos{Line: 2}, "c"}},
		},
	}
	_, errpos, err := TopologicalSort(input)
	require.Equal(t, 2, errpos.Line)
	require.Equal(t, "Name not found: c", err.Error())
}
