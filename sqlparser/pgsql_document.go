package sqlparser

import (
	"fmt"

	"github.com/jackc/pgx/v5/stdlib"
)

var PGSQLStatementTokens = []string{"create"}

type PGSqlDocument struct {
	creates []Create
	errors  []Error

	Pragma
}

func (d PGSqlDocument) HasErrors() bool {
	return len(d.errors) > 0
}

func (d *PGSqlDocument) Parse(s *Scanner) error {
	err := d.ParsePragmas(s)
	if err != nil {
		d.errors = append(d.errors, Error{s.Start(), err.Error()})
	}

	return nil
}

func (d PGSqlDocument) Creates() []Create {
	return d.creates
}

// Not yet implemented
func (d PGSqlDocument) Declares() []Declare {
	return nil
}

func (d PGSqlDocument) Errors() []Error {
	return d.errors
}

func (d PGSqlDocument) Empty() bool {
	return len(d.creates) == 0
}

func (d PGSqlDocument) Sort() {

}

func (d PGSqlDocument) Include(other Document) {

}

func (d PGSqlDocument) WithoutPos() Document {
	return &PGSqlDocument{}
}

// No GO batch separator:
//
//	PostgreSQL uses semicolons (;) to separate statements, not GO.
//	Multiple CREATE statements can exist in the same file.
//
// No top-level DECLARE:
//
//	In PostgreSQL, DECLARE is only used inside function/procedure bodies within BEGIN...END blocks, not as top-level batch statements.
//
// Multiple CREATEs per batch:
//
//	Unlike T-SQL which requires procedures/functions to be alone in a batch, PostgreSQL allows multiple CREATE statements separated by semicolons.
//
// Semicolon handling:
//
//	The semicolon is a statement terminator, not a batch separator, so parsing continues after encountering one.
//
// Dollar quoting:
//
//	PostgreSQL uses $$ or $tag$ for quoting function bodies instead of BEGIN...END (this would be handled in parseCreate).
//
// CREATE OR REPLACE:
//
//	PostgreSQL commonly uses CREATE OR REPLACE which would need special handling in parseCreate.
//
// Schema qualification:
//
//	PostgreSQL uses schema.object notation rather than [schema].[object].
func (doc *PGSqlDocument) parseBatch(s *Scanner, isFirst bool) (hasMore bool) {
	batch := &Batch{
		TokenHandlers: map[string]func(*Scanner, *Batch) bool{
			"create": func(s *Scanner, n *Batch) bool {
				// Parse CREATE FUNCTION, CREATE PROCEDURE, CREATE TYPE, etc.
				c := doc.parseCreate(s, n.CreateStatements)
				c.Driver = &stdlib.Driver{}

				// Prepend any leading comments/whitespace
				c.Body = append(n.Nodes, c.Body...)
				c.Docstring = n.DocString
				doc.creates = append(doc.creates, c)

				return false
			},
		},
	}

	hasMore = batch.Parse(s)
	if batch.HasErrors() {
		doc.errors = append(doc.errors, batch.Errors...)
	}

	return hasMore

	// var nodes []Unparsed
	// var docstring []PosString
	// newLineEncounteredInDocstring := false

	// for {
	// 	tt := s.TokenType()
	// 	switch tt {
	// 	case EOFToken:
	// 		return false
	// 	case WhitespaceToken, MultilineCommentToken:
	// 		nodes = append(nodes, CreateUnparsed(s))
	// 		// do not reset docstring for a single trailing newline
	// 		t := s.Token()
	// 		if !newLineEncounteredInDocstring && (t == "\n" || t == "\r\n") {
	// 			newLineEncounteredInDocstring = true
	// 		} else {
	// 			docstring = nil
	// 		}
	// 		s.NextToken()
	// 	case SinglelineCommentToken:
	// 		// Build up a list of single line comments for the "docstring";
	// 		// it is reset whenever we encounter something else
	// 		docstring = append(docstring, PosString{s.Start(), s.Token()})
	// 		nodes = append(nodes, CreateUnparsed(s))
	// 		newLineEncounteredInDocstring = false
	// 		s.NextToken()
	// 	case ReservedWordToken:
	// 		switch s.ReservedWord() {
	// 		case "declare":
	// 			// PostgreSQL doesn't have top-level DECLARE batches like T-SQL
	// 			// DECLARE is only used inside function/procedure bodies
	// 			if isFirst {
	// 				doc.addError(s, "PostgreSQL 'declare' is used inside function bodies, not as top-level batch statements")
	// 			}
	// 			nodes = append(nodes, CreateUnparsed(s))
	// 			s.NextToken()
	// 			docstring = nil
	// 		case "create":
	// 			// Parse CREATE FUNCTION, CREATE PROCEDURE, CREATE TYPE, etc.
	// 			createStart := len(doc.creates)
	// 			c := doc.parseCreate(s, createStart)
	// 			c.Driver = &stdlib.Driver{}

	// 			// Prepend any leading comments/whitespace
	// 			c.Body = append(nodes, c.Body...)
	// 			c.Docstring = docstring
	// 			doc.creates = append(doc.creates, c)

	// 			// Reset for next statement
	// 			nodes = nil
	// 			docstring = nil
	// 			newLineEncounteredInDocstring = false
	// 		default:
	// 			doc.addError(s, "Expected 'create', got: "+s.ReservedWord())
	// 			s.NextToken()
	// 			docstring = nil
	// 		}
	// 	case SemicolonToken:
	// 		// PostgreSQL uses semicolons as statement terminators
	// 		// Multiple CREATE statements can exist in same file
	// 		nodes = append(nodes, CreateUnparsed(s))
	// 		s.NextToken()
	// 		// Continue parsing - don't return like T-SQL does with GO
	// 	case BatchSeparatorToken:
	// 		// PostgreSQL doesn't use GO batch separators
	// 		// Q: Do we want to use GO batch separators as a feature of sqlcode?
	// 		doc.addError(s, "PostgreSQL does not use 'GO' batch separators; use semicolons instead")
	// 		s.NextToken()
	// 		docstring = nil
	// 	default:
	// 		doc.addError(s, fmt.Sprintf("Unexpected token in PostgreSQL document: %s", s.Token()))
	// 		s.NextToken()
	// 		docstring = nil
	// 	}
	// }
}

// parseCreate parses PostgreSQL CREATE statements (FUNCTION, PROCEDURE, TYPE, etc.)
// Position is *on* the CREATE token.
//
// PostgreSQL CREATE syntax differences from T-SQL:
// - Supports CREATE OR REPLACE for functions/procedures
// - Uses dollar quoting ($$...$$) or $tag$...$tag$ for function bodies
// - Schema qualification uses dot notation: schema.function_name
// - Double-quoted identifiers preserve case: "MyFunction"
// - Function parameters use different syntax: func(param1 type1, param2 type2)
// - RETURNS clause specifies return type
// - LANGUAGE clause (plpgsql, sql, etc.) is required
// - Function characteristics: IMMUTABLE, STABLE, VOLATILE, PARALLEL SAFE, etc.
//
// We parse until we hit a semicolon or EOF, tracking dependencies on other objects.
func (doc *PGSqlDocument) parseCreate(s *Scanner, createCountInBatch int) (result Create) {
	var body []Unparsed

	// Copy the CREATE token
	CopyToken(s, &body)
	s.NextNonWhitespaceCommentToken()

	// Check for OR REPLACE
	// NOTE: "or replace" doesn't make sense within sqlcode as this will be created within a new
	// schema.
	if s.TokenType() == ReservedWordToken && s.ReservedWord() == "or" {
		CopyToken(s, &body)
		s.NextNonWhitespaceCommentToken()

		if s.TokenType() == ReservedWordToken && s.ReservedWord() == "replace" {
			CopyToken(s, &body)
			s.NextNonWhitespaceCommentToken()
		} else {
			doc.addError(s, "Expected 'REPLACE' after 'OR'")
			RecoverToNextStatementCopying(s, &body, PGSQLStatementTokens)
			result.Body = body
			return
		}
	}

	// Parse the object type (FUNCTION, PROCEDURE, TYPE, etc.)
	if s.TokenType() != ReservedWordToken {
		doc.addError(s, "Expected object type after CREATE (e.g., FUNCTION, PROCEDURE, TYPE)")
		RecoverToNextStatementCopying(s, &body, PGSQLStatementTokens)
		result.Body = body
		return
	}

	createType := s.ReservedWord()
	result.CreateType = createType
	CopyToken(s, &body)
	s.NextNonWhitespaceCommentToken()

	// Validate supported CREATE types
	switch createType {
	case "function", "procedure", "type":
		// Supported types
	default:
		doc.addError(s, fmt.Sprintf("Unsupported CREATE type for PostgreSQL: %s", createType))
		RecoverToNextStatementCopying(s, &body, PGSQLStatementTokens)
		result.Body = body
		return
	}

	// Insist on [code] to provide the ability for sqlcode to patch function bodies
	// with references to other sqlcode objects.
	if s.TokenType() != QuotedIdentifierToken || s.Token() != "[code]" {
		doc.addError(s, fmt.Sprintf("create %s must be followed by [code].", result.CreateType))
		RecoverToNextStatementCopying(s, &result.Body, PGSQLStatementTokens)
		return
	}
	var err error
	result.QuotedName, err = ParseCodeschemaName(s, &result.Body, PGSQLStatementTokens)
	if err != nil {
		doc.addError(s, err.Error())
	}
	if result.QuotedName.String() == "" {
		return
	}

	// Parse function/procedure signature or type definition
	switch createType {
	case "function", "procedure":
		doc.parseFunctionSignature(s, &body, &result)
	case "type":
		doc.parseTypeDefinition(s, &body, &result)
	}

	// Parse the rest of the CREATE statement body until semicolon or EOF
	doc.parseCreateBody(s, &body, &result)

	result.Body = body
	return
}

// parseQualifiedName parses schema-qualified or simple object names
// Supports: simple_name, schema.name, "Quoted Name", schema."Quoted Name"
func (doc *PGSqlDocument) parseQualifiedName(s *Scanner, body *[]Unparsed) string {
	var nameParts []string

	for {
		switch s.TokenType() {
		case UnquotedIdentifierToken:
			nameParts = append(nameParts, s.Token())
			CopyToken(s, body)
			s.NextNonWhitespaceCommentToken()
		case QuotedIdentifierToken:
			// PostgreSQL uses double quotes for case-sensitive identifiers
			nameParts = append(nameParts, s.Token())
			CopyToken(s, body)
			s.NextNonWhitespaceCommentToken()
		default:
			if len(nameParts) == 0 {
				return ""
			}
			// Return the last part as the object name (without schema)
			return nameParts[len(nameParts)-1]
		}

		// Check for dot separator (schema.object)
		if s.TokenType() == DotToken {
			CopyToken(s, body)
			s.NextNonWhitespaceCommentToken()
			continue
		}

		break
	}

	if len(nameParts) == 0 {
		return ""
	}
	return nameParts[len(nameParts)-1]
}

// parseFunctionSignature parses function/procedure parameters and RETURNS clause
func (doc *PGSqlDocument) parseFunctionSignature(s *Scanner, body *[]Unparsed, result *Create) {
	// Expect opening parenthesis for parameters
	if s.TokenType() != LeftParenToken {
		doc.addError(s, "Expected '(' for function parameters")
		return
	}

	CopyToken(s, body)
	s.NextNonWhitespaceCommentToken()

	// Parse parameters until closing parenthesis
	parenDepth := 1
	for parenDepth > 0 {
		switch s.TokenType() {
		case EOFToken:
			doc.addError(s, "Unexpected EOF in function parameters")
			return
		case LeftParenToken:
			parenDepth++
			CopyToken(s, body)
			s.NextToken()
		case RightParenToken:
			parenDepth--
			CopyToken(s, body)
			s.NextToken()
		case SemicolonToken:
			doc.addError(s, "Unexpected semicolon in function parameters")
			return
		default:
			CopyToken(s, body)
			s.NextToken()
		}
	}

	s.SkipWhitespaceComments()

	// Parse RETURNS clause (for functions, not procedures)
	if result.CreateType == "function" {
		if s.TokenType() == ReservedWordToken && s.ReservedWord() == "returns" {
			CopyToken(s, body)
			s.NextNonWhitespaceCommentToken()

			// Handle RETURNS TABLE(...)
			if s.TokenType() == ReservedWordToken && s.ReservedWord() == "table" {
				CopyToken(s, body)
				s.NextNonWhitespaceCommentToken()

				if s.TokenType() == LeftParenToken {
					doc.parseReturnTable(s, body)
				}
			} else {
				// Parse simple return type
				doc.parseTypeExpression(s, body)
			}
		}
	}
}

// parseReturnTable parses RETURNS TABLE(...) syntax
func (doc *PGSqlDocument) parseReturnTable(s *Scanner, body *[]Unparsed) {
	parenDepth := 0
	for {
		switch s.TokenType() {
		case EOFToken, SemicolonToken:
			return
		case LeftParenToken:
			parenDepth++
		case RightParenToken:
			parenDepth--
			CopyToken(s, body)
			s.NextToken()
			if parenDepth == 0 {
				return
			}
			continue
		}
		CopyToken(s, body)
		s.NextToken()
	}
}

// parseTypeExpression parses PostgreSQL type expressions
// Supports: int, integer, text, varchar(n), numeric(p,s), arrays (int[]), etc.
func (doc *PGSqlDocument) parseTypeExpression(s *Scanner, body *[]Unparsed) {
	// Parse base type
	if s.TokenType() != UnquotedIdentifierToken && s.TokenType() != ReservedWordToken {
		return
	}

	CopyToken(s, body)
	s.NextNonWhitespaceCommentToken()

	// Handle array notation: type[]
	// if s.TokenType() == LeftBracketToken {
	// 	CopyToken(s, body)
	// 	s.NextNonWhitespaceCommentToken()

	// 	if s.TokenType() == RightBracketToken {
	// 		CopyToken(s, body)
	// 		s.NextNonWhitespaceCommentToken()
	// 	}
	// }

	// Handle type parameters: varchar(100), numeric(10,2)
	if s.TokenType() == LeftParenToken {
		parenDepth := 1
		CopyToken(s, body)
		s.NextToken()

		for parenDepth > 0 {
			switch s.TokenType() {
			case EOFToken, SemicolonToken:
				return
			case LeftParenToken:
				parenDepth++
			case RightParenToken:
				parenDepth--
			}
			CopyToken(s, body)
			s.NextToken()
		}
	}
}

// parseTypeDefinition parses CREATE TYPE syntax
// Supports: ENUM, composite types, range types
func (doc *PGSqlDocument) parseTypeDefinition(s *Scanner, body *[]Unparsed, result *Create) {
	// TYPE definitions use AS keyword
	if s.TokenType() == ReservedWordToken && s.ReservedWord() == "as" {
		CopyToken(s, body)
		s.NextNonWhitespaceCommentToken()

		// Check for ENUM, RANGE, or composite type
		if s.TokenType() == ReservedWordToken {
			typeKind := s.ReservedWord()
			switch typeKind {
			case "enum", "range":
				CopyToken(s, body)
				s.NextNonWhitespaceCommentToken()
			}
		}
	}
}

// parseCreateBody parses the body of a CREATE statement
// Handles dollar-quoted strings, tracks dependencies, continues until semicolon/EOF
func (doc *PGSqlDocument) parseCreateBody(s *Scanner, body *[]Unparsed, result *Create) {
	dollarQuoteDepth := 0
	var currentDollarTag string

	for {
		switch s.TokenType() {
		case EOFToken:
			return
		case SemicolonToken:
			// Statement terminator - we're done
			CopyToken(s, body)
			s.NextToken()
			return
		case DollarQuotedStringStartToken:
			// PostgreSQL dollar quoting: $$...$$  or $tag$...$tag$
			currentDollarTag = s.Token()
			dollarQuoteDepth++
			CopyToken(s, body)
			s.NextToken()
		case DollarQuotedStringEndToken:
			if s.Token() == currentDollarTag {
				dollarQuoteDepth--
			}
			CopyToken(s, body)
			s.NextToken()
			if dollarQuoteDepth == 0 {
				currentDollarTag = ""
			}
		case UnquotedIdentifierToken, QuotedIdentifierToken:
			// Track dependencies on tables/views/functions
			// In PostgreSQL, identifiers can be qualified: schema.object
			identifier := s.Token()

			// Check if this might be a dependency (after FROM, JOIN, etc.)
			if doc.mightBeDependency(s) {
				// Extract just the object name (without schema prefix)
				objectName := doc.extractObjectName(identifier)
				result.DependsOn = append(result.DependsOn, PosString{s.Start(), objectName})
			}

			CopyToken(s, body)
			s.NextToken()
		default:
			CopyToken(s, body)
			s.NextToken()
		}
	}
}

// mightBeDependency checks if current context suggests a table/view/function reference
func (doc *PGSqlDocument) mightBeDependency(s *Scanner) bool {
	// Simple heuristic: look back for FROM, JOIN, INTO, etc.
	// This would need to track parse context for accurate dependency detection
	return false // Placeholder - implement context-aware dependency tracking
}

// extractObjectName extracts object name from schema-qualified identifier
func (doc *PGSqlDocument) extractObjectName(identifier string) string {
	// Handle schema.object notation
	// For now, return as-is; proper implementation would split on dot
	return identifier
}

func (doc *PGSqlDocument) addError(s *Scanner, err string) {
	doc.errors = append(doc.errors, Error{
		s.Start(), err,
	})
}

func (doc *PGSqlDocument) parseDeclareBatch(s *Scanner) (hasMore bool) {
	// PostgreSQL doesn't have top-level DECLARE batches like T-SQL
	// DECLARE is only used inside function/procedure bodies (in BEGIN...END blocks)
	doc.addError(s, "PostgreSQL does not support top-level DECLARE statements outside of function bodies")
	RecoverToNextStatement(s, PGSQLStatementTokens)
	return false
}

func (doc *PGSqlDocument) parseBatchSeparator(s *Scanner) {
	// PostgreSQL doesn't use GO batch separators
	doc.addError(s, "PostgreSQL does not use 'GO' batch separators; use semicolons")
	s.NextToken()
}
