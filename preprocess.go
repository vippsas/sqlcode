package sqlcode

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/vippsas/sqlcode/sqlparser"
	"regexp"
	"strings"
)

func SchemaSuffixFromHash(doc sqlparser.Document) string {
	hasher := sha256.New()
	for _, dec := range doc.Declares {
		hasher.Write([]byte(dec.String() + "\n"))
	}
	for _, c := range doc.Creates {
		if err := c.SerializeBytes(hasher); err != nil {
			panic(err) // asserting that sha256 will never return a write error...
		}
	}
	// As we deploy schemas there will also be a cleanup job
	// that deletes older schemas that will then no longer be relevant
	// for collisions. So, assume that we have 100 concurrent schemas;
	// and 6 bytes of hash = 48 bits, odds of collision is:
	// In [8]: 1 - np.exp((-100.**2/(2.**(8*6))))
	// Out[8]: 3.552713678800501e-11
	//
	// The experiment is repeated a lot of times, but we should be good...

	return hex.EncodeToString(hasher.Sum(nil)[:6])
}

func SchemaName(suffix string) string {
	return "code@" + suffix
}

type lineNumberCorrection struct {
	inputLineNumber, extraLinesInOutput int
}

type Batch struct {
	StartPos sqlparser.Pos
	Lines    string

	// lineNumberCorrections contains data that helps us map from errors in the `Lines`
	// SQL result and back to the original source file (pointed at by StartPos).
	// See comments in RelativeLineNumberInInput()
	lineNumberCorrections []lineNumberCorrection
}

// LineNumberInInput transforms an error line number when executing the batch,
// into an absolute line number in StartPos.File
func (b Batch) LineNumberInInput(outputline int) int {
	return b.RelativeLineNumberInInput(outputline) + b.StartPos.Line - 1
}

// RelativeLineNumberInInput maps a line number from the output of preprocessing
// (`outputline)` to a line number in the input of the pre-processing.
// PS: StartPos must be considered *in addition* to this transform.
func (b Batch) RelativeLineNumberInInput(outputline int) int {
	// See testcase for example
	// lineNumberCorrections has the number of extra lines each input line
	// ended up consuming in the output. Most lines are not mentioned; these
	// are the one that mapped 1:1, no extra lines.

	totalExtraLines := 0
	for _, c := range b.lineNumberCorrections {
		// refer to `c` as a "checkpoint" ... it's a point in the line number series
		checkpointLineNumberInOutput := c.inputLineNumber + totalExtraLines
		distanceToCheckpoint := outputline - checkpointLineNumberInOutput
		if distanceToCheckpoint < c.extraLinesInOutput {
			if distanceToCheckpoint < 0 {
				distanceToCheckpoint = 0
			}
			return outputline - totalExtraLines - distanceToCheckpoint
		}
		totalExtraLines += c.extraLinesInOutput
	}
	return outputline - totalExtraLines
}

type PreprocessedFile struct {
	Batches []Batch
}

type PreprocessorError struct {
	Pos     sqlparser.Pos
	Message string
}

func (p PreprocessorError) Error() string {
	return fmt.Sprintf("%s:%d:%d: %s", p.Pos.File, p.Pos.Line, p.Pos.Col, p.Message)
}

var codeSchemaRegexp = regexp.MustCompile(`(?i)\[code\]`)

func sqlcodeTransformCreate(declares map[string]string, c sqlparser.Create, quotedTargetSchema string) (result Batch, err error) {
	var w strings.Builder

	if len(c.Body) > 0 {
		result.StartPos = c.Body[0].Start
	}

	// Since the parser doesn't understand much
	// this is currently very simple to do, just transform a
	// a flat list of sqlparser.Unparsed and do replacements for [code] and @Enum.

	// A @Enum replacement can lead to line numbers changing due to \n present in the literal.
	// For this reason we need to make a mapping between source line numbers and result
	// line numbers
	for _, u := range c.Body {
		token := u.RawValue
		switch {
		case u.Type == sqlparser.QuotedIdentifierToken && u.RawValue == "[code]":
			token = quotedTargetSchema
		case u.Type == sqlparser.VariableIdentifierToken && sqlparser.IsSqlcodeConstVariable(u.RawValue):
			constLiteral, ok := declares[u.RawValue]
			if !ok {
				err = PreprocessorError{u.Start, fmt.Sprintf("sqlcode constant `%s` not declared", u.RawValue)}
				return
			}
			token = constLiteral + "/*=" + u.RawValue + "*/"
			newlineCount := strings.Count(constLiteral, "\n")
			if newlineCount > 0 {
				relativeLine := u.Start.Line - result.StartPos.Line
				result.lineNumberCorrections = append(result.lineNumberCorrections, lineNumberCorrection{relativeLine, newlineCount})
			}
		}

		if _, err = w.WriteString(token); err != nil {
			return
		}
	}

	result.Lines = w.String()
	return
}

func Preprocess(doc sqlparser.Document, schemasuffix string) (PreprocessedFile, error) {
	var result PreprocessedFile

	if strings.Contains(schemasuffix, "]") {
		return result, errors.New("schemasuffix cannot contain the escape character ]")
	}

	declares := make(map[string]string)
	for _, dec := range doc.Declares {
		declares[dec.VariableName] = dec.Literal.RawValue
	}

	for _, create := range doc.Creates {
		if len(create.Body) == 0 {
			continue
		}
		batch, err := sqlcodeTransformCreate(declares, create, "[code@"+schemasuffix+"]")
		if err != nil {
			return result, err
		}
		result.Batches = append(result.Batches, batch)
	}

	return result, nil
}

func preprocessString(schemasuffix string, sql string) string {
	return codeSchemaRegexp.ReplaceAllString(sql, "["+SchemaName(schemasuffix)+"]")
}
