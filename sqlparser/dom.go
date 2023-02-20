package sqlparser

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"io"
	"strings"
)

type Unparsed struct {
	Type        TokenType
	Start, Stop Pos
	RawValue    string
}

func (u Unparsed) WithoutPos() Unparsed {
	return Unparsed{
		Type:     u.Type,
		Start:    Pos{},
		Stop:     Pos{},
		RawValue: u.RawValue,
	}
}

type Declare struct {
	Start        Pos
	Stop         Pos
	VariableName string
	Datatype     Type
	Literal      Unparsed
}

func (d Declare) String() string {
	// silly thing just meant for use for hashing and debugging, not legal SQL..
	return fmt.Sprintf("declare %s %s(%s) = %s",
		d.VariableName,
		d.Datatype.BaseType,
		strings.Join(d.Datatype.Args, ","),
		d.Literal.RawValue)
}

func (d Declare) WithoutPos() Declare {
	return Declare{
		Start:        Pos{},
		Stop:         Pos{},
		VariableName: d.VariableName,
		Datatype:     d.Datatype,
		Literal:      d.Literal.WithoutPos(),
	}
}

// A string that has a Pos-ition in a source document
type PosString struct {
	Pos
	Value string
}

func (p PosString) String() string {
	return p.Value
}

type Create struct {
	CreateType string    // "procedure", "function" or "type"
	QuotedName PosString // proc/func/type name, including []
	Body       []Unparsed
	DependsOn  []PosString
	Docstring  []PosString // comment lines before the create statement. Note: this is also part of Body
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

type Type struct {
	BaseType string
	Args     []string
}

func (t Type) String() (result string) {
	result = t.BaseType
	if len(t.Args) > 0 {
		result = fmt.Sprintf("(%s)", strings.Join(t.Args, ","))
	}
	return result
}

type Error struct {
	Pos     Pos
	Message string
}

func (e Error) Error() string {
	return fmt.Sprintf("%s:%d:%d %s", e.Pos.File, e.Pos.Line, e.Pos.Col, e.Message)
}

func (e Error) WithoutPos() Error {
	return Error{Message: e.Message}
}

type Document struct {
	PragmaIncludeIf []string
	Creates         []Create
	Declares        []Declare
	Errors          []Error
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

// Transform a Document to remove all Position information; this is used
// to 'unclutter' a DOM to more easily write assertions on it.
func (d Document) WithoutPos() Document {
	var cs []Create
	for _, x := range d.Creates {
		cs = append(cs, x.WithoutPos())
	}
	var ds []Declare
	for _, x := range d.Declares {
		ds = append(ds, x.WithoutPos())
	}
	var es []Error
	for _, x := range d.Errors {
		es = append(es, x.WithoutPos())
	}
	return Document{
		Creates:  cs,
		Declares: ds,
		Errors:   es,
	}
}

func (d *Document) Include(other Document) {
	// Do not copy PragmaIncludeIf, since that is local to a single file.
	// Its contents is also present in each Create.
	d.Declares = append(d.Declares, other.Declares...)
	d.Creates = append(d.Creates, other.Creates...)
	d.Errors = append(d.Errors, other.Errors...)
}

func (d *Document) parseSinglePragma(s *Scanner) {
	pragma := strings.TrimSpace(strings.TrimPrefix(s.Token(), "--sqlcode:"))
	if pragma == "" {
		return
	}
	parts := strings.Split(pragma, " ")
	if len(parts) != 2 {
		d.addError(s, "Illegal pragma: "+s.Token())
		return
	}
	if parts[0] != "include-if" {
		d.addError(s, "Illegal pragma: "+s.Token())
		return
	}
	d.PragmaIncludeIf = append(d.PragmaIncludeIf, strings.Split(parts[1], ",")...)
}

func (d *Document) parsePragmas(s *Scanner) {
	for s.TokenType() == PragmaToken {
		d.parseSinglePragma(s)
		s.NextNonWhitespaceToken()
	}
}

func (d Document) Empty() bool {
	return len(d.Creates) > 0 || len(d.Declares) > 0
}
