package sqldocument

type Unparsed struct {
	Type        TokenType
	Start, Stop Pos
	RawValue    string
}

func CreateUnparsed(s Scanner) Unparsed {
	return Unparsed{
		Type:     s.TokenType(),
		Start:    s.Start(),
		Stop:     s.Stop(),
		RawValue: s.Token(),
	}
}
