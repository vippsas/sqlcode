package mssql

import (
	"fmt"
	"sort"
	"strings"

	mssql "github.com/microsoft/go-mssqldb"
	"github.com/vippsas/sqlcode/sqlparser/sqldocument"
)

// TSQLStatementTokens defines the keywords that start new statements.
// Used by error recovery to find a safe point to resume parsing.
var TSQLStatementTokens = []string{"create", "declare", "go"}

// TSqlDocument represents a T-SQL source file.
//
// The document contains:
//   - creates: CREATE PROCEDURE/FUNCTION/TYPE statements with dependency info
//   - declares: DECLARE statements for sqlcode constants (@Enum*, @Global*, @Const*)
//   - errors: Syntax and semantic errors encountered during parsing
//   - pragmaIncludeIf: Conditional compilation directives from --sqlcode:include-if
//
// Parsing follows T-SQL batch semantics where batches are separated by GO.
// The first batch may contain DECLARE statements for constants.
// Subsequent batches contain CREATE statements for database objects.
type TSqlDocument struct {
	pragmaIncludeIf []string
	creates         []sqldocument.Create
	declares        []sqldocument.Declare
	errors          []sqldocument.Error

	sqldocument.Pragma
}

// Parse processes a T-SQL source file from the given input.
//
// Parsing proceeds in phases:
//  1. Parse pragma comments at the file start (--sqlcode:...)
//  2. Parse batches sequentially, separated by GO
//
// The first batch has special rules: it may contain DECLARE statements
// for sqlcode constants. CREATE statements may appear in any batch,
// but procedures/functions must be alone in their batch (T-SQL requirement).
//
// Errors are accumulated in the document rather than stopping parsing,
// allowing partial results even with syntax errors.
func (d *TSqlDocument) Parse(input []byte, file sqldocument.FileRef) error {
	s := &Scanner{}
	s.SetInput(input)
	s.SetFile(file)

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

func (d TSqlDocument) HasErrors() bool {
	return len(d.errors) > 0
}

func (d TSqlDocument) Creates() []sqldocument.Create {
	return d.creates
}

func (d TSqlDocument) Declares() []sqldocument.Declare {
	return d.declares
}

func (d TSqlDocument) Errors() []sqldocument.Error {
	return d.errors
}

func (d *TSqlDocument) Sort() {
	// Do the topological sort; and include any error with it as part
	// of `result`, *not* return it as err
	sortedCreates, errpos, sortErr := sqldocument.TopologicalSort(d.creates)

	if sortErr != nil {
		d.errors = append(d.errors, sqldocument.Error{
			Pos:     errpos,
			Message: sortErr.Error(),
		})
	} else {
		d.creates = sortedCreates
	}
}

func (d *TSqlDocument) Include(other sqldocument.Document) {
	// Do not copy pragmaIncludeIf, since that is local to a single file.
	// Its contents is also present in each Create.
	d.declares = append(d.declares, other.Declares()...)
	d.creates = append(d.creates, other.Creates()...)
	d.errors = append(d.errors, other.Errors()...)
}

func (d TSqlDocument) Empty() bool {
	return len(d.creates) == 0 || len(d.declares) == 0
}

func (d *TSqlDocument) addError(s sqldocument.Scanner, msg string) {
	d.errors = append(d.errors, sqldocument.Error{
		Pos:     s.Start(),
		Message: msg,
	})
}

func (d *TSqlDocument) unexpectedTokenError(s sqldocument.Scanner) {
	d.addError(s, "Unexpected: "+s.Token())
}

func (doc *TSqlDocument) parseTypeExpression(s sqldocument.Scanner) (t sqldocument.Type) {
	parseArgs := func() {
		// parses *after* the initial (; consumes trailing )
		for {
			switch {
			case s.TokenType() == sqldocument.NumberToken:
				t.Args = append(t.Args, s.Token())
			case s.TokenType() == sqldocument.UnquotedIdentifierToken && s.TokenLower() == "max":
				t.Args = append(t.Args, "max")
			default:
				doc.unexpectedTokenError(s)
				sqldocument.RecoverToNextStatement(s, TSQLStatementTokens)
				return
			}
			s.NextNonWhitespaceCommentToken()
			switch {
			case s.TokenType() == sqldocument.CommaToken:
				s.NextNonWhitespaceCommentToken()
				continue
			case s.TokenType() == sqldocument.RightParenToken:
				s.NextNonWhitespaceCommentToken()
				return
			default:
				doc.unexpectedTokenError(s)
				sqldocument.RecoverToNextStatement(s, TSQLStatementTokens)
				return
			}
		}
	}

	if s.TokenType() != sqldocument.UnquotedIdentifierToken {
		panic("assertion failed, bug in caller")
	}
	t.BaseType = s.Token()
	s.NextNonWhitespaceCommentToken()
	if s.TokenType() == sqldocument.LeftParenToken {
		s.NextNonWhitespaceCommentToken()
		parseArgs()
	}
	return
}

func (doc *TSqlDocument) parseDeclare(s sqldocument.Scanner) (result []sqldocument.Declare) {
	declareStart := s.Start()
	// parse what is *after* the `declare` reserved keyword
loop:
	for {
		if s.TokenType() != sqldocument.VariableIdentifierToken {
			doc.unexpectedTokenError(s)
			sqldocument.RecoverToNextStatement(s, TSQLStatementTokens)
			return
		}

		variableName := s.Token()
		if !strings.HasPrefix(strings.ToLower(variableName), "@enum") &&
			!strings.HasPrefix(strings.ToLower(variableName), "@global") &&
			!strings.HasPrefix(strings.ToLower(variableName), "@const") {
			doc.addError(s, "sqlcode constants needs to have names starting with @Enum, @Global or @Const: "+variableName)
		}

		s.NextNonWhitespaceCommentToken()
		var variableType sqldocument.Type
		switch s.TokenType() {
		case sqldocument.EqualToken:
			doc.addError(s, "sqlcode constants needs a type declared explicitly")
			s.NextNonWhitespaceCommentToken()
		case sqldocument.UnquotedIdentifierToken:
			variableType = doc.parseTypeExpression(s)
		}

		if s.TokenType() != sqldocument.EqualToken {
			doc.addError(s, "sqlcode constants needs to be assigned at once using =")
			sqldocument.RecoverToNextStatement(s, TSQLStatementTokens)
		}

		switch s.NextNonWhitespaceCommentToken() {
		case sqldocument.NumberToken, NVarcharLiteralToken, VarcharLiteralToken:
			declare := sqldocument.Declare{
				Start:        declareStart,
				Stop:         s.Stop(),
				VariableName: variableName,
				Datatype:     variableType,
				Literal:      sqldocument.CreateUnparsed(s),
			}
			result = append(result, declare)
		default:
			doc.unexpectedTokenError(s)
			sqldocument.RecoverToNextStatement(s, TSQLStatementTokens)
			return
		}

		switch s.NextNonWhitespaceCommentToken() {
		case sqldocument.CommaToken:
			s.NextNonWhitespaceCommentToken()
			continue
		case sqldocument.SemicolonToken:
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

func (doc *TSqlDocument) parseBatchSeparator(s sqldocument.Scanner) {
	// just saw a 'go'; just make sure there's nothing bad trailing it
	// (if there is, convert to errors and move on until the line is consumed
	errorEmitted := false
	// continuously process tokens until a non-whitespace, non-malformed token is encountered.
	for {
		switch s.NextToken() {
		case sqldocument.WhitespaceToken:
			continue
		case sqldocument.MalformedBatchSeparatorToken:
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

func (doc *TSqlDocument) parseDeclareBatch(s sqldocument.Scanner) (hasMore bool) {
	if s.ReservedWord() != "declare" {
		panic("assertion failed, incorrect use in caller")
	}
	for {
		tt := s.TokenType()
		switch {
		case tt == sqldocument.EOFToken:
			return false
		case tt == sqldocument.ReservedWordToken && s.ReservedWord() == "declare":
			s.NextNonWhitespaceCommentToken()
			d := doc.parseDeclare(s)
			doc.declares = append(doc.declares, d...)
		case tt == sqldocument.ReservedWordToken && s.ReservedWord() != "declare":
			doc.addError(s, "Only 'declare' allowed in this batch")
			sqldocument.RecoverToNextStatement(s, TSQLStatementTokens)
		case tt == sqldocument.BatchSeparatorToken:
			doc.parseBatchSeparator(s)
			return true
		default:
			doc.unexpectedTokenError(s)
			sqldocument.RecoverToNextStatement(s, TSQLStatementTokens)
		}
	}
}

// parseBatch processes a single T-SQL batch (content between GO separators).
//
// Batch processing strategy:
//   - Track tokens before the first significant statement for docstrings
//   - Dispatch to specialized parsers based on statement type (CREATE, DECLARE)
//   - Handle batch separator (GO) to signal batch boundary
//
// The isFirst parameter indicates whether this is the first batch in the file,
// which affects whether DECLARE statements are allowed.
func (doc *TSqlDocument) parseBatch(s sqldocument.Scanner, isFirst bool) (hasMore bool) {
	batch := &sqldocument.Batch{
		TokenHandlers: map[string]func(sqldocument.Scanner, *sqldocument.Batch) bool{
			"declare": func(s sqldocument.Scanner, n *sqldocument.Batch) bool {
				// First declare-statement; enter a mode where we assume all contents
				// of batch are declare statements
				if !isFirst {
					doc.addError(s, "'declare' statement only allowed in first batch")
				}

				// regardless of errors, go on and parse as far as we get...
				return doc.parseDeclareBatch(s)
			},
			"create": func(s sqldocument.Scanner, n *sqldocument.Batch) bool {
				// should be start of create procedure or create function...
				c := doc.parseCreate(s, n.CreateStatements)
				c.Driver = &mssql.Driver{}

				// *prepend* what we saw before getting to the 'create'
				n.CreateStatements++
				c.Body = append(n.Nodes, c.Body...)
				c.Docstring = n.DocString
				doc.creates = append(doc.creates, c)

				// fmt.Printf("%#v\n", s)
				// fmt.Printf("%#v\n", n)
				// fmt.Printf("%#v\n", doc)
				return false
			},
		},
	}
	hasMore = batch.Parse(s)
	if batch.HasErrors() {
		doc.errors = append(doc.errors, batch.Errors...)
	}

	return hasMore
}

// parseCreate parses CREATE PROCEDURE/FUNCTION/TYPE statements.
//
// This is the core of the sqlcode parser. It:
//  1. Validates the CREATE type is one we support (procedure/function/type)
//  2. Extracts the object name from [code].ObjectName syntax
//  3. Copies the entire statement body for later emission
//  4. Tracks dependencies by finding [code].OtherObject references
//
// The parser is intentionally permissive about T-SQL syntax details,
// delegating full validation to SQL Server. It focuses on extracting
// the structural information needed for dependency ordering and code generation.
//
// Parameters:
//   - s: Scanner positioned on the CREATE keyword
//   - createCountInBatch: Number of CREATE statements already seen in this batch
//     (used to enforce "one procedure/function per batch" rule)
func (d *TSqlDocument) parseCreate(s sqldocument.Scanner, createCountInBatch int) (result sqldocument.Create) {
	if s.ReservedWord() != "create" {
		panic("illegal use by caller")
	}
	sqldocument.CopyToken(s, &result.Body)

	sqldocument.NextTokenCopyingWhitespace(s, &result.Body)

	createType := strings.ToLower(s.Token())
	if !(createType == "procedure" || createType == "function" || createType == "type") {
		d.addError(s, fmt.Sprintf("sqlcode only supports creating procedures, functions or types; not `%s`", createType))
		sqldocument.RecoverToNextStatementCopying(s, &result.Body, TSQLStatementTokens)
		return
	}
	if (createType == "procedure" || createType == "function") && createCountInBatch > 0 {
		d.addError(s, "a procedure/function must be alone in a batch; use 'go' to split batches")
		sqldocument.RecoverToNextStatementCopying(s, &result.Body, TSQLStatementTokens)
		return
	}

	result.CreateType = createType
	sqldocument.CopyToken(s, &result.Body)

	sqldocument.NextTokenCopyingWhitespace(s, &result.Body)

	// Insist on [code].
	if s.TokenType() != sqldocument.QuotedIdentifierToken || s.Token() != "[code]" {
		d.addError(s, fmt.Sprintf("create %s must be followed by [code].", result.CreateType))
		sqldocument.RecoverToNextStatementCopying(s, &result.Body, TSQLStatementTokens)
		return
	}
	var err error
	result.QuotedName, err = sqldocument.ParseCodeschemaName(s, &result.Body, TSQLStatementTokens)
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
		case tt == sqldocument.ReservedWordToken && s.ReservedWord() == "create":
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
				sqldocument.CopyToken(s, &result.Body)
				sqldocument.NextTokenCopyingWhitespace(s, &result.Body)
				tt2 := s.TokenType()

				if (tt2 == sqldocument.ReservedWordToken && (s.ReservedWord() == "function" || s.ReservedWord() == "procedure")) ||
					(tt2 == sqldocument.UnquotedIdentifierToken && s.TokenLower() == "type") {
					sqldocument.RecoverToNextStatementCopying(s, &result.Body, TSQLStatementTokens)
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

		case tt == sqldocument.EOFToken || tt == sqldocument.BatchSeparatorToken:
			break tailloop
		case tt == sqldocument.QuotedIdentifierToken && s.Token() == "[code]":
			// Parse a dependency
			dep, err := sqldocument.ParseCodeschemaName(s, &result.Body, TSQLStatementTokens)
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
		case tt == sqldocument.ReservedWordToken && s.Token() == "as":
			sqldocument.CopyToken(s, &result.Body)
			sqldocument.NextTokenCopyingWhitespace(s, &result.Body)
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
			sqldocument.CopyToken(s, &result.Body)
			sqldocument.NextTokenCopyingWhitespace(s, &result.Body)
		}
	}

	sort.Slice(result.DependsOn, func(i, j int) bool {
		return result.DependsOn[i].Value < result.DependsOn[j].Value
	})
	return
}
