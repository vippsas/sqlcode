package sqlparser

import (
	"fmt"
	"sort"
	"strings"

	mssql "github.com/microsoft/go-mssqldb"
)

var TSQLStatementTokens = []string{"create", "declare", "go"}

type TSqlDocument struct {
	pragmaIncludeIf []string
	creates         []Create
	declares        []Declare
	errors          []Error

	Pragma
}

func (d TSqlDocument) HasErrors() bool {
	return len(d.errors) > 0
}

func (d TSqlDocument) Creates() []Create {
	return d.creates
}

func (d TSqlDocument) Declares() []Declare {
	return d.declares
}

func (d TSqlDocument) Errors() []Error {
	return d.errors
}

func (d *TSqlDocument) Parse(s *Scanner) error {
	err := d.ParsePragmas(s)
	if err != nil {
		d.addError(s, err.Error())
	}

	hasMore := d.parseBatch(s, true)
	for hasMore {
		hasMore = d.parseBatch(s, false)
	}

	return nil
}

func (d *TSqlDocument) Sort() {
	// Do the topological sort; and include any error with it as part
	// of `result`, *not* return it as err
	sortedCreates, errpos, sortErr := TopologicalSort(d.creates)

	if sortErr != nil {
		d.errors = append(d.errors, Error{
			Pos:     errpos,
			Message: sortErr.Error(),
		})
	} else {
		d.creates = sortedCreates
	}
}

// Transform a TSqlDocument to remove all Position information; this is used
// to 'unclutter' a DOM to more easily write assertions on it.
func (d TSqlDocument) WithoutPos() Document {
	var cs []Create
	for _, x := range d.creates {
		cs = append(cs, x.WithoutPos())
	}
	var ds []Declare
	for _, x := range d.declares {
		ds = append(ds, x.WithoutPos())
	}
	var es []Error
	for _, x := range d.errors {
		es = append(es, x.WithoutPos())
	}
	return &TSqlDocument{
		creates:  cs,
		declares: ds,
		errors:   es,
	}
}

func (d *TSqlDocument) Include(other Document) {
	// Do not copy pragmaIncludeIf, since that is local to a single file.
	// Its contents is also present in each Create.
	d.declares = append(d.declares, other.Declares()...)
	d.creates = append(d.creates, other.Creates()...)
	d.errors = append(d.errors, other.Errors()...)
}

func (d TSqlDocument) Empty() bool {
	return len(d.creates) == 0 || len(d.declares) == 0
}

func (d *TSqlDocument) addError(s *Scanner, msg string) {
	d.errors = append(d.errors, Error{
		Pos:     s.Start(),
		Message: msg,
	})
}

func (d *TSqlDocument) unexpectedTokenError(s *Scanner) {
	d.addError(s, "Unexpected: "+s.Token())
}

func (doc *TSqlDocument) parseTypeExpression(s *Scanner) (t Type) {
	parseArgs := func() {
		// parses *after* the initial (; consumes trailing )
		for {
			switch {
			case s.TokenType() == NumberToken:
				t.Args = append(t.Args, s.Token())
			case s.TokenType() == UnquotedIdentifierToken && s.TokenLower() == "max":
				t.Args = append(t.Args, "max")
			default:
				doc.unexpectedTokenError(s)
				RecoverToNextStatement(s, TSQLStatementTokens)
				return
			}
			s.NextNonWhitespaceCommentToken()
			switch {
			case s.TokenType() == CommaToken:
				s.NextNonWhitespaceCommentToken()
				continue
			case s.TokenType() == RightParenToken:
				s.NextNonWhitespaceCommentToken()
				return
			default:
				doc.unexpectedTokenError(s)
				RecoverToNextStatement(s, TSQLStatementTokens)
				return
			}
		}
	}

	if s.TokenType() != UnquotedIdentifierToken {
		panic("assertion failed, bug in caller")
	}
	t.BaseType = s.Token()
	s.NextNonWhitespaceCommentToken()
	if s.TokenType() == LeftParenToken {
		s.NextNonWhitespaceCommentToken()
		parseArgs()
	}
	return
}

func (doc *TSqlDocument) parseDeclare(s *Scanner) (result []Declare) {
	declareStart := s.Start()
	// parse what is *after* the `declare` reserved keyword
loop:
	for {
		if s.TokenType() != VariableIdentifierToken {
			doc.unexpectedTokenError(s)
			RecoverToNextStatement(s, TSQLStatementTokens)
			return
		}

		variableName := s.Token()
		if !strings.HasPrefix(strings.ToLower(variableName), "@enum") &&
			!strings.HasPrefix(strings.ToLower(variableName), "@global") &&
			!strings.HasPrefix(strings.ToLower(variableName), "@const") {
			doc.addError(s, "sqlcode constants needs to have names starting with @Enum, @Global or @Const: "+variableName)
		}

		s.NextNonWhitespaceCommentToken()
		var variableType Type
		switch s.TokenType() {
		case EqualToken:
			doc.addError(s, "sqlcode constants needs a type declared explicitly")
			s.NextNonWhitespaceCommentToken()
		case UnquotedIdentifierToken:
			variableType = doc.parseTypeExpression(s)
		}

		if s.TokenType() != EqualToken {
			doc.addError(s, "sqlcode constants needs to be assigned at once using =")
			RecoverToNextStatement(s, TSQLStatementTokens)
		}

		switch s.NextNonWhitespaceCommentToken() {
		case NumberToken, NVarcharLiteralToken, VarcharLiteralToken:
			declare := Declare{
				Start:        declareStart,
				Stop:         s.Stop(),
				VariableName: variableName,
				Datatype:     variableType,
				Literal:      CreateUnparsed(s),
			}
			result = append(result, declare)
		default:
			doc.unexpectedTokenError(s)
			RecoverToNextStatement(s, TSQLStatementTokens)
			return
		}

		switch s.NextNonWhitespaceCommentToken() {
		case CommaToken:
			s.NextNonWhitespaceCommentToken()
			continue
		case SemicolonToken:
			s.NextNonWhitespaceCommentToken()
			break loop
		default:
			break loop
		}
	}
	if len(result) == 0 {
		doc.addError(s, "incorrect syntax; no variables successfully declared")
	}
	return
}

func (doc *TSqlDocument) parseBatchSeparator(s *Scanner) {
	// just saw a 'go'; just make sure there's nothing bad trailing it
	// (if there is, convert to errors and move on until the line is consumed
	errorEmitted := false
	for {
		switch s.NextToken() {
		case WhitespaceToken:
			continue
		case MalformedBatchSeparatorToken:
			if !errorEmitted {
				doc.addError(s, "`go` should be alone on a line without any comments")
				errorEmitted = true
			}
			continue
		default:
			return
		}
	}
}

func (doc *TSqlDocument) parseDeclareBatch(s *Scanner) (hasMore bool) {
	if s.ReservedWord() != "declare" {
		panic("assertion failed, incorrect use in caller")
	}
	for {
		tt := s.TokenType()
		switch {
		case tt == EOFToken:
			return false
		case tt == ReservedWordToken && s.ReservedWord() == "declare":
			s.NextNonWhitespaceCommentToken()
			d := doc.parseDeclare(s)
			doc.declares = append(doc.declares, d...)
		case tt == ReservedWordToken && s.ReservedWord() != "declare":
			doc.addError(s, "Only 'declare' allowed in this batch")
			RecoverToNextStatement(s, TSQLStatementTokens)
		case tt == BatchSeparatorToken:
			doc.parseBatchSeparator(s)
			return true
		default:
			doc.unexpectedTokenError(s)
			RecoverToNextStatement(s, TSQLStatementTokens)
		}
	}
}

func (doc *TSqlDocument) parseBatch(s *Scanner, isFirst bool) (hasMore bool) {
	nodes := &Nodes{
		TokenHandlers: map[string]func(*Scanner, *Nodes) bool{
			"declare": func(s *Scanner, n *Nodes) bool {
				// First declare-statement; enter a mode where we assume all contents
				// of batch are declare statements
				if !isFirst {
					doc.addError(s, "'declare' statement only allowed in first batch")
				}

				// regardless of errors, go on and parse as far as we get...
				return doc.parseDeclareBatch(s)
			},
			"create": func(s *Scanner, n *Nodes) bool {
				// should be start of create procedure or create function...
				c := doc.parseCreate(s, n.CreateStatements)
				c.Driver = &mssql.Driver{}

				// *prepend* what we saw before getting to the 'create'
				n.CreateStatements++
				c.Body = append(n.Nodes, c.Body...)
				c.Docstring = n.DocString
				doc.creates = append(doc.creates, c)
				return false
			},
		},
	}
	hasMore = nodes.Parse(s)
	if nodes.HasErrors() {
		doc.errors = append(doc.errors, nodes.Errors...)
	}

	return hasMore
}

// parseCreate parses anything that starts with "create". Position is
// *on* the create token.
// At this stage in sqlcode parser development we're only interested
// in procedures/functions/types as opaque blocks of SQL code where
// we only track dependencies between them and their declared name;
// so we treat them with the same code. We consume until the end of
// the batch; only one declaration allowed per batch. Everything
// parsed here will also be added to `batch`. On any error, copying
// to batch stops / becomes erratic..
func (d *TSqlDocument) parseCreate(s *Scanner, createCountInBatch int) (result Create) {
	if s.ReservedWord() != "create" {
		panic("illegal use by caller")
	}
	CopyToken(s, &result.Body)

	NextTokenCopyingWhitespace(s, &result.Body)

	createType := strings.ToLower(s.Token())
	if !(createType == "procedure" || createType == "function" || createType == "type") {
		d.addError(s, fmt.Sprintf("sqlcode only supports creating procedures, functions or types; not `%s`", createType))
		RecoverToNextStatementCopying(s, &result.Body, TSQLStatementTokens)
		return
	}
	if (createType == "procedure" || createType == "function") && createCountInBatch > 0 {
		d.addError(s, "a procedure/function must be alone in a batch; use 'go' to split batches")
		RecoverToNextStatementCopying(s, &result.Body, TSQLStatementTokens)
		return
	}

	result.CreateType = createType
	CopyToken(s, &result.Body)

	NextTokenCopyingWhitespace(s, &result.Body)

	// Insist on [code].
	if s.TokenType() != QuotedIdentifierToken || s.Token() != "[code]" {
		d.addError(s, fmt.Sprintf("create %s must be followed by [code].", result.CreateType))
		RecoverToNextStatementCopying(s, &result.Body, TSQLStatementTokens)
		return
	}
	var err error
	result.QuotedName, err = ParseCodeschemaName(s, &result.Body, TSQLStatementTokens)
	if err != nil {
		d.addError(s, err.Error())
	}
	if result.QuotedName.String() == "" {
		return
	}

	// We have matched "create <createType> [code].<quotedName>"; at this
	// point we copy the rest until the batch ends; *but* track dependencies
	// + some other details mentioned below

	//firstAs := true // See comment below on rowcount

tailloop:
	for {
		tt := s.TokenType()
		switch {
		case tt == ReservedWordToken && s.ReservedWord() == "create":
			// So, we're currently parsing 'create ...' and we see another 'create'.
			// We split in two cases depending on the context we are currently in
			// (createType is referring to how we entered this function, *NOT* the
			// `create` statement we are looking at now
			switch createType { // note: this is the *outer* create type, not the one of current scanner position
			case "function", "procedure":
				// Within a function/procedure we can allow 'create index', 'create table' and nothing
				// else. (Well, only procedures can have them, but we'll leave it to T-SQL to complain
				// about that aspect, not relevant for batch / dependency parsing)
				//
				// What is important is a function/procedure/type isn't started on without a 'go'
				// in between; so we block those 3 from appearing in the same batch
				CopyToken(s, &result.Body)
				NextTokenCopyingWhitespace(s, &result.Body)
				tt2 := s.TokenType()

				if (tt2 == ReservedWordToken && (s.ReservedWord() == "function" || s.ReservedWord() == "procedure")) ||
					(tt2 == UnquotedIdentifierToken && s.TokenLower() == "type") {
					RecoverToNextStatementCopying(s, &result.Body, TSQLStatementTokens)
					d.addError(s, "a procedure/function must be alone in a batch; use 'go' to split batches")
					return
				}
			case "type":
				// We allow more than one type creation in a batch; and 'create' can never appear
				// scoped within 'create type'. So at a new create we are done with the previous
				// one, and return it -- the caller can then re-enter this function from the top
				break tailloop
			default:
				panic("assertion failed")
			}

		case tt == EOFToken || tt == BatchSeparatorToken:
			break tailloop
		case tt == QuotedIdentifierToken && s.Token() == "[code]":
			// Parse a dependency
			dep, err := ParseCodeschemaName(s, &result.Body, TSQLStatementTokens)
			if err != nil {
				d.addError(s, err.Error())
			}
			found := false
			for _, existing := range result.DependsOn {
				if existing.Value == dep.Value {
					found = true
					break
				}
			}
			if !found {
				result.DependsOn = append(result.DependsOn, dep)
			}
		case tt == ReservedWordToken && s.Token() == "as":
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

		default:
			CopyToken(s, &result.Body)
			NextTokenCopyingWhitespace(s, &result.Body)
		}
	}

	sort.Slice(result.DependsOn, func(i, j int) bool {
		return result.DependsOn[i].Value < result.DependsOn[j].Value
	})
	return
}
