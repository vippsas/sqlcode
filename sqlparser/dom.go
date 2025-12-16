package sqlparser

import (
	"fmt"
	"strings"
)

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
