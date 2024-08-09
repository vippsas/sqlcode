// Simple non-performant recursive descent parser for purposes of sqlcode; currently only
// supports the special @Enum declarations used by sqlcode. We only allow
// these on the top, and parsing will stop
// without any errors at the point hitting anything else.
package sqlparser

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io/fs"
	"regexp"
	"sort"
	"strings"
)

func CopyToken(s *Scanner, target *[]Unparsed) {
	*target = append(*target, CreateUnparsed(s))
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

// AdvanceAndCopy is like NextToken; advance to next token that is not whitespace and return
// Note: The 'go' and EOF tokens are *not* copied
func AdvanceAndCopy(s *Scanner, target *[]Unparsed) {
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
			// copy, and return
			CopyToken(s, target)
			return
		}
	}
}

func CreateUnparsed(s *Scanner) Unparsed {
	return Unparsed{
		Type:     s.TokenType(),
		Start:    s.Start(),
		Stop:     s.Stop(),
		RawValue: s.Token(),
	}
}

func (d *Document) addError(s *Scanner, msg string) {
	d.Errors = append(d.Errors, Error{
		Pos:     s.Start(),
		Message: msg,
	})
}

func (d *Document) unexpectedTokenError(s *Scanner) {
	d.addError(s, "Unexpected: "+s.Token())
}

func (doc *Document) parseTypeExpression(s *Scanner) (t Type) {
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

func (doc *Document) parseDeclare(s *Scanner) (result []Declare) {
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
			result = append(result, Declare{
				Start:        declareStart,
				Stop:         s.Stop(),
				VariableName: variableName,
				Datatype:     variableType,
				Literal:      CreateUnparsed(s),
			})
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

func (doc *Document) parseBatchSeparator(s *Scanner) {
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

func (doc *Document) parseDeclareBatch(s *Scanner) (hasMore bool) {
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
			doc.Declares = append(doc.Declares, d...)
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

func (doc *Document) parseBatch(s *Scanner, isFirst bool) (hasMore bool) {
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
				// *prepend* what we saw before getting to the 'create'
				createCountInBatch++
				c.Body = append(nodes, c.Body...)
				c.Docstring = docstring
				doc.Creates = append(doc.Creates, c)
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

func (d *Document) recoverToNextStatementCopying(s *Scanner, target *[]Unparsed) {
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

func (d *Document) recoverToNextStatement(s *Scanner) {
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
func (d *Document) parseCodeschemaName(s *Scanner, target *[]Unparsed) PosString {
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
func (d *Document) parseCreate(s *Scanner, createCountInBatch int) (result Create) {
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

	firstAs := true

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
			if firstAs { //&& tt == ReservedWordToken && s.Token() != "begin" {
				// Add the `RoutineName` token to a procedure
				// The `RoutineName` token is just a convenience so that
				// we can refer to the procedure/function name inside the procedure
				// (for example, when logging)
				if result.CreateType == "procedure" {
					procNameToken := Unparsed{
						Type:     OtherToken,
						RawValue: fmt.Sprintf("DECLARE @RoutineName NVARCHAR(128)\nSET @RoutineName = '%s'\n", strings.Trim(result.QuotedName.Value, "[]")),
					}
					result.Body = append(result.Body, procNameToken)
				}
				firstAs = false
			}

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

func Parse(s *Scanner, result *Document) {
	// Top-level parse; this focuses on splitting into "batches" separated
	// by 'go'.

	// CONVENTION:
	// All functions should expect `s` positioned on what they are documented
	// to consume/parse.
	//
	// Functions typically consume *after* the keyword that triggered their
	// invoication; e.g. parseCreate parses from first non-whitespace-token
	// *after* `create`.
	//
	// On return, `s` is positioned at the token that starts the next statement/
	// sub-expression. In particular trailing ';' and whitespace has been consumed.
	//
	// `s` will typically never be positioned on whitespace except in
	// whitespace-preserving parsing

	s.NextNonWhitespaceToken()
	result.parsePragmas(s)
	hasMore := result.parseBatch(s, true)
	for hasMore {
		hasMore = result.parseBatch(s, false)
	}
	return
}

func ParseString(filename FileRef, input string) (result Document) {
	Parse(&Scanner{input: input, file: filename}, &result)
	return
}

// ParseFileystems iterates through a list of filesystems and parses all files
// matching `*.sql`, determines which one are sqlcode files from the contents,
// and returns the combination of all of them.
//
// err will only return errors related to filesystems/reading. Errors
// related to parsing/sorting will be in result.Errors.
//
// ParseFilesystems will also sort create statements topologically.
func ParseFilesystems(fslst []fs.FS, includeTags []string) (filenames []string, result Document, err error) {
	// We are being passed several *filesystems* here. It may be easy to pass in the same
	// directory twice but that should not be encouraged, so if we get the same hash from
	// two files, return an error. Only files containing [code] in some way will be
	// considered here anyway

	hashes := make(map[[32]byte]string)

	for fidx, fsys := range fslst {
		// WalkDir is in lexical order according to docs, so output should be stable
		err = fs.WalkDir(fsys, ".",
			func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				// Skip over any hidden directories; in particular .git
				if strings.HasPrefix(path, ".") || strings.Contains(path, "/.") {
					return nil
				}
				if !strings.HasSuffix(path, ".sql") {
					return nil
				}

				buf, err := fs.ReadFile(fsys, path)
				if err != nil {
					return err
				}

				// Sniff whether the file is a SQLCode file or not. We can NOT use the parser
				// for this, because the parser can be thrown off by errors, and we can't have
				// a system where files are suddenly ignored when there are syntax errors!
				// So using a more stable regex
				if isSqlCodeRegex.Find(buf) != nil {

					// protect against same file being referenced from 2 identical file systems..or just same file included twice
					pathDesc := fmt.Sprintf("fs[%d]:%s", fidx, path)
					hash := sha256.Sum256(buf)
					existingPathDesc, hashExists := hashes[hash]
					if hashExists {
						return errors.New(fmt.Sprintf("file %s has exact same contents as %s (possibly in different filesystems)",
							pathDesc, existingPathDesc))
					}
					hashes[hash] = pathDesc

					var fdoc Document
					Parse(&Scanner{input: string(buf), file: FileRef(path)}, &fdoc)

					if matchesIncludeTags(fdoc.PragmaIncludeIf, includeTags) {
						filenames = append(filenames, pathDesc)
						result.Include(fdoc)
					}
				}
				return nil
			})
		if err != nil {
			return
		}
	}

	// Do the topological sort; and include any error with it as part
	// of `result`, *not* return it as err
	sortedCreates, errpos, sortErr := TopologicalSort(result.Creates)
	if sortErr != nil {
		result.Errors = append(result.Errors, Error{
			Pos:     errpos,
			Message: sortErr.Error(),
		})
	} else {
		result.Creates = sortedCreates
	}

	return
}

func matchesIncludeTags(required []string, got []string) bool {
	for _, r := range required {
		found := false
		for _, g := range got {
			if g == r {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func IsSqlcodeConstVariable(varname string) bool {
	return strings.HasPrefix(varname, "@Enum") ||
		strings.HasPrefix(varname, "@ENUM_") ||
		strings.HasPrefix(varname, "@enum_") ||
		strings.HasPrefix(varname, "@Const") ||
		strings.HasPrefix(varname, "@CONST_") ||
		strings.HasPrefix(varname, "@const_")
}

// consider something a "sqlcode source file" if it contains [code]
// or a --sqlcode: header
var isSqlCodeRegex = regexp.MustCompile(`^--sqlcode:|\[code\]`)
