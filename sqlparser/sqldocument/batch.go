package sqldocument

import (
	"fmt"
)

type Batch struct {
	Nodes               []Unparsed
	DocString           []PosString
	CreateStatements    int
	TokenHandlers       map[string]func(Scanner, *Batch) bool
	Errors              []Error
	BatchSeparatorToken TokenType
}

func (n *Batch) Create(s Scanner) {
	n.Nodes = append(n.Nodes, CreateUnparsed(s))
}

func (n *Batch) HasErrors() bool {
	return len(n.Errors) > 0
}

// Parse processes tokens from the scanner and builds the batch's node list.
// It uses CommonTokenType() for dialect-agnostic token handling.
func (n *Batch) Parse(s Scanner) bool {
	newLineEncounteredInDocstring := false

	for {
		// Use CommonTokenType for dialect-agnostic switching
		tt := s.NextToken()
		switch tt {
		case EOFToken:
			return false
		case WhitespaceToken, MultilineCommentToken:
			n.Create(s)
			t := s.Token()
			if !newLineEncounteredInDocstring && (t == "\n" || t == "\r\n") {
				newLineEncounteredInDocstring = true
			} else {
				n.DocString = nil
			}
			s.NextToken()
		case SinglelineCommentToken:
			n.DocString = append(n.DocString, PosString{s.Start(), s.Token()})
			n.Create(s)
			newLineEncounteredInDocstring = false
			s.NextToken()
		case ReservedWordToken:
			token := s.ReservedWord()
			handler, exists := n.TokenHandlers[token]
			if !exists {
				n.Errors = append(n.Errors, Error{
					s.Start(), fmt.Sprintf("Expected , got: %s", token),
				})
				s.NextToken()
			} else {
				if handler(s, n) {
					return true
				}
			}
		case BatchSeparatorToken:
			errorEmitted := false
			for {
				switch s.NextToken() {
				case WhitespaceToken:
					continue
				case MalformedBatchSeparatorToken:
					if !errorEmitted {
						n.Errors = append(n.Errors, Error{
							s.Start(), "`go` should be alone on a line without any comments",
						})
						errorEmitted = true
					}
					continue
				default:
					return true
				}
			}
		default:
			n.Errors = append(n.Errors, Error{
				s.Start(), fmt.Sprintf("Unexpected token: %s", s.Token()),
			})
			s.NextToken()
			n.DocString = nil
		}
	}
}
