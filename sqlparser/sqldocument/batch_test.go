package sqldocument

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBatch(t *testing.T) {
	b := NewBatch()

	assert.NotNil(t, b)
	assert.NotNil(t, b.TokenCalls)
	assert.NotNil(t, b.Nodes)
	assert.NotNil(t, b.DocString)
	assert.NotNil(t, b.Errors)
	assert.Empty(t, b.TokenCalls)
	assert.Empty(t, b.Nodes)
	assert.Empty(t, b.DocString)
	assert.Empty(t, b.Errors)
}

func TestBatch_Create(t *testing.T) {
	b := NewBatch()
	scanner := &MockScanner{
		tokens: []MockToken{
			{Type: ReservedWordToken, Text: "SELECT", StartPos: Pos{Line: 1, Col: 1}},
		},
	}

	b.Create(scanner)

	require.Len(t, b.Nodes, 1)
	assert.Equal(t, "SELECT", b.Nodes[0].RawValue)
	assert.Equal(t, Pos{Line: 1, Col: 1}, b.Nodes[0].Start)
}

func TestBatch_HasErrors(t *testing.T) {
	t.Run("no errors", func(t *testing.T) {
		b := NewBatch()
		assert.False(t, b.HasErrors())
	})

	t.Run("with errors", func(t *testing.T) {
		b := NewBatch()
		b.Errors = append(b.Errors, Error{Message: "test error"})
		assert.True(t, b.HasErrors())
	})
}

func TestBatch_Parse_EOF(t *testing.T) {
	b := NewBatch()
	scanner := &MockScanner{
		tokens: []MockToken{
			{Type: EOFToken, Text: ""},
		},
	}

	result := b.Parse(scanner)

	assert.False(t, result)
}

func TestBatch_Parse_Whitespace(t *testing.T) {
	b := NewBatch()
	scanner := &MockScanner{
		tokens: []MockToken{
			{Type: WhitespaceToken, Text: "   "},
			{Type: EOFToken, Text: ""},
		},
	}

	result := b.Parse(scanner)

	assert.False(t, result)
	require.Len(t, b.Nodes, 1)
	assert.Equal(t, "   ", b.Nodes[0].RawValue)
	// DocString should be nil after non-newline whitespace
	assert.Nil(t, b.DocString)
}

func TestBatch_Parse_WhitespaceNewline(t *testing.T) {
	b := NewBatch()
	b.DocString = []PosString{{Value: "-- comment"}}
	scanner := &MockScanner{
		tokens: []MockToken{
			{Type: WhitespaceToken, Text: "\n"},
			{Type: EOFToken, Text: ""},
		},
	}

	result := b.Parse(scanner)

	assert.False(t, result)
	// DocString should be preserved after single newline
	require.Len(t, b.DocString, 1)
}

func TestBatch_Parse_MultilineComment(t *testing.T) {
	b := NewBatch()
	scanner := &MockScanner{
		tokens: []MockToken{
			{Type: MultilineCommentToken, Text: "/* comment */"},
			{Type: EOFToken, Text: ""},
		},
	}

	result := b.Parse(scanner)

	assert.False(t, result)
	require.Len(t, b.Nodes, 1)
	assert.Equal(t, "/* comment */", b.Nodes[0].RawValue)
}

func TestBatch_Parse_SinglelineComment(t *testing.T) {
	b := NewBatch()
	scanner := &MockScanner{
		tokens: []MockToken{
			{Type: SinglelineCommentToken, Text: "-- comment 1", StartPos: Pos{Line: 1, Col: 1}},
			{Type: SinglelineCommentToken, Text: "-- comment 2", StartPos: Pos{Line: 2, Col: 1}},
			{Type: EOFToken, Text: ""},
		},
	}

	result := b.Parse(scanner)

	assert.False(t, result)
	require.Len(t, b.Nodes, 2)
	require.Len(t, b.DocString, 2)
	assert.Equal(t, "-- comment 1", b.DocString[0].Value)
	assert.Equal(t, "-- comment 2", b.DocString[1].Value)
}

func TestBatch_Parse_DocStringReset(t *testing.T) {
	t.Run("reset on multiline whitespace", func(t *testing.T) {
		b := NewBatch()
		scanner := &MockScanner{
			tokens: []MockToken{
				{Type: SinglelineCommentToken, Text: "-- comment"},
				{Type: WhitespaceToken, Text: "   "}, // Not just newline
				{Type: EOFToken, Text: ""},
			},
		}

		b.Parse(scanner)

		// DocString should be nil after non-newline whitespace
		assert.Nil(t, b.DocString)
	})

	t.Run("reset on unexpected token", func(t *testing.T) {
		b := NewBatch()
		b.TokenHandlers = map[string]func(Scanner, *Batch) int{}
		scanner := &MockScanner{
			tokens: []MockToken{
				{Type: SinglelineCommentToken, Text: "-- comment"},
				{Type: OtherToken, Text: "@"}, // Unexpected
				{Type: EOFToken, Text: ""},
			},
		}

		b.Parse(scanner)

		assert.Nil(t, b.DocString)
	})
}

func TestBatch_Parse_ReservedWord_WithHandler(t *testing.T) {
	t.Run("handler returns 0 (continue)", func(t *testing.T) {
		handlerCalled := false
		b := NewBatch()
		b.TokenHandlers = map[string]func(Scanner, *Batch) int{
			"select": func(s Scanner, batch *Batch) int {
				handlerCalled = true
				s.NextToken() // Advance to EOF
				return 0
			},
		}
		scanner := &MockScanner{
			tokens: []MockToken{
				{Type: ReservedWordToken, Text: "SELECT", Reserved: "select"},
				{Type: EOFToken, Text: ""},
			},
		}

		result := b.Parse(scanner)

		assert.True(t, handlerCalled)
		assert.False(t, result)
		assert.Equal(t, 1, b.TokenCalls["select"])
	})

	t.Run("handler returns 1 (new batch)", func(t *testing.T) {
		b := NewBatch()
		b.TokenHandlers = map[string]func(Scanner, *Batch) int{
			"declare": func(s Scanner, batch *Batch) int {
				return 1
			},
		}
		scanner := &MockScanner{
			tokens: []MockToken{
				{Type: ReservedWordToken, Text: "DECLARE", Reserved: "declare"},
				{Type: EOFToken, Text: ""},
			},
		}

		result := b.Parse(scanner)

		assert.True(t, result)
		assert.Equal(t, 1, b.TokenCalls["declare"])
	})

	t.Run("handler returns -1 (stop)", func(t *testing.T) {
		b := NewBatch()
		b.TokenHandlers = map[string]func(Scanner, *Batch) int{
			"create": func(s Scanner, batch *Batch) int {
				return -1
			},
		}
		scanner := &MockScanner{
			tokens: []MockToken{
				{Type: ReservedWordToken, Text: "CREATE", Reserved: "create"},
				{Type: EOFToken, Text: ""},
			},
		}

		result := b.Parse(scanner)

		assert.False(t, result)
		assert.Equal(t, 1, b.TokenCalls["create"])
	})
}

func TestBatch_Parse_ReservedWord_NoHandler(t *testing.T) {
	b := NewBatch()
	b.TokenHandlers = map[string]func(Scanner, *Batch) int{}
	scanner := &MockScanner{
		tokens: []MockToken{
			{Type: ReservedWordToken, Text: "SELECT", Reserved: "select"},
			{Type: EOFToken, Text: ""},
		},
	}

	result := b.Parse(scanner)

	assert.False(t, result)
	// Should not track calls for unhandled reserved words
	assert.Equal(t, 0, b.TokenCalls["select"])
}

func TestBatch_Parse_BatchSeparator(t *testing.T) {
	handlerCalled := false
	b := NewBatch()
	b.BatchSeparatorHandler = func(s Scanner, batch *Batch) {
		handlerCalled = true
	}
	scanner := &MockScanner{
		tokens: []MockToken{
			{Type: BatchSeparatorToken, Text: "GO"},
			{Type: EOFToken, Text: ""},
		},
	}

	result := b.Parse(scanner)

	assert.True(t, handlerCalled)
	assert.True(t, result)
}

func TestBatch_Parse_UnexpectedToken(t *testing.T) {
	b := NewBatch()
	b.TokenHandlers = map[string]func(Scanner, *Batch) int{}
	scanner := &MockScanner{
		tokens: []MockToken{
			{Type: OtherToken, Text: "@unexpected", StartPos: Pos{Line: 1, Col: 1}},
			{Type: EOFToken, Text: ""},
		},
	}

	result := b.Parse(scanner)

	assert.False(t, result)
	require.Len(t, b.Errors, 1)
	assert.Contains(t, b.Errors[0].Message, "Unexpected token: @unexpected")
}

func TestBatch_Parse_MultipleTokenCalls(t *testing.T) {
	b := NewBatch()
	callCount := 0
	b.TokenHandlers = map[string]func(Scanner, *Batch) int{
		"create": func(s Scanner, batch *Batch) int {
			callCount++
			s.NextToken() // Advance past CREATE
			return 0      // Continue parsing
		},
	}
	scanner := &MockScanner{
		tokens: []MockToken{
			{Type: ReservedWordToken, Text: "CREATE", Reserved: "create"},
			{Type: ReservedWordToken, Text: "CREATE", Reserved: "create"},
			{Type: ReservedWordToken, Text: "CREATE", Reserved: "create"},
			{Type: EOFToken, Text: ""},
		},
	}

	result := b.Parse(scanner)

	assert.False(t, result)
	assert.Equal(t, 3, callCount)
	assert.Equal(t, 3, b.TokenCalls["create"])
}

func TestBatch_Parse_PreservesNodes(t *testing.T) {
	b := NewBatch()
	b.TokenHandlers = map[string]func(Scanner, *Batch) int{
		"create": func(s Scanner, batch *Batch) int {
			// Handler can access accumulated nodes
			assert.Len(t, batch.Nodes, 2) // Comment + whitespace
			return -1
		},
	}

	scanner := &MockScanner{
		tokens: []MockToken{
			{Type: SinglelineCommentToken, Text: "-- comment"},
			{Type: WhitespaceToken, Text: "\n"},
			{Type: ReservedWordToken, Text: "CREATE", Reserved: "create"},
			{Type: EOFToken, Text: ""},
		},
	}

	b.Parse(scanner)

	// Nodes should be preserved
	require.Len(t, b.Nodes, 2)
}

func TestBatch_Parse_DocStringBuildup(t *testing.T) {
	b := NewBatch()
	b.TokenHandlers = map[string]func(Scanner, *Batch) int{
		"create": func(s Scanner, batch *Batch) int {
			// At this point, we should have two docstring entries
			require.Len(t, batch.DocString, 2)
			assert.Equal(t, "-- First comment", batch.DocString[0].Value)
			assert.Equal(t, "-- Second comment", batch.DocString[1].Value)
			return -1
		},
	}

	scanner := &MockScanner{
		tokens: []MockToken{
			{Type: SinglelineCommentToken, Text: "-- First comment", StartPos: Pos{Line: 1, Col: 1}},
			{Type: WhitespaceToken, Text: "\n"},
			{Type: SinglelineCommentToken, Text: "-- Second comment", StartPos: Pos{Line: 2, Col: 1}},
			{Type: WhitespaceToken, Text: "\n"},
			{Type: ReservedWordToken, Text: "CREATE", Reserved: "create"},
			{Type: EOFToken, Text: ""},
		},
	}

	b.Parse(scanner)
}

func TestBatch_Parse_ErrorAccumulation(t *testing.T) {
	b := NewBatch()
	b.TokenHandlers = map[string]func(Scanner, *Batch) int{}

	scanner := &MockScanner{
		tokens: []MockToken{
			{Type: OtherToken, Text: "@error1", StartPos: Pos{Line: 1, Col: 1}},
			{Type: OtherToken, Text: "@error2", StartPos: Pos{Line: 2, Col: 1}},
			{Type: OtherToken, Text: "@error3", StartPos: Pos{Line: 3, Col: 1}},
			{Type: EOFToken, Text: ""},
		},
	}

	b.Parse(scanner)

	require.Len(t, b.Errors, 3)
	assert.Contains(t, b.Errors[0].Message, "@error1")
	assert.Contains(t, b.Errors[1].Message, "@error2")
	assert.Contains(t, b.Errors[2].Message, "@error3")
}
