package sqldocument

// import (
// 	"testing"
// )

// // MockScanner implements TokenScanner for testing purposes.
// type MockScanner struct {
// 	tokens  []MockToken
// 	current int
// }

// // MockToken represents a token for testing.
// type MockToken struct {
// 	Type     TokenType
// 	Text     string
// 	Reserved string
// 	StartPos Pos
// 	StopPos  Pos
// }

// // NewMockScanner creates a new MockScanner with the given tokens.
// // The scanner starts positioned on the first token.
// func NewMockScanner(tokens []MockToken) *MockScanner {
// 	return &MockScanner{
// 		tokens:  tokens,
// 		current: 0,
// 	}
// }

// func (m *MockScanner) TokenType() TokenType {
// 	if m.current >= len(m.tokens) {
// 		return EOFToken
// 	}
// 	return m.tokens[m.current].Type
// }

// func (m *MockScanner) Token() string {
// 	if m.current >= len(m.tokens) {
// 		return ""
// 	}
// 	return m.tokens[m.current].Text
// }

// func (m *MockScanner) TokenLower() string {
// 	return m.Token()
// }

// func (m *MockScanner) ReservedWord() string {
// 	if m.current >= len(m.tokens) {
// 		return ""
// 	}
// 	return m.tokens[m.current].Reserved
// }

// func (m *MockScanner) Start() Pos {
// 	if m.current >= len(m.tokens) {
// 		return Pos{}
// 	}
// 	return m.tokens[m.current].StartPos
// }

// func (m *MockScanner) Stop() Pos {
// 	if m.current >= len(m.tokens) {
// 		return Pos{}
// 	}
// 	return m.tokens[m.current].StopPos
// }

// func (m *MockScanner) NextToken() TokenType {
// 	if m.current < len(m.tokens) {
// 		m.current++
// 	}
// 	return m.TokenType()
// }

// func (m *MockScanner) NextNonWhitespaceToken() TokenType {
// 	m.NextToken()
// 	m.SkipWhitespace()
// 	return m.TokenType()
// }

// func (m *MockScanner) NextNonWhitespaceCommentToken() TokenType {
// 	m.NextToken()
// 	m.SkipWhitespaceComments()
// 	return m.TokenType()
// }

// func (m *MockScanner) SkipWhitespace() {
// 	for m.TokenType() == WhitespaceToken {
// 		m.NextToken()
// 	}
// }

// func (m *MockScanner) SkipWhitespaceComments() {
// 	for {
// 		switch m.TokenType() {
// 		case WhitespaceToken, MultilineCommentToken, SinglelineCommentToken:
// 			m.NextToken()
// 		default:
// 			return
// 		}
// 	}
// }

// func (m *MockScanner) Clone() *Scanner {
// 	// For testing, we return nil as Clone returns concrete *Scanner
// 	// In real tests, you may need to handle this differently
// 	return nil
// }

// // Ensure MockScanner implements TokenScanner
// var _ TokenScanner = (*MockScanner)(nil)

// // Helper to create a real Scanner for simple test cases
// func newTestScanner(input string) *Scanner {
// 	s := NewScanner("test.sql", input)
// 	s.NextToken()
// 	return s
// }

// func TestBatch_Parse_EOF(t *testing.T) {
// 	s := newTestScanner("")
// 	b := &Batch{}

// 	result := b.Parse(s)

// 	if result != false {
// 		t.Errorf("expected false on EOF, got true")
// 	}
// }

// func TestBatch_Parse_Whitespace(t *testing.T) {
// 	s := newTestScanner("   ")
// 	b := &Batch{}

// 	result := b.Parse(s)

// 	if result != false {
// 		t.Errorf("expected false, got true")
// 	}
// 	if len(b.Nodes) != 1 {
// 		t.Errorf("expected 1 node, got %d", len(b.Nodes))
// 	}
// 	if b.DocString != nil {
// 		t.Errorf("expected DocString to be nil after whitespace")
// 	}
// }

// func TestBatch_Parse_MultilineComment(t *testing.T) {
// 	s := newTestScanner("/* comment */")
// 	b := &Batch{}

// 	result := b.Parse(s)

// 	if result != false {
// 		t.Errorf("expected false, got true")
// 	}
// 	if len(b.Nodes) != 1 {
// 		t.Errorf("expected 1 node, got %d", len(b.Nodes))
// 	}
// }

// func TestBatch_Parse_SinglelineComment_BuildsDocString(t *testing.T) {
// 	s := newTestScanner("-- comment 1\n-- comment 2")
// 	b := &Batch{}

// 	b.Parse(s)

// 	if len(b.DocString) != 2 {
// 		t.Errorf("expected 2 docstring entries, got %d", len(b.DocString))
// 	}
// }

// func TestBatch_Parse_DocStringResetOnNonComment(t *testing.T) {
// 	s := newTestScanner("-- comment\n   ")
// 	b := &Batch{}

// 	b.Parse(s)

// 	// After "-- comment\n" followed by spaces (not just \n), docstring should reset
// 	if b.DocString != nil {
// 		t.Errorf("expected DocString to be nil after non-newline whitespace, got %v", b.DocString)
// 	}
// }

// func TestBatch_Parse_DocStringPreservedOnSingleNewline(t *testing.T) {
// 	s := newTestScanner("-- comment\n")
// 	b := &Batch{}

// 	b.Parse(s)

// 	if len(b.DocString) != 1 {
// 		t.Errorf("expected 1 docstring entry after single newline, got %d", len(b.DocString))
// 	}
// }

// func TestBatch_Parse_ReservedWord_WithHandler(t *testing.T) {
// 	handlerCalled := false
// 	s := newTestScanner("SELECT")
// 	b := &Batch{
// 		TokenHandlers: map[string]func(*Scanner, *Batch) bool{
// 			"select": func(s *Scanner, b *Batch) bool {
// 				handlerCalled = true
// 				return true
// 			},
// 		},
// 	}

// 	result := b.Parse(s)

// 	if !handlerCalled {
// 		t.Errorf("expected handler to be called")
// 	}
// 	if result != true {
// 		t.Errorf("expected true when handler returns true, got false")
// 	}
// }

// func TestBatch_Parse_ReservedWord_HandlerReturnsFalse(t *testing.T) {
// 	s := newTestScanner("SELECT")
// 	b := &Batch{
// 		TokenHandlers: map[string]func(*Scanner, *Batch) bool{
// 			"select": func(s *Scanner, b *Batch) bool {
// 				s.NextToken() // consume to reach EOF
// 				return false
// 			},
// 		},
// 	}

// 	result := b.Parse(s)

// 	if result != false {
// 		t.Errorf("expected false when handler returns false and hits EOF")
// 	}
// }

// func TestBatch_Parse_ReservedWord_NoHandler(t *testing.T) {
// 	s := newTestScanner("SELECT")
// 	b := &Batch{
// 		TokenHandlers: map[string]func(*Scanner, *Batch) bool{},
// 	}

// 	b.Parse(s)

// 	if len(b.Errors) != 1 {
// 		t.Errorf("expected 1 error, got %d", len(b.Errors))
// 	}
// }

// func TestBatch_Parse_BatchSeparator(t *testing.T) {
// 	s := newTestScanner("\ngo\nSELECT")
// 	b := &Batch{}

// 	result := b.Parse(s)

// 	if result != true {
// 		t.Errorf("expected true after batch separator, got false")
// 	}
// }

// func TestBatch_Parse_BatchSeparator_WithMalformed(t *testing.T) {
// 	s := newTestScanner("\ngo 123\nSELECT")
// 	b := &Batch{}

// 	result := b.Parse(s)

// 	if result != true {
// 		t.Errorf("expected true after batch separator")
// 	}
// 	if len(b.Errors) != 1 {
// 		t.Errorf("expected 1 error for malformed batch separator, got %d", len(b.Errors))
// 	}
// }

// func TestBatch_Parse_UnexpectedToken(t *testing.T) {
// 	s := newTestScanner("123")
// 	b := &Batch{}

// 	b.Parse(s)

// 	if len(b.Errors) != 1 {
// 		t.Errorf("expected 1 error, got %d", len(b.Errors))
// 	}
// 	if b.DocString != nil {
// 		t.Errorf("expected DocString to be nil after unexpected token")
// 	}
// }

// func TestBatch_HasErrors(t *testing.T) {
// 	b := &Batch{}
// 	if b.HasErrors() {
// 		t.Errorf("expected no errors initially")
// 	}

// 	b.Errors = append(b.Errors, Error{Pos: Pos{}, Message: "test"})
// 	if !b.HasErrors() {
// 		t.Errorf("expected HasErrors to return true")
// 	}
// }
