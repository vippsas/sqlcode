package sqlparser

const (
	WhitespaceToken TokenType = iota + 1

	LeftParenToken
	RightParenToken
	SemicolonToken
	EqualToken
	CommaToken
	DotToken

	VarcharLiteralToken
	NVarcharLiteralToken

	MultilineCommentToken
	SinglelineCommentToken

	// PragmaToken is like SinglelineCommentToken but starting with `--sqlcode:`.
	// It is useful to scan this as a separate token type because then this comment
	// anywhere else than the top of the file will not be treated as whitespace,
	// but give an error.
	PragmaToken

	NumberToken

	// Note: A lot of stuff pass as identifiers that should really have been
	// reserved words
	ReservedWordToken
	VariableIdentifierToken
	QuotedIdentifierToken
	UnquotedIdentifierToken
	OtherToken

	UnterminatedVarcharLiteralErrorToken
	UnterminatedQuotedIdentifierErrorToken
	DoubleQuoteErrorToken // we don't want to support double quotes, for simplicity, so that is an error and stops parsing
	UnexpectedCharacterToken
	NonUTF8ErrorToken

	BatchSeparatorToken
	MalformedBatchSeparatorToken
	EOFToken
)

func (tt TokenType) GoString() string {
	return tokenToDescription[tt]
}

func (tt TokenType) String() string {
	return tokenToDescription[tt]
}

func init() {
	// make sure we panic if a description isn't declared
	for tt := TokenType(1); tt != EOFToken; tt++ {
		if tokenToDescription[tt] == "" {
			panic("you have not updated tokenToDescription")
		}
	}
}

var tokenToDescription = map[TokenType]string{
	WhitespaceToken: "WhitespaceToken",
	LeftParenToken:  "LeftParenToken",
	RightParenToken: "RightParenToken",
	SemicolonToken:  "SemicolonToken",
	EqualToken:      "EqualToken",
	CommaToken:      "CommaToken",
	DotToken:        "DotToken",

	VarcharLiteralToken:  "VarcharLiteralToken",
	NVarcharLiteralToken: "NVarcharLiteralToken",

	MultilineCommentToken:  "MultilineCommentToken",
	SinglelineCommentToken: "SinglelineCommentToken",
	PragmaToken:            "PragmaToken",

	NumberToken: "NumberToken",

	ReservedWordToken:       "ReservedWordToken",
	VariableIdentifierToken: "VariableIdentifierToken",
	QuotedIdentifierToken:   "QuotedIdentifierToken",
	UnquotedIdentifierToken: "UnquotedIdentifierToken",
	OtherToken:              "OtherToken",

	UnterminatedVarcharLiteralErrorToken:   "UnterminatedVarcharLiteralErrorToken",
	UnterminatedQuotedIdentifierErrorToken: "UnterminatedQuotedIdentifierErrorToken",
	DoubleQuoteErrorToken:                  "DoubleQuoteErrorToken",
	UnexpectedCharacterToken:               "UnexpectedCharacterToken",
	NonUTF8ErrorToken:                      "NonUTF8ErrorToken",

	// After a lot of back and forth we added the batch separater to the scanner.
	// We implement sqlcmd's use of the go
	// do separate batches. sqlcmd will only support GO at the start of
	// the line, but it is case-insensitive and skips whitespace...
	// For instance:
	//
	// select 1 as
	//    GO
	// select 2
	//
	// This will create 2 batches and give a syntax error for the first select
	// as the target column is missing -- unlike "select 1 as GO select 2" which
	// is valid (2 queries, 1st one has a column GO, 2nd has anonymous column).
	//
	// BUT: A GO inside [], "" or '' is NEVER a batch separator.
	//
	// PS: Having anything else on the same line as the go token is an error too,
	// but that has to be handled by the parser.
	BatchSeparatorToken:          "BatchSeparatorToken",
	MalformedBatchSeparatorToken: "MalformedBatchSeparatorToken",
	EOFToken:                     "EOFToken",
}
