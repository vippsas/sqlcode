package sqlparser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDocumentFromExtension(t *testing.T) {
	t.Run("returns TSqlDocument for .sql extension", func(t *testing.T) {
		doc := NewDocumentFromExtension(".sql")

		_, ok := doc.(*TSqlDocument)
		assert.True(t, ok, "Expected TSqlDocument type")
		assert.NotNil(t, doc)
	})

	t.Run("returns PGSqlDocument for .pgsql extension", func(t *testing.T) {
		doc := NewDocumentFromExtension(".pgsql")

		_, ok := doc.(*PGSqlDocument)
		assert.True(t, ok, "Expected PGSqlDocument type")
		assert.NotNil(t, doc)
	})

	t.Run("panics for unsupported extension", func(t *testing.T) {
		assert.Panics(t, func() {
			NewDocumentFromExtension(".txt")
		}, "Expected panic for unsupported extension")
	})

	t.Run("panics for empty extension", func(t *testing.T) {
		assert.Panics(t, func() {
			NewDocumentFromExtension("")
		}, "Expected panic for empty extension")
	})

	t.Run("panics for unknown SQL extension", func(t *testing.T) {
		assert.Panics(t, func() {
			NewDocumentFromExtension(".mysql")
		}, "Expected panic for .mysql extension")
	})

	t.Run("extension matching is case insensitive", func(t *testing.T) {
		assert.Panics(t, func() {
			NewDocumentFromExtension(".SQL")
		}, "Expected panic for uppercase .SQL")
	})

	t.Run("returned documents implement Document interface", func(t *testing.T) {
		sqlDoc := NewDocumentFromExtension(".sql")
		pgsqlDoc := NewDocumentFromExtension(".pgsql")
		require.NotEqual(t, sqlDoc, pgsqlDoc)
	})
}
