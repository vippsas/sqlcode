package sqldocument

import (
	"fmt"
)

// Batch represents a single SQL batch during parsing.
//
// A batch is a sequence of SQL statements separated by a token in SQL.
// The Batch struct accumulates tokens and metadata while parsing,
// allowing handlers to process specific statement types (CREATE, DECLARE, etc.).
//
// The parsing flow:
//  1. Tokens are read sequentially via Parse()
//  2. Whitespace and comments are accumulated in Nodes
//  3. Single-line comments build up the DocString for documentation
//  4. When a reserved word is encountered, its handler is invoked
//  5. Handlers can consume additional tokens and update the batch state
type Batch struct {
	// Nodes contains all unparsed tokens accumulated before a statement.
	// These are prepended to CREATE statement bodies to preserve
	// leading whitespace and comments in the output.
	Nodes []Unparsed

	// DocString holds single-line comments that precede a statement.
	// These comments form the documentation for procedures/functions.
	// Reset when non-comment, non-whitespace tokens are encountered.
	DocString []PosString

	// TokenCalls tracks how many Token handlers have been
	// called in this batch. Can be used to enforce the validation rules
	// that procedures and functions must be alone in their batch.
	TokenCalls map[string]int

	// TokenHandlers maps reserved words to their processing functions.
	// When a ReservedWordToken is encountered, its lowercase form is
	// looked up here. If found, the handler is called with the scanner
	// and batch.
	// The handler returns 1 if parsing should continue
	// with a new batch (e.g., after processing a batch separator).
	// 		1 = break and parse a new batch
	// 		0 = continue (no return)
	//  	-1 = return false (stop parsing)
	TokenHandlers map[string]func(Scanner, *Batch) int

	// Errors accumulates parsing errors encountered during batch processing.
	// Errors are collected rather than stopping parsing immediately,
	// allowing partial results even with syntax errors.
	Errors []Error

	// BatchSeparatorHandler is called when a token is encountered.
	BatchSeparatorHandler func(Scanner, *Batch)
}

func NewBatch() *Batch {
	return &Batch{
		TokenCalls: make(map[string]int, 0),
		Nodes:      make([]Unparsed, 0),
		DocString:  make([]PosString, 0),
		Errors:     make([]Error, 0),
	}
}

func (n *Batch) Create(s Scanner) {
	n.Nodes = append(n.Nodes, CreateUnparsed(s))
}

func (n *Batch) HasErrors() bool {
	return len(n.Errors) > 0
}

// Parse processes tokens from the scanner and builds the batch's node list.
func (n *Batch) Parse(s Scanner) bool {
	newLineEncounteredInDocstring := false

	for {
		tt := s.TokenType()
		switch tt {
		case EOFToken:
			return false
		case WhitespaceToken, MultilineCommentToken:
			n.Create(s)
			// do not reset token for a single trailing newline
			t := s.Token()
			if !newLineEncounteredInDocstring && (t == "\n" || t == "\r\n") {
				newLineEncounteredInDocstring = true
			} else {
				n.DocString = nil
			}
			s.NextToken()
		case SinglelineCommentToken:
			// We build up a list of single line comments for the "docstring"
			// it is reset when we hit non-comment, non-whitespace tokens
			n.DocString = append(n.DocString, PosString{s.Start(), s.Token()})
			n.Create(s)
			newLineEncounteredInDocstring = false
			s.NextToken()
		case ReservedWordToken:
			token := s.ReservedWord()
			handler, exists := n.TokenHandlers[token]
			if exists {
				// Invoke the handler for this reserved word
				// The handler is responsible for advancing the scanner
				// and updating the batch as needed.
				rt := handler(s, n)
				n.TokenCalls[token] += 1
				if rt == 1 {
					return true
				}
				if rt == -1 {
					return false
				}
			} else {
				s.NextToken()
			}
		case BatchSeparatorToken:
			n.BatchSeparatorHandler(s, n)
			return true
		default:
			n.Errors = append(n.Errors, Error{
				s.Start(), fmt.Sprintf("Unexpected token: %s", s.Token()),
			})
			s.NextToken()
			// reset docstring on unexpected token
			n.DocString = nil
		}
	}
}
