package sqldocument

import (
	"fmt"
	"strings"
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

	// ReservedTokenHandlers maps reserved words to their processing functions.
	// When a ReservedWordToken is encountered, its lowercase form is
	// looked up here. If found, the handler is called with the scanner
	// and batch.
	// The handler returns 1 if parsing should continue
	// with a new batch (e.g., after processing a batch separator).
	// 		1 = break and parse a new batch
	// 		0 = continue (no return)
	//  	-1 = return false (stop parsing)
	ReservedTokenHandlers map[string]func(Scanner, *Batch) int

	// Errors accumulates parsing errors encountered during batch processing.
	// Errors are collected rather than stopping parsing immediately,
	// allowing partial results even with syntax errors.
	Errors []Error

	// SeparatorHandler is called when a token is encountered.
	SeparatorHandler func(Scanner, *Batch)

	// called with a quoted identifer is encountered within a create statement
	QuotedIdentifierHandler func(Scanner, *Create) (PosString, error)
	QuotedIdentifierPattern string

	// List of statement tokens to fallback on
	StatementTokens []string
}

func NewBatch() *Batch {
	b := &Batch{
		TokenCalls: make(map[string]int, 0),
		Nodes:      make([]Unparsed, 0),
		DocString:  make([]PosString, 0),
		Errors:     make([]Error, 0),
		// Default code pattern
		QuotedIdentifierPattern: "[code]",
		QuotedIdentifierHandler: func(s Scanner, target *Create) (PosString, error) {
			switch s.TokenType() {
			case UnquotedIdentifierToken:
				// To get something uniform for comparison, quote all names
				CopyToken(s, &target.Body)
				return PosString{Pos: s.Start(), Value: "[" + s.Token() + "]"}, nil
			case QuotedIdentifierToken:
				CopyToken(s, &target.Body)
				return PosString{Pos: s.Start(), Value: s.Token()}, nil
			default:
				return PosString{Value: ""}, fmt.Errorf("[code]. must be followed an identifier")
			}
		},
	}

	return b
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
			handler, exists := n.ReservedTokenHandlers[token]
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
			// Note: I'm not yet sure if batch separators is a feature of sqlcode
			// or a feature of the sql dialect.
			if n.SeparatorHandler != nil {
				n.SeparatorHandler(s, n)
			}
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

func (n *Batch) ParseCodeSchemaReference(s Scanner, result *Create) (PosString, error) {
	CopyToken(s, &result.Body)
	NextTokenCopyingWhitespace(s, &result.Body)

	if s.TokenType() != DotToken {
		RecoverToNextStatementCopying(s, &result.Body, n.StatementTokens)
		return PosString{}, fmt.Errorf("%s must be followed by '.'", n.QuotedIdentifierPattern)
	}

	CopyToken(s, &result.Body)
	NextTokenCopyingWhitespace(s, &result.Body)

	if n.QuotedIdentifierHandler == nil {
		panic("QuotedIdentiferHandler must be defined")
	}

	r, err := n.QuotedIdentifierHandler(s, result)
	if err != nil {
		RecoverToNextStatementCopying(s, &result.Body, n.StatementTokens)
		return r, err
	} else {
		NextTokenCopyingWhitespace(s, &result.Body)
	}
	return r, nil
}

// parseCreate parses CREATE PROCEDURE/FUNCTION/TYPE statements.
//
// This is the core of the sqlcode batch parser. It:
//  1. Validates the CREATE type is one we support (procedure/function/type)
//  2. Extracts the object name from <code>.ObjectName syntax
//  3. Copies the entire statement body for later emission
//  4. Tracks dependencies by finding <code>.OtherObject references
//
// The parser is intentionally permissive about SQL syntax details,
// delegating full validation to supported SQL dialects. It focuses on extracting
// the structural information needed for dependency ordering and code generation.
func (n *Batch) ParseCreate(s Scanner, result *Create) error {
	if s.ReservedWord() != "create" {
		panic("illegal use by caller")
	}

	CopyToken(s, &result.Body)
	NextTokenCopyingWhitespace(s, &result.Body)

	rawType := strings.ToLower(s.Token())
	createType, exists := CreateTypeMapping[rawType]

	if !exists {
		RecoverToNextStatementCopying(s, &result.Body, n.StatementTokens)
		return fmt.Errorf("sqlcode only supports creating procedures, functions or types; not `%s`", rawType)
	}

	createCountInBatch, _ := n.TokenCalls["create"]
	if (createType == SQLProcedure || createType == SQLFunction) && createCountInBatch > 0 {
		RecoverToNextStatementCopying(s, &result.Body, n.StatementTokens)
		return fmt.Errorf("a procedure/function must be alone in a batch; use 'go' to split batches")
	}

	result.CreateType = createType
	CopyToken(s, &result.Body)
	NextTokenCopyingWhitespace(s, &result.Body)

	fmt.Printf("%#v\n", result.Body)

	// insist on create <pattern>
	if s.TokenType() != QuotedIdentifierToken || s.TokenLower() != n.QuotedIdentifierPattern {
		return fmt.Errorf("create %s must be followed by %s", rawType, n.QuotedIdentifierPattern)
	}

	var err error
	result.QuotedName, err = n.ParseCodeSchemaReference(s, result)
	if err != nil {
		return fmt.Errorf("QuotedIdentiferHandler returned error: %w", err)
	}
	if result.QuotedName.String() == "" {
		return fmt.Errorf("expected a defined quoted name")
	}

	// we have matched the create "<createType <code>.<quotedname>"
	// we copy the rest until the batch ends; *but* track dependencies
	// + some other details mentioned below

	//firstAs := true // See comment below on rowcount

tailloop:
	for {
		tt := s.TokenType()
		switch {
		case tt == QuotedIdentifierToken && s.TokenLower() == n.QuotedIdentifierPattern:
			// parse a dependency
			dep, err := n.ParseCodeSchemaReference(s, result)
			if err != nil {
				return fmt.Errorf("failed to parse code schema dependency: %w", err)
			}
			if !result.HasDependsOn(dep) {
				result.AddDependency(dep)
			}
		case tt == ReservedWordToken && s.ReservedWord() == "as":
			CopyToken(s, &result.Body)
			NextTokenCopyingWhitespace(s, &result.Body)
			/*
						TODO: Fix and re-enable
						This code add RoutineName for convenience.  So:

						create procedure [code@5420c0269aaf].Test as
						begin
							select 1
						end
						go

						becomes:

						create procedure [code@5420c0269aaf].Test as
						declare @RoutineName nvarchar(128)
						set @RoutineName = 'Test'
						begin
							select 1
						end
						go

						However, for some very strange reason, @@rowcount is 1 with the first version,
						and it is 2 with the second version.
				if firstAs {
					// Add the `RoutineName` token as a convenience, so that we can refer to the procedure/function name
					// from inside the procedure (for example, when logging)
					if result.CreateType == "procedure" {
						procNameToken := Unparsed{
							Type:     OtherToken,
							RawValue: fmt.Sprintf(templateRoutineName, strings.Trim(result.QuotedName.Value, "[]")),
						}
						result.Body = append(result.Body, procNameToken)
					}
					firstAs = false
				}
			*/
		case tt == ReservedWordToken && s.ReservedWord() == "create":
			// So, we're currently parsing 'create ...' and we see another 'create'.
			// We split in two cases depending on the context we are currently in
			// (createType is referring to how we entered this function, *NOT* the
			// `create` statement we are looking at now
			switch createType { // note: this is the *outer* create type, not the one of current scanner position
			case SQLFunction, SQLProcedure:
				// Within a function/procedure we can allow 'create index', 'create table' and nothing
				// else. (Well, only procedures can have them, but we'll leave it to T-SQL to complain
				// about that aspect, not relevant for batch / dependency parsing)
				//
				// What is important is a function/procedure/type isn't started on without a 'go'
				// in between; so we block those 3 from appearing in the same batch
				CopyToken(s, &result.Body)
				NextTokenCopyingWhitespace(s, &result.Body)
				tt2 := s.TokenType()

				if (tt2 == ReservedWordToken && (s.ReservedWord() == "function" ||
					s.ReservedWord() == "procedure")) ||
					(tt2 == UnquotedIdentifierToken &&
						s.TokenLower() == "type") {
					RecoverToNextStatementCopying(s, &result.Body, n.StatementTokens)
					// TODO: note all sql dialects use "go" so slit batches.
					// Q: Do we make this a feature of sqlcode?
					return fmt.Errorf("a procedure/function must be alone in a batch; use 'go' to split batches")
				}
			case SQLType:
				// We allow more than one type creation in a batch; and 'create' can never appear
				// scoped within 'create type'. So at a new create we are done with the previous
				// one, and return it -- the caller can then re-enter this function from the top
				break tailloop
			default:
				panic("assertion failed")
			}
		case tt == BatchSeparatorToken || tt == EOFToken:
			break tailloop
		default:
			CopyToken(s, &result.Body)
			NextTokenCopyingWhitespace(s, &result.Body)
		}
	}

	return nil
}
