package sqlparser

import (
	"database/sql/driver"
	"io"
	"strings"

	"gopkg.in/yaml.v3"
)

type Create struct {
	CreateType string    // "procedure", "function" or "type"
	QuotedName PosString // proc/func/type name, including []
	Body       []Unparsed
	DependsOn  []PosString
	Docstring  []PosString   // comment lines before the create statement. Note: this is also part of Body
	Driver     driver.Driver // the sql driver this document is intended for
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

func (c Create) WithoutPos() Create {
	var body []Unparsed
	for _, x := range c.Body {
		body = append(body, x.WithoutPos())
	}
	return Create{
		CreateType: c.CreateType,
		QuotedName: c.QuotedName,
		DependsOn:  c.DependsOn,
		Body:       body,
	}
}

func (c Create) DependsOnStrings() (result []string) {
	for _, x := range c.DependsOn {
		result = append(result, x.Value)
	}
	return
}
