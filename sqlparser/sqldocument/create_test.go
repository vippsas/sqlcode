package sqldocument

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreate_DocstringAsString(t *testing.T) {
	create := Create{
		Docstring: []PosString{
			{Value: "-- This is a comment"},
			{Value: "-- Another comment"},
		},
	}

	expected := "-- This is a comment\n-- Another comment"
	assert.Equal(t, expected, create.DocstringAsString())
}

func TestCreate_DocstringYamldoc(t *testing.T) {
	t.Run("Valid YAML docstring", func(t *testing.T) {
		create := Create{
			Docstring: []PosString{
				{Value: "--! key1: value1"},
				{Value: "--! key2: value2"},
			},
		}

		expected := "key1: value1\nkey2: value2"
		yamlDoc, err := create.DocstringYamldoc()
		assert.NoError(t, err)
		assert.Equal(t, expected, yamlDoc)
	})

	t.Run("Invalid YAML docstring (missing space)", func(t *testing.T) {
		create := Create{
			Docstring: []PosString{
				{Value: "--!key1: value1"},
			},
		}

		_, err := create.DocstringYamldoc()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "missing space after `--!`")
	})

	t.Run("Invalid YAML docstring (non-continuous)", func(t *testing.T) {
		create := Create{
			Docstring: []PosString{
				{Value: "--! key1: value1"},
				{Value: "-- This breaks the YAML doc"},
			},
		}

		_, err := create.DocstringYamldoc()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "must continue until create statement")
	})
}

func TestCreate_ParseYamlInDocstring(t *testing.T) {
	create := Create{
		Docstring: []PosString{
			{Value: "--! key1: value1"},
			{Value: "--! key2: value2"},
		},
	}

	var result map[string]string
	err := create.ParseYamlInDocstring(&result)
	assert.NoError(t, err)
	assert.Equal(t, "value1", result["key1"])
	assert.Equal(t, "value2", result["key2"])
}

func TestCreate_Serialize(t *testing.T) {
	create := Create{
		Body: []Unparsed{
			{RawValue: "CREATE PROCEDURE [code].Test AS"},
			{RawValue: " BEGIN"},
			{RawValue: " SELECT 1;"},
			{RawValue: " END;"},
		},
	}

	var buf strings.Builder
	err := create.Serialize(&buf)
	assert.NoError(t, err)

	expected := "CREATE PROCEDURE [code].Test AS BEGIN SELECT 1; END;"
	assert.Equal(t, expected, buf.String())
}

func TestCreate_SerializeBytes(t *testing.T) {
	create := Create{
		Body: []Unparsed{
			{RawValue: "CREATE PROCEDURE [code].Test AS"},
			{RawValue: " BEGIN"},
			{RawValue: " SELECT 1;"},
			{RawValue: " END;"},
		},
	}

	var buf strings.Builder
	err := create.SerializeBytes(&buf)
	assert.NoError(t, err)

	expected := "CREATE PROCEDURE [code].Test AS BEGIN SELECT 1; END;"
	assert.Equal(t, expected, buf.String())
}

func TestCreate_String(t *testing.T) {
	create := Create{
		Body: []Unparsed{
			{RawValue: "CREATE PROCEDURE [code].Test AS"},
			{RawValue: " BEGIN"},
			{RawValue: " SELECT 1;"},
			{RawValue: " END;"},
		},
	}

	expected := "CREATE PROCEDURE [code].Test AS BEGIN SELECT 1; END;"
	assert.Equal(t, expected, create.String())
}

func TestCreate_DependsOnStrings(t *testing.T) {
	create := Create{
		DependsOn: []PosString{
			{Value: "[Dependency1]"},
			{Value: "[Dependency2]"},
		},
	}

	expected := []string{"[Dependency1]", "[Dependency2]"}
	assert.Equal(t, expected, create.DependsOnStrings())
}
