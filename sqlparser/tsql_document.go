package sqlparser

import (
	"fmt"
	"sort"
	"strings"

	mssql "github.com/microsoft/go-mssqldb"
)

type TSqlDocument struct {
	pragmaIncludeIf []string
	creates         []Create
	declares        []Declare
	errors          []Error
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
func (d TSqlDocument) PragmaIncludeIf() []string {
	return d.pragmaIncludeIf
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

func (d *TSqlDocument) parseSinglePragma(s *Scanner) {
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
	d.pragmaIncludeIf = append(d.pragmaIncludeIf, strings.Split(parts[1], ",")...)
}

func (d *TSqlDocument) ParsePragmas(s *Scanner) {
	for s.TokenType() == PragmaToken {
		d.parseSinglePragma(s)
		s.NextNonWhitespaceToken()
	}
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
				doc.recoverToNextStatement(s)
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
				doc.recoverToNextStatement(s)
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
			doc.recoverToNextStatement(s)
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
			doc.recoverToNextStatement(s)
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
			doc.recoverToNextStatement(s)
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
			doc.recoverToNextStatement(s)
		case tt == BatchSeparatorToken:
			doc.parseBatchSeparator(s)
			return true
		default:
			doc.unexpectedTokenError(s)
			doc.recoverToNextStatement(s)
		}
	}
}

func (doc *TSqlDocument) ParseBatch(s *Scanner, isFirst bool) (hasMore bool) {
	var nodes []Unparsed
	var docstring []PosString
	newLineEncounteredInDocstring := false

	var createCountInBatch int

	for {
		tt := s.TokenType()
		switch tt {
		case EOFToken:
			return false
		case WhitespaceToken, MultilineCommentToken:
			nodes = append(nodes, CreateUnparsed(s))
			// do not reset token for a single trailing newline
			t := s.Token()
			if !newLineEncounteredInDocstring && (t == "\n" || t == "\r\n") {
				newLineEncounteredInDocstring = true
			} else {
				docstring = nil
			}
			s.NextToken()
		case SinglelineCommentToken:
			// We build up a list of single line comments for the "docstring";
			// it is reset whenever we encounter something else
			docstring = append(docstring, PosString{s.Start(), s.Token()})
			nodes = append(nodes, CreateUnparsed(s))
			newLineEncounteredInDocstring = false
			s.NextToken()
		case ReservedWordToken:
			switch s.ReservedWord() {
			case "declare":
				// First declare-statement; enter a mode where we assume all contents
				// of batch are declare statements
				if !isFirst {
					doc.addError(s, "'declare' statement only allowed in first batch")
				}
				// regardless of errors, go on and parse as far as we get...
				return doc.parseDeclareBatch(s)
			case "create":
				// should be start of create procedure or create function...
				c := doc.parseCreate(s, createCountInBatch)
				c.Driver = &mssql.Driver{}

				// *prepend* what we saw before getting to the 'create'
				createCountInBatch++
				c.Body = append(nodes, c.Body...)
				c.Docstring = docstring
				doc.creates = append(doc.creates, c)
			default:
				doc.addError(s, "Expected 'declare' or 'create', got: "+s.ReservedWord())
				s.NextToken()
			}
		case BatchSeparatorToken:
			doc.parseBatchSeparator(s)
			return true
		default:
			doc.unexpectedTokenError(s)
			s.NextToken()
			docstring = nil
		}
	}
}

func (d *TSqlDocument) recoverToNextStatementCopying(s *Scanner, target *[]Unparsed) {
	// We hit an unexpected token ... as an heuristic for continuing parsing,
	// skip parsing until we hit a reserved word that starts a statement
	// we recognize
	for {
		NextTokenCopyingWhitespace(s, target)
		switch s.TokenType() {
		case ReservedWordToken:
			switch s.ReservedWord() {
			case "declare", "create", "go":
				return
			}
		case EOFToken:
			return
		default:
			CopyToken(s, target)
		}
	}
}

func (d *TSqlDocument) recoverToNextStatement(s *Scanner) {
	// We hit an unexpected token ... as an heuristic for continuing parsing,
	// skip parsing until we hit a reserved word that starts a statement
	// we recognize
	for {
		s.NextNonWhitespaceCommentToken()
		switch s.TokenType() {
		case ReservedWordToken:
			switch s.ReservedWord() {
			case "declare", "create", "go":
				return
			}
		case EOFToken:
			return
		}
	}
}

// parseCodeschemaName parses `[code] . something`, and returns `something`
// in quoted form (`[something]`). Also copy to `target`. Empty string on error.
// Note: To follow conventions, consume one extra token at the end even if we know
// it fill not be consumed by this function...
func (d *TSqlDocument) parseCodeschemaName(s *Scanner, target *[]Unparsed) PosString {
	CopyToken(s, target)
	NextTokenCopyingWhitespace(s, target)
	if s.TokenType() != DotToken {
		d.addError(s, fmt.Sprintf("[code] must be followed by '.'"))
		d.recoverToNextStatementCopying(s, target)
		return PosString{Value: ""}
	}
	CopyToken(s, target)

	NextTokenCopyingWhitespace(s, target)
	switch s.TokenType() {
	case UnquotedIdentifierToken:
		// To get something uniform for comparison, quote all names
		CopyToken(s, target)
		result := PosString{Pos: s.Start(), Value: "[" + s.Token() + "]"}
		NextTokenCopyingWhitespace(s, target)
		return result
	case QuotedIdentifierToken:
		CopyToken(s, target)
		result := PosString{Pos: s.Start(), Value: s.Token()}
		NextTokenCopyingWhitespace(s, target)
		return result
	default:
		d.addError(s, fmt.Sprintf("[code]. must be followed an identifier"))
		d.recoverToNextStatementCopying(s, target)
		return PosString{Value: ""}
	}
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
		d.recoverToNextStatementCopying(s, &result.Body)
		return
	}
	if (createType == "procedure" || createType == "function") && createCountInBatch > 0 {
		d.addError(s, "a procedure/function must be alone in a batch; use 'go' to split batches")
		d.recoverToNextStatementCopying(s, &result.Body)
		return
	}

	result.CreateType = createType
	CopyToken(s, &result.Body)

	NextTokenCopyingWhitespace(s, &result.Body)

	// Insist on [code].
	if s.TokenType() != QuotedIdentifierToken || s.Token() != "[code]" {
		d.addError(s, fmt.Sprintf("create %s must be followed by [code].", result.CreateType))
		d.recoverToNextStatementCopying(s, &result.Body)
		return
	}
	result.QuotedName = d.parseCodeschemaName(s, &result.Body)
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
					d.recoverToNextStatementCopying(s, &result.Body)
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
			dep := d.parseCodeschemaName(s, &result.Body)
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

// NextTokenCopyingWhitespace is like s.NextToken(), but if whitespace is encountered
// it is simply copied into `target`. Upon return, the scanner is located at a non-whitespace
// token, and target is either unmodified or filled with some whitespace nodes.
func NextTokenCopyingWhitespace(s *Scanner, target *[]Unparsed) {
	for {
		tt := s.NextToken()
		switch tt {
		case EOFToken, BatchSeparatorToken:
			// do not copy
			return
		case WhitespaceToken, MultilineCommentToken, SinglelineCommentToken:
			// copy, and loop around
			CopyToken(s, target)
			continue
		default:
			return
		}
	}

}
