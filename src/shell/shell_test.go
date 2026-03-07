package shell

import (
	"testing"
	"fmt"
)

func tokenTypes(tokens []Token) []TokenType {
	types := make([]TokenType, len(tokens))
	for i, t := range tokens {
		types[i] = t.Type
	}
	return types
}

func TestScanBasicTokens(t *testing.T) {
	src := `echo "hello" > out.txt && cat < in.txt | grep "pattern"`
	expected := []TokenType{
		TokenWord,   // echo
		TokenString, // "hello"
		TokenOut,    // >
		TokenWord,   // out.txt
		TokenAndIf,  // &&
		TokenWord,   // cat
		TokenIn,     // <
		TokenWord,   // in.txt
		TokenPipe,   // |
		TokenWord,   // grep
		TokenString, // "pattern"
		TokenEOF,
	}

	tokens, err := Scan(src)
	if err != nil {
		for _, tok := range tokens {
			fmt.Printf("%s ", tok.Type)
		}
		t.Fatalf("unexpected error: %v", err)
	}

	got := tokenTypes(tokens)
	for i, tok := range got {
		if tok != expected[i] {
			t.Errorf("token %d: got %v, want %v", i, tok, expected[i])
		}
	}
}

func TestScanWhitespaceAndComments(t *testing.T) {
	src := `
# comment line
ls   -l   # another comment
echo hi
`
	expected := []TokenType{
		TokenWord, // ls
		TokenWord, // -l
		TokenWord, // echo
		TokenWord, // hi
		TokenEOF,
	}

	tokens, err := Scan(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := tokenTypes(tokens)
	if len(got) != len(expected) {
		t.Fatalf("wrong number of tokens: got %d, want %d", len(got), len(expected))
	}

	for i, tok := range got {
		if tok != expected[i] {
			t.Errorf("token %d: got %v, want %v", i, tok, expected[i])
		}
	}
}

func TestScanOperators(t *testing.T) {
	src := `a && b || c > out < in >> append >> append 1>&2`
	expected := []TokenType{
		TokenWord,    // a
		TokenAndIf,   // &&
		TokenWord,    // b
		TokenOrIf,    // ||
		TokenWord,    // c
		TokenOut,     // >
		TokenWord,    // out
		TokenIn,      // <
		TokenWord,    // in
		TokenAppend,  // >>
		TokenWord,    // append
		TokenAppend,  // >>
		TokenWord,    // append2
		TokenFd,      // 1
		TokenDupOut,  // >&
		TokenFd,      // 2
		TokenEOF,
	}

	tokens, err := Scan(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := tokenTypes(tokens)
	if len(got) != len(expected) {
		t.Fatalf("wrong number of tokens: got %d, want %d", len(got), len(expected))
	}

	for i, tok := range got {
		if tok != expected[i] {
			t.Errorf("token %d: got %v, want %v (content=%q)", i, tok, expected[i], tokens[i].Content)
		}
	}
}

func TestScanUnterminatedString(t *testing.T) {
	src := `echo "hello`
	tokens, err := Scan(src)

	if err == nil {
		t.Fatalf("expected error for unterminated string, got nil")
	}

	if tokens[len(tokens)-1].Type != TokenError {
		t.Fatalf("expected last token to be TokenError, got %v", tokens[len(tokens)-1].Type)
	}

	expectedMsg := "unterminated string"
	if tokens[len(tokens)-1].Content != expectedMsg {
		t.Errorf("error message: got %q, want %q", tokens[len(tokens)-1].Content, expectedMsg)
	}
}

func TestScanEmptyInput(t *testing.T) {
	src := `   `
	tokens, err := Scan(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tokens) != 1 || tokens[0].Type != TokenEOF {
		t.Fatalf("expected single EOF token, got %v", tokens)
	}
}

func TestScanComplexMix(t *testing.T) {
	src := `
echo "hi" # comment
cat < input.txt | grep foo && echo done >> out.txt
`
	expected := []TokenType{
		TokenWord,   // echo
		TokenString, // "hi"
		TokenWord,   // cat
		TokenIn,     // <
		TokenWord,   // input.txt
		TokenPipe,   // |
		TokenWord,   // grep
		TokenWord,   // foo
		TokenAndIf,  // &&
		TokenWord,   // echo
		TokenWord,   // done
		TokenAppend, // >>
		TokenWord,   // out.txt
		TokenEOF,
	}

	tokens, err := Scan(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := tokenTypes(tokens)
	if len(got) != len(expected) {
		t.Fatalf("wrong number of tokens: got %d, want %d", len(got), len(expected))
	}

	for i, tok := range got {
		if tok != expected[i] {
			t.Errorf("token %d: got %v, want %v", i, tok, expected[i])
		}
	}
}

func TestScanLineNumbers(t *testing.T) {
	src := `echo a
# comment
ls
`
	tokens, err := Scan(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lines := []int{1, 1, 3, 4, 0}
	for i, tok := range tokens {
		if i >= len(lines) {
			break
		}
		if tok.Line != lines[i] {
			t.Errorf("token %v line: got %d, want %d", tok, tok.Line, lines[i])
		}
	}
}

func TestScanBlock(t *testing.T) {

	src := `:lua {
		print("hello world")
	}`
	expected := []TokenType{
		TokenWord, // :py
		TokenBlock,
		TokenEOF,
	}

	tokens, err := Scan(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := tokenTypes(tokens)
	if len(got) != len(expected) {
		t.Fatalf("wrong number of tokens: got %d, want %d", len(got), len(expected))
	}

	for i, tok := range got {
		if tok != expected[i] {
			t.Errorf("token %d: got %v, want %v", i, tok, expected[i])
		}
	}


}
