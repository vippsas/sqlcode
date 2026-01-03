package sqldocument

// import (
// 	"fmt"
// 	"testing"

// 	"github.com/stretchr/testify/assert"
// )

// func TestDocument_parseCodeschemaName(t *testing.T) {
// 	t.Run("parses unquoted identifier", func(t *testing.T) {
// 		s := NewScanner("test.sql", "[code].TestProc")
// 		s.NextToken()
// 		var target []Unparsed

// 		result, err := ParseCodeschemaName(s, &target, nil)
// 		assert.NoError(t, err)
// 		assert.Equal(t, "[TestProc]", result.Value)
// 		assert.NotEmpty(t, target)
// 	})

// 	t.Run("parses quoted identifier", func(t *testing.T) {
// 		s := NewScanner("test.sql", "[code].[Test Proc]")
// 		s.NextToken()
// 		var target []Unparsed

// 		result, err := ParseCodeschemaName(s, &target, nil)
// 		assert.NoError(t, err)

// 		assert.Equal(t, "[Test Proc]", result.Value)
// 	})

// 	t.Run("errors on missing dot", func(t *testing.T) {
// 		s := NewScanner("test.sql", "[code] TestProc")
// 		s.NextToken()
// 		var target []Unparsed

// 		result, err := ParseCodeschemaName(s, &target, nil)
// 		assert.Error(t, err)

// 		assert.Equal(t, "", result.Value)
// 		assert.ErrorContains(t, err, "must be followed by '.'")
// 	})

// 	t.Run("errors on missing identifier", func(t *testing.T) {
// 		s := NewScanner("test.sql", "[code].123")
// 		s.NextToken()
// 		var target []Unparsed

// 		result, err := ParseCodeschemaName(s, &target, nil)

// 		assert.Error(t, err)
// 		assert.Equal(t, "", result.Value)
// 		assert.ErrorContains(t, err, "must be followed an identifier")
// 	})
// }

// func TestDocument_recoverToNextStatement(t *testing.T) {
// 	t.Run("recovers to declare", func(t *testing.T) {
// 		s := NewScanner("test.sql", "invalid tokens here declare @x int = 1")
// 		s.NextToken()

// 		RecoverToNextStatement(s, []string{"declare"})

// 		fmt.Printf("%#v\n", s)

// 		assert.Equal(t, ReservedWordToken, s.TokenType())
// 		assert.Equal(t, "declare", s.ReservedWord())
// 	})

// 	t.Run("recovers to create", func(t *testing.T) {
// 		s := NewScanner("test.sql", "bad stuff create procedure")
// 		s.NextToken()

// 		RecoverToNextStatement(s, []string{"create"})

// 		assert.Equal(t, ReservedWordToken, s.TokenType())
// 		assert.Equal(t, "create", s.ReservedWord())
// 	})

// 	t.Run("stops at EOF", func(t *testing.T) {
// 		s := NewScanner("test.sql", "no keywords")
// 		s.NextToken()

// 		RecoverToNextStatement(s, []string{})

// 		assert.Equal(t, EOFToken, s.TokenType())
// 	})
// }

// func TestDocument_recoverToNextStatementCopying(t *testing.T) {
// 	t.Run("copies tokens while recovering", func(t *testing.T) {
// 		s := NewScanner("test.sql", "bad token declare")
// 		s.NextToken()
// 		var target []Unparsed

// 		RecoverToNextStatementCopying(s, &target, []string{"declare"})

// 		assert.NotEmpty(t, target)
// 		assert.Equal(t, "declare", s.ReservedWord())
// 	})
// }
