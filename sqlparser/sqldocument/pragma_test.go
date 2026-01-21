package sqldocument

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPragma_PragmaIncludeIf(t *testing.T) {
	t.Run("empty pragmas", func(t *testing.T) {
		p := Pragma{}
		assert.Empty(t, p.PragmaIncludeIf())
	})

	t.Run("with pragmas", func(t *testing.T) {
		p := Pragma{pragmas: []string{"feature1", "feature2"}}
		assert.Equal(t, []string{"feature1", "feature2"}, p.PragmaIncludeIf())
	})
}

func TestPragma_parseSinglePragma(t *testing.T) {
	t.Run("valid include-if with single feature", func(t *testing.T) {
		scanner := &MockScanner{
			tokens: []MockToken{
				{Type: PragmaToken, Text: "--sqlcode:include-if feature1"},
			},
		}
		p := &Pragma{}

		err := p.parseSinglePragma(scanner)

		require.NoError(t, err)
		assert.Equal(t, []string{"feature1"}, p.pragmas)
	})

	t.Run("valid include-if with multiple features", func(t *testing.T) {
		scanner := &MockScanner{
			tokens: []MockToken{
				{Type: PragmaToken, Text: "--sqlcode:include-if feature1,feature2,feature3"},
			},
		}
		p := &Pragma{}

		err := p.parseSinglePragma(scanner)

		require.NoError(t, err)
		assert.Equal(t, []string{"feature1", "feature2", "feature3"}, p.pragmas)
	})

	t.Run("empty pragma content", func(t *testing.T) {
		scanner := &MockScanner{
			tokens: []MockToken{
				{Type: PragmaToken, Text: "--sqlcode:"},
			},
		}
		p := &Pragma{}

		err := p.parseSinglePragma(scanner)

		require.NoError(t, err)
		assert.Empty(t, p.pragmas)
	})

	t.Run("pragma with whitespace", func(t *testing.T) {
		scanner := &MockScanner{
			tokens: []MockToken{
				{Type: PragmaToken, Text: "--sqlcode: include-if feature1"},
			},
		}
		p := &Pragma{}

		err := p.parseSinglePragma(scanner)

		require.NoError(t, err)
		assert.Equal(t, []string{"feature1"}, p.pragmas)
	})

	t.Run("invalid pragma - unknown directive", func(t *testing.T) {
		scanner := &MockScanner{
			tokens: []MockToken{
				{Type: PragmaToken, Text: "--sqlcode:unknown-directive value"},
			},
		}
		p := &Pragma{}

		err := p.parseSinglePragma(scanner)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "Illegal pragma")
	})

	t.Run("invalid pragma - missing value", func(t *testing.T) {
		scanner := &MockScanner{
			tokens: []MockToken{
				{Type: PragmaToken, Text: "--sqlcode:include-if"},
			},
		}
		p := &Pragma{}

		err := p.parseSinglePragma(scanner)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "Illegal pragma")
	})

	t.Run("invalid pragma - too many parts", func(t *testing.T) {
		scanner := &MockScanner{
			tokens: []MockToken{
				{Type: PragmaToken, Text: "--sqlcode:include-if feature1 extra"},
			},
		}
		p := &Pragma{}

		err := p.parseSinglePragma(scanner)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "Illegal pragma")
	})

	t.Run("accumulates pragmas", func(t *testing.T) {
		p := &Pragma{pragmas: []string{"existing"}}
		scanner := &MockScanner{
			tokens: []MockToken{
				{Type: PragmaToken, Text: "--sqlcode:include-if new1,new2"},
			},
		}

		err := p.parseSinglePragma(scanner)

		require.NoError(t, err)
		assert.Equal(t, []string{"existing", "new1", "new2"}, p.pragmas)
	})
}

func TestPragma_ParsePragmas(t *testing.T) {
	t.Run("no pragmas", func(t *testing.T) {
		scanner := &MockScanner{
			tokens: []MockToken{
				{Type: ReservedWordToken, Text: "CREATE"},
			},
		}
		p := &Pragma{}

		err := p.ParsePragmas(scanner)

		require.NoError(t, err)
		assert.Empty(t, p.pragmas)
	})

	t.Run("single pragma", func(t *testing.T) {
		scanner := &MockScanner{
			tokens: []MockToken{
				{Type: PragmaToken, Text: "--sqlcode:include-if feature1"},
				{Type: WhitespaceToken, Text: "\n"},
				{Type: ReservedWordToken, Text: "CREATE"},
			},
		}
		p := &Pragma{}

		err := p.ParsePragmas(scanner)

		require.NoError(t, err)
		assert.Equal(t, []string{"feature1"}, p.pragmas)
	})

	t.Run("multiple pragmas", func(t *testing.T) {
		scanner := &MockScanner{
			tokens: []MockToken{
				{Type: PragmaToken, Text: "--sqlcode:include-if feature1"},
				{Type: WhitespaceToken, Text: "\n"},
				{Type: PragmaToken, Text: "--sqlcode:include-if feature2"},
				{Type: WhitespaceToken, Text: "\n"},
				{Type: PragmaToken, Text: "--sqlcode:include-if feature3"},
				{Type: WhitespaceToken, Text: "\n"},
				{Type: ReservedWordToken, Text: "CREATE"},
			},
		}
		p := &Pragma{}

		err := p.ParsePragmas(scanner)

		require.NoError(t, err)
		assert.Equal(t, []string{"feature1", "feature2", "feature3"}, p.pragmas)
	})

	t.Run("pragma with comma-separated features", func(t *testing.T) {
		scanner := &MockScanner{
			tokens: []MockToken{
				{Type: PragmaToken, Text: "--sqlcode:include-if one,two"},
				{Type: WhitespaceToken, Text: "\n"},
				{Type: PragmaToken, Text: "--sqlcode:include-if three"},
				{Type: WhitespaceToken, Text: "\n"},
				{Type: ReservedWordToken, Text: "CREATE"},
			},
		}
		p := &Pragma{}

		err := p.ParsePragmas(scanner)

		require.NoError(t, err)
		assert.Equal(t, []string{"one", "two", "three"}, p.pragmas)
	})

	t.Run("stops on invalid pragma", func(t *testing.T) {
		scanner := &MockScanner{
			tokens: []MockToken{
				{Type: PragmaToken, Text: "--sqlcode:include-if feature1"},
				{Type: WhitespaceToken, Text: "\n"},
				{Type: PragmaToken, Text: "--sqlcode:invalid"},
				{Type: WhitespaceToken, Text: "\n"},
				{Type: ReservedWordToken, Text: "CREATE"},
			},
		}
		p := &Pragma{}

		err := p.ParsePragmas(scanner)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "Illegal pragma")
		// First pragma should have been parsed
		assert.Equal(t, []string{"feature1"}, p.pragmas)
	})

	t.Run("handles EOF after pragma", func(t *testing.T) {
		scanner := &MockScanner{
			tokens: []MockToken{
				{Type: PragmaToken, Text: "--sqlcode:include-if feature1"},
				{Type: WhitespaceToken, Text: "\n"},
				{Type: EOFToken, Text: ""},
			},
		}
		p := &Pragma{}

		err := p.ParsePragmas(scanner)

		require.NoError(t, err)
		assert.Equal(t, []string{"feature1"}, p.pragmas)
	})
}

// MockScanner for testing Pragma
type MockScanner struct {
	tokens  []MockToken
	current int
}

type MockToken struct {
	Type     TokenType
	Text     string
	Reserved string
	StartPos Pos
	StopPos  Pos
}

func (m *MockScanner) TokenType() TokenType {
	if m.current >= len(m.tokens) {
		return EOFToken
	}
	return m.tokens[m.current].Type
}

func (m *MockScanner) Token() string {
	if m.current >= len(m.tokens) {
		return ""
	}
	return m.tokens[m.current].Text
}

func (m *MockScanner) TokenLower() string {
	return m.Token()
}

func (m *MockScanner) ReservedWord() string {
	if m.current >= len(m.tokens) {
		return ""
	}
	return m.tokens[m.current].Reserved
}

func (m *MockScanner) Start() Pos {
	if m.current >= len(m.tokens) {
		return Pos{}
	}
	return m.tokens[m.current].StartPos
}

func (m *MockScanner) Stop() Pos {
	if m.current >= len(m.tokens) {
		return Pos{}
	}
	return m.tokens[m.current].StopPos
}

func (m *MockScanner) NextToken() TokenType {
	if m.current < len(m.tokens) {
		m.current++
	}
	return m.TokenType()
}

func (m *MockScanner) NextNonWhitespaceToken() TokenType {
	m.NextToken()
	m.SkipWhitespace()
	return m.TokenType()
}

func (m *MockScanner) NextNonWhitespaceCommentToken() TokenType {
	m.NextToken()
	m.SkipWhitespaceComments()
	return m.TokenType()
}

func (m *MockScanner) SkipWhitespace() {
	for m.TokenType() == WhitespaceToken {
		m.NextToken()
	}
}

func (m *MockScanner) SkipWhitespaceComments() {
	for {
		switch m.TokenType() {
		case WhitespaceToken, MultilineCommentToken, SinglelineCommentToken:
			m.NextToken()
		default:
			return
		}
	}
}

func (m *MockScanner) SetInput(input []byte) {
	// No-op for mock
}

func (m *MockScanner) SetFile(file FileRef) {
	// No-op for mock
}

// Ensure MockScanner implements Scanner interface
var _ Scanner = (*MockScanner)(nil)
