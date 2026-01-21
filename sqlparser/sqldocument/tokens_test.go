package sqldocument

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestScannerInput_SetInput(t *testing.T) {
	si := &ScannerInput{}
	si.SetInput([]byte("SELECT * FROM table"))
	assert.Equal(t, "SELECT * FROM table", si.input)
}

func TestScannerInput_SetFile(t *testing.T) {
	si := &ScannerInput{}
	si.SetFile(FileRef("test.sql"))
	assert.Equal(t, FileRef("test.sql"), si.file)
}

func TestTokenScanner_IncIndexes(t *testing.T) {
	ts := &TokenScanner{}
	ts.curIndex = 10
	ts.stopLine = 5
	ts.indexAtStopLine = 8
	ts.reservedWord = "select"

	ts.IncIndexes()

	assert.Equal(t, 10, ts.startIndex)
	assert.Equal(t, 5, ts.startLine)
	assert.Equal(t, 8, ts.indexAtStartLine)
	assert.Equal(t, "", ts.reservedWord)
}

func TestTokenScanner_TokenType(t *testing.T) {
	ts := &TokenScanner{}
	ts.tokenType = ReservedWordToken

	assert.Equal(t, ReservedWordToken, ts.TokenType())
}

func TestTokenScanner_SetToken(t *testing.T) {
	ts := &TokenScanner{}
	ts.SetToken(StringLiteralToken)

	assert.Equal(t, StringLiteralToken, ts.tokenType)
}

func TestTokenScanner_Token(t *testing.T) {
	ts := &TokenScanner{}
	ts.input = "SELECT * FROM table"
	ts.startIndex = 0
	ts.curIndex = 6

	assert.Equal(t, "SELECT", ts.Token())
}

func TestTokenScanner_TokenLower(t *testing.T) {
	ts := &TokenScanner{}
	ts.input = "SELECT * FROM table"
	ts.startIndex = 0
	ts.curIndex = 6

	assert.Equal(t, "select", ts.TokenLower())
}

func TestTokenScanner_TokenRune(t *testing.T) {
	ts := &TokenScanner{}
	ts.input = "SELECT"
	ts.curIndex = 0

	r, w := ts.TokenRune(0)
	assert.Equal(t, 'S', r)
	assert.Equal(t, 1, w)

	r, w = ts.TokenRune(1)
	assert.Equal(t, 'E', r)
	assert.Equal(t, 1, w)
}

func TestTokenScanner_TokenRune_Unicode(t *testing.T) {
	ts := &TokenScanner{}
	ts.input = "日本語"
	ts.curIndex = 0

	r, w := ts.TokenRune(0)
	assert.Equal(t, '日', r)
	assert.Equal(t, 3, w) // UTF-8 encoding of Japanese character
}

func TestTokenScanner_TokenChar(t *testing.T) {
	ts := &TokenScanner{}
	ts.input = "SELECT * FROM table"
	ts.curIndex = 7

	assert.Equal(t, "* FROM table", ts.TokenChar())
}

func TestTokenScanner_ReservedWord(t *testing.T) {
	ts := &TokenScanner{}
	ts.reservedWord = "select"

	assert.Equal(t, "select", ts.ReservedWord())
}

func TestTokenScanner_SetReservedWord(t *testing.T) {
	ts := &TokenScanner{}
	ts.SetReservedWord("from")

	assert.Equal(t, "from", ts.reservedWord)
}

func TestTokenScanner_IncCurIndex(t *testing.T) {
	ts := &TokenScanner{}
	ts.curIndex = 5

	ts.IncCurIndex(3)

	assert.Equal(t, 8, ts.curIndex)
}

func TestTokenScanner_SetCurIndex(t *testing.T) {
	ts := &TokenScanner{}
	ts.input = "SELECT * FROM table"
	ts.curIndex = 5

	ts.SetCurIndex()

	assert.Equal(t, len(ts.input), ts.curIndex)
}

func TestTokenScanner_Start(t *testing.T) {
	ts := &TokenScanner{}
	ts.file = FileRef("test.sql")
	ts.startLine = 2         // 0-indexed
	ts.startIndex = 15       // byte position
	ts.indexAtStartLine = 10 // byte position at start of line

	pos := ts.Start()

	assert.Equal(t, 3, pos.Line) // 1-indexed
	assert.Equal(t, 6, pos.Col)  // 15 - 10 + 1 = 6
	assert.Equal(t, FileRef("test.sql"), pos.File)
}

func TestTokenScanner_IsStartOfLine(t *testing.T) {
	t.Run("at start of line", func(t *testing.T) {
		ts := &TokenScanner{}
		ts.startLine = 0
		ts.stopLine = 1

		assert.True(t, ts.IsStartOfLine())
	})

	t.Run("not at start of line", func(t *testing.T) {
		ts := &TokenScanner{}
		ts.startLine = 1
		ts.stopLine = 1

		assert.False(t, ts.IsStartOfLine())
	})
}

func TestTokenScanner_Stop(t *testing.T) {
	ts := &TokenScanner{}
	ts.file = FileRef("test.sql")
	ts.stopLine = 3         // 0-indexed
	ts.curIndex = 25        // byte position
	ts.indexAtStopLine = 20 // byte position at start of line

	pos := ts.Stop()

	assert.Equal(t, 4, pos.Line) // 1-indexed
	assert.Equal(t, 6, pos.Col)  // 25 - 20 + 1 = 6
	assert.Equal(t, FileRef("test.sql"), pos.File)
}

func TestTokenScanner_BumpLine(t *testing.T) {
	ts := &TokenScanner{}
	ts.curIndex = 10
	ts.stopLine = 0

	ts.BumpLine(5)

	assert.Equal(t, 1, ts.stopLine)
	assert.Equal(t, 16, ts.indexAtStopLine) // 10 + 5 + 1 = 16
}

func TestTokenScanner_ScanMultilineComment(t *testing.T) {
	t.Run("simple comment", func(t *testing.T) {
		ts := &TokenScanner{}
		ts.input = "/* comment */"
		ts.curIndex = 2 // after /*

		tokenType := ts.ScanMultilineComment()

		assert.Equal(t, MultilineCommentToken, tokenType)
		assert.Equal(t, len(ts.input), ts.curIndex)
	})

	t.Run("multiline comment with newlines", func(t *testing.T) {
		ts := &TokenScanner{}
		ts.input = "/* line1\nline2\nline3 */"
		ts.curIndex = 2

		tokenType := ts.ScanMultilineComment()

		assert.Equal(t, MultilineCommentToken, tokenType)
		assert.Equal(t, 2, ts.stopLine) // Two newlines
	})

	t.Run("unterminated comment", func(t *testing.T) {
		ts := &TokenScanner{}
		ts.input = "/* unterminated"
		ts.curIndex = 2

		tokenType := ts.ScanMultilineComment()

		assert.Equal(t, MultilineCommentToken, tokenType)
		assert.Equal(t, len(ts.input), ts.curIndex)
	})
}

func TestTokenScanner_ScanSinglelineComment(t *testing.T) {
	t.Run("regular comment", func(t *testing.T) {
		ts := &TokenScanner{}
		ts.input = "-- this is a comment\nSELECT"
		ts.curIndex = 2 // after --

		tokenType := ts.ScanSinglelineComment()

		assert.Equal(t, SinglelineCommentToken, tokenType)
		assert.Equal(t, 20, ts.curIndex) // before newline
	})

	t.Run("pragma comment", func(t *testing.T) {
		ts := &TokenScanner{}
		ts.input = "--sqlcode:include-if feature\nSELECT"
		ts.curIndex = 2 // after --

		tokenType := ts.ScanSinglelineComment()

		assert.Equal(t, PragmaToken, tokenType)
	})

	t.Run("comment at end of file", func(t *testing.T) {
		ts := &TokenScanner{}
		ts.input = "-- comment at EOF"
		ts.curIndex = 2

		tokenType := ts.ScanSinglelineComment()

		assert.Equal(t, SinglelineCommentToken, tokenType)
		assert.Equal(t, len(ts.input), ts.curIndex)
	})
}

func TestTokenScanner_ScanWhitespace(t *testing.T) {
	t.Run("spaces and tabs", func(t *testing.T) {
		ts := &TokenScanner{}
		ts.input = "   \t  SELECT"
		ts.curIndex = 0

		tokenType := ts.ScanWhitespace()

		assert.Equal(t, WhitespaceToken, tokenType)
		assert.Equal(t, 6, ts.curIndex) // before 'S'
	})

	t.Run("with newlines", func(t *testing.T) {
		ts := &TokenScanner{}
		ts.input = "  \n  \n  SELECT"
		ts.curIndex = 0

		tokenType := ts.ScanWhitespace()

		assert.Equal(t, WhitespaceToken, tokenType)
		assert.Equal(t, 2, ts.stopLine) // Two newlines
	})

	t.Run("only whitespace", func(t *testing.T) {
		ts := &TokenScanner{}
		ts.input = "   \t  \n"
		ts.curIndex = 0

		tokenType := ts.ScanWhitespace()

		assert.Equal(t, WhitespaceToken, tokenType)
		assert.Equal(t, len(ts.input), ts.curIndex)
	})
}

func TestTokenScanner_SkipWhitespace(t *testing.T) {
	callCount := 0
	ts := &TokenScanner{}
	ts.input = "   SELECT"
	ts.tokenType = WhitespaceToken
	ts.NextToken = func() TokenType {
		callCount++
		if callCount == 1 {
			ts.tokenType = ReservedWordToken
		}
		return ts.tokenType
	}

	ts.SkipWhitespace()

	assert.Equal(t, 1, callCount)
	assert.Equal(t, ReservedWordToken, ts.tokenType)
}

func TestTokenScanner_SkipWhitespaceComments(t *testing.T) {
	callCount := 0
	tokens := []TokenType{WhitespaceToken, SinglelineCommentToken, MultilineCommentToken, ReservedWordToken}
	ts := &TokenScanner{}
	ts.tokenType = tokens[0]
	ts.NextToken = func() TokenType {
		callCount++
		if callCount < len(tokens) {
			ts.tokenType = tokens[callCount]
		}
		return ts.tokenType
	}

	ts.SkipWhitespaceComments()

	assert.Equal(t, 3, callCount)
	assert.Equal(t, ReservedWordToken, ts.tokenType)
}

func TestTokenScanner_NextNonWhitespaceToken(t *testing.T) {
	callCount := 0
	tokens := []TokenType{WhitespaceToken, WhitespaceToken, ReservedWordToken}
	ts := &TokenScanner{}
	ts.NextToken = func() TokenType {
		ts.tokenType = tokens[callCount]
		callCount++
		return ts.tokenType
	}

	result := ts.NextNonWhitespaceToken()

	assert.Equal(t, ReservedWordToken, result)
}

func TestTokenScanner_NextNonWhitespaceCommentToken(t *testing.T) {
	callCount := 0
	tokens := []TokenType{WhitespaceToken, SinglelineCommentToken, ReservedWordToken}
	ts := &TokenScanner{}
	ts.NextToken = func() TokenType {
		ts.tokenType = tokens[callCount]
		callCount++
		return ts.tokenType
	}

	result := ts.NextNonWhitespaceCommentToken()

	assert.Equal(t, ReservedWordToken, result)
}

func TestTokenScanner_LineTracking(t *testing.T) {
	ts := &TokenScanner{}
	ts.input = "line1\nline2\nline3"
	ts.curIndex = 0

	// Scan through whitespace to track lines
	for i, r := range ts.input {
		if r == '\n' {
			ts.BumpLine(i - ts.curIndex)
		}
	}

	assert.Equal(t, 2, ts.stopLine) // Two newlines = line index 2 (0-indexed)
}

func TestTokenScanner_PositionCalculation(t *testing.T) {
	ts := &TokenScanner{}
	ts.file = FileRef("test.sql")
	ts.input = "SELECT\nFROM\nWHERE"

	// Simulate being at "WHERE" (third line, after second newline)
	ts.startLine = 2
	ts.stopLine = 2
	ts.startIndex = 12       // byte position of 'W' in "WHERE"
	ts.curIndex = 17         // byte position after "WHERE"
	ts.indexAtStartLine = 12 // byte position at start of third line
	ts.indexAtStopLine = 12

	start := ts.Start()
	stop := ts.Stop()

	assert.Equal(t, 3, start.Line) // 1-indexed line number
	assert.Equal(t, 1, start.Col)  // Column 1 (start of line)
	assert.Equal(t, 3, stop.Line)
	assert.Equal(t, 6, stop.Col) // After "WHERE" (5 chars + 1)
}
