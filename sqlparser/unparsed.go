package sqlparser

type Unparsed struct {
	Type        TokenType
	Start, Stop Pos
	RawValue    string
}

func CreateUnparsed(s *Scanner) Unparsed {
	return Unparsed{
		Type:     s.TokenType(),
		Start:    s.Start(),
		Stop:     s.Stop(),
		RawValue: s.Token(),
	}
}

func (u Unparsed) WithoutPos() Unparsed {
	return Unparsed{
		Type:     u.Type,
		Start:    Pos{},
		Stop:     Pos{},
		RawValue: u.RawValue,
	}
}
