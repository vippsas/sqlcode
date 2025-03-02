package sqlcode

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vippsas/sqlcode/sqlparser"
)

func TestSchemaSuffixFromHash(t *testing.T) {
	t.Run("returns a unique hash", func(t *testing.T) {
		doc := sqlparser.Document{
			Declares: []sqlparser.Declare{},
		}

		value := SchemaSuffixFromHash(doc)
		require.Equal(t, value, SchemaSuffixFromHash(doc))
	})
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
