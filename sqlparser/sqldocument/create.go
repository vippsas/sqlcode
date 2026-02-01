package sqldocument

import (
	"database/sql/driver"
	"io"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type CreateType int

const (
	SQLProcedure CreateType = iota
	SQLFunction
	SQLType
)

var CreateTypeMapping = map[string]CreateType{
	"procedure": SQLProcedure,
	"function":  SQLFunction,
	"type":      SQLType,
}

// Create represents a SQL CREATE statement, such as a procedure, function, or type.
//
// This struct captures the metadata and body of the CREATE statement, including:
//   - The type of object being created (procedure, function, or type)
//   - The quoted name of the object
//   - The full body of the CREATE statement
//   - Any dependencies on other objects
//   - Docstring comments preceding the CREATE statement
//   - The SQL driver associated with the statement
type Create struct {
	// CreateType specifies the type of object being created.
	// Valid values are "procedure", "function", or "type".
	CreateType CreateType

	// QuotedName is the name of the object being created, including square brackets.
	// For example: [MyProcedure].
	QuotedName PosString

	// Body contains the full body of the CREATE statement, including all tokens.
	// This includes the CREATE keyword, object definition, and any trailing comments.
	Body []Unparsed

	// DependsOn lists other objects that this CREATE statement depends on.
	// Each dependency is represented as a PosString.
	DependsOn []PosString

	// Docstring contains comments that precede the CREATE statement.
	// These comments are typically used for documentation purposes.
	Docstring []PosString

	// Driver specifies the SQL driver associated with this CREATE statement.
	// This is used to determine dialect-specific behavior.
	Driver driver.Driver
}

func (c Create) DocstringAsString() string {
	var result []string
	for _, line := range c.Docstring {
		result = append(result, line.Value)
	}
	return strings.Join(result, "\n")
}

func (c Create) DocstringYamldoc() (string, error) {
	var yamldoc []string
	parsing := false
	for _, line := range c.Docstring {
		if strings.HasPrefix(line.Value, "--!") {
			parsing = true
			if !strings.HasPrefix(line.Value, "--! ") {
				return "", Error{line.Pos, "YAML document in docstring; missing space after `--!`"}
			}
			yamldoc = append(yamldoc, line.Value[4:])
		} else if parsing {
			return "", Error{line.Pos, "once embedded yaml document is started (lines prefixed with `--!`), it must continue until create statement"}
		}
	}
	return strings.Join(yamldoc, "\n"), nil
}

func (c Create) ParseYamlInDocstring(out any) error {
	yamldoc, err := c.DocstringYamldoc()
	if err != nil {
		return err
	}
	return yaml.Unmarshal([]byte(yamldoc), out)
}

func (c Create) Serialize(w io.StringWriter) error {
	for _, l := range c.Body {
		if _, err := w.WriteString(l.RawValue); err != nil {
			return err
		}
	}
	return nil
}

func (c Create) SerializeBytes(w io.Writer) error {
	for _, l := range c.Body {
		if _, err := w.Write([]byte(l.RawValue)); err != nil {
			return err
		}
	}
	return nil
}

func (c Create) String() string {
	var buf strings.Builder
	err := c.Serialize(&buf)
	if err != nil {
		panic(err)
	}
	return buf.String()
}

func (c Create) DependsOnStrings() (result []string) {
	for _, x := range c.DependsOn {
		result = append(result, x.Value)
	}
	return
}

func (c Create) HasDependsOn(dep PosString) bool {
	for _, existing := range c.DependsOn {
		if existing.Value == dep.Value {
			return true
		}
	}
	return false
}

func (c *Create) AddDependency(dep PosString) {
	c.DependsOn = append(c.DependsOn, dep)

	sort.Slice(c.DependsOn, func(i, j int) bool {
		return c.DependsOn[i].Value < c.DependsOn[j].Value
	})
}
