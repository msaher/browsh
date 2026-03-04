package shell

import (
	"testing"
)

// helpers

func tok(tt TokenType, content string) Token {
	return Token{Type: tt, Content: content, Line: 1}
}

func mustParse(t *testing.T, tokens []Token) *Node {
	t.Helper()
	tokens = append(tokens, tok(TokenEOF, ""))
	node, err := NewParser(tokens).Parse()
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	return node
}

func mustFail(t *testing.T, tokens []Token) error {
	t.Helper()
	tokens = append(tokens, tok(TokenEOF, ""))
	_, err := NewParser(tokens).Parse()
	if err == nil {
		t.Fatal("expected parse error, got nil")
	}
	return err
}

// assertNode checks a node's token type and number of kids.
func assertNode(t *testing.T, n *Node, tt TokenType, nKids int) {
	t.Helper()
	if n == nil {
		t.Fatal("node is nil")
	}
	if n.Token.Type != tt {
		t.Errorf("token type: want %d, got %d", tt, n.Token.Type)
	}
	if len(n.Kids) != nKids {
		t.Errorf("kids: want %d, got %d", nKids, len(n.Kids))
	}
}

// --- simple cmd ---

func TestParseSimpleWord(t *testing.T) {
	// echo
	node := mustParse(t, []Token{tok(TokenWord, "echo")})
	// root is a cmd node (zero token type)
	if len(node.Kids) != 1 {
		t.Fatalf("want 1 kid, got %d", len(node.Kids))
	}
	assertNode(t, node.Kids[0], TokenWord, 0)
}

func TestParseCmdMultipleArgs(t *testing.T) {
	// echo hello world
	node := mustParse(t, []Token{
		tok(TokenWord, "echo"),
		tok(TokenWord, "hello"),
		tok(TokenWord, "world"),
	})
	if len(node.Kids) != 3 {
		t.Fatalf("want 3 kids, got %d", len(node.Kids))
	}
}

func TestParseCmdStringArg(t *testing.T) {
	// echo "hello world"
	node := mustParse(t, []Token{
		tok(TokenWord, "echo"),
		tok(TokenString, "hello world"),
	})
	if len(node.Kids) != 2 {
		t.Fatalf("want 2 kids, got %d", len(node.Kids))
	}
	assertNode(t, node.Kids[1], TokenString, 0)
}

// --- redirects ---

func TestParseRedirectOut(t *testing.T) {
	// echo foo > out.txt
	node := mustParse(t, []Token{
		tok(TokenWord, "echo"),
		tok(TokenWord, "foo"),
		tok(TokenOut, ">"),
		tok(TokenWord, "out.txt"),
	})
	// cmd has 3 kids: echo, foo, redirect
	if len(node.Kids) != 3 {
		t.Fatalf("want 3 kids, got %d", len(node.Kids))
	}
	redir := node.Kids[2]
	assertNode(t, redir, TokenOut, 1)
	assertNode(t, redir.Kids[0], TokenWord, 0)
}

func TestParseRedirectOutWithFd(t *testing.T) {
	// echo foo 2> err.txt
	node := mustParse(t, []Token{
		tok(TokenWord, "echo"),
		tok(TokenWord, "foo"),
		tok(TokenFd, "2"),
		tok(TokenOut, ">"),
		tok(TokenWord, "err.txt"),
	})
	redir := node.Kids[2]
	assertNode(t, redir, TokenOut, 2) // fd + target
	assertNode(t, redir.Kids[0], TokenFd, 0)
	assertNode(t, redir.Kids[1], TokenWord, 0)
}

func TestParseRedirectDupOut(t *testing.T) {
	// echo foo 2>& 1
	node := mustParse(t, []Token{
		tok(TokenWord, "echo"),
		tok(TokenFd, "2"),
		tok(TokenDupOut, ">&"),
		tok(TokenFd, "1"),
	})
	redir := node.Kids[1]
	assertNode(t, redir, TokenDupOut, 2) // fd + target fd
	assertNode(t, redir.Kids[0], TokenFd, 0)
	assertNode(t, redir.Kids[1], TokenFd, 0)
}

func TestParseRedirectDupOutNoFd(t *testing.T) {
	// echo >& 1
	node := mustParse(t, []Token{
		tok(TokenWord, "echo"),
		tok(TokenDupOut, ">&"),
		tok(TokenFd, "1"),
	})
	redir := node.Kids[1]
	assertNode(t, redir, TokenDupOut, 1) // just target fd, no leading fd
}

func TestParseRedirectIn(t *testing.T) {
	// cat < input.txt
	node := mustParse(t, []Token{
		tok(TokenWord, "cat"),
		tok(TokenIn, "<"),
		tok(TokenWord, "input.txt"),
	})
	redir := node.Kids[1]
	assertNode(t, redir, TokenIn, 1)
	assertNode(t, redir.Kids[0], TokenWord, 0)
}

func TestParseRedirectAppend(t *testing.T) {
	// echo foo >> log.txt
	node := mustParse(t, []Token{
		tok(TokenWord, "echo"),
		tok(TokenWord, "foo"),
		tok(TokenAppend, ">>"),
		tok(TokenWord, "log.txt"),
	})
	redir := node.Kids[2]
	assertNode(t, redir, TokenAppend, 1)
}

func TestParseRedirectOutToFd(t *testing.T) {
	// echo > 1  (redirect to fd as target)
	node := mustParse(t, []Token{
		tok(TokenWord, "echo"),
		tok(TokenOut, ">"),
		tok(TokenFd, "1"),
	})
	redir := node.Kids[1]
	assertNode(t, redir, TokenOut, 1)
	assertNode(t, redir.Kids[0], TokenFd, 0)
}

func TestParseLeadingRedirect(t *testing.T) {
	// < input.txt cat
	node := mustParse(t, []Token{
		tok(TokenIn, "<"),
		tok(TokenWord, "input.txt"),
		tok(TokenWord, "cat"),
	})
	// cmd: [in-redirect, cat]
	if len(node.Kids) != 2 {
		t.Fatalf("want 2 kids, got %d", len(node.Kids))
	}
	assertNode(t, node.Kids[0], TokenIn, 1)
	assertNode(t, node.Kids[1], TokenWord, 0)
}

func TestParseMultipleRedirects(t *testing.T) {
	// cmd < in.txt > out.txt
	node := mustParse(t, []Token{
		tok(TokenWord, "cmd"),
		tok(TokenIn, "<"),
		tok(TokenWord, "in.txt"),
		tok(TokenOut, ">"),
		tok(TokenWord, "out.txt"),
	})
	if len(node.Kids) != 3 {
		t.Fatalf("want 3 kids, got %d", len(node.Kids))
	}
}

// --- andif / orif ---

func TestParseAndIf(t *testing.T) {
	// foo && bar
	node := mustParse(t, []Token{
		tok(TokenWord, "foo"),
		tok(TokenAndIf, "&&"),
		tok(TokenWord, "bar"),
	})
	assertNode(t, node, TokenAndIf, 2)
	assertNode(t, node.Kids[0], 0, 1) // cmd node (zero type)
	assertNode(t, node.Kids[1], 0, 1)
}

func TestParseOrIf(t *testing.T) {
	// foo || bar
	node := mustParse(t, []Token{
		tok(TokenWord, "foo"),
		tok(TokenOrIf, "||"),
		tok(TokenWord, "bar"),
	})
	assertNode(t, node, TokenOrIf, 2)
}

func TestParseAndIfChain(t *testing.T) {
	node := mustParse(t, []Token{
		tok(TokenWord, "a"),
		tok(TokenAndIf, "&&"),
		tok(TokenWord, "b"),
		tok(TokenAndIf, "&&"),
		tok(TokenWord, "c"),
	})
	assertNode(t, node, TokenAndIf, 3)
	assertNode(t, node.Kids[0], 0, 1)
	assertNode(t, node.Kids[1], 0, 1)
	assertNode(t, node.Kids[2], 0, 1)
}

func TestParseOrIfOverAndIf(t *testing.T) {
	node := mustParse(t, []Token{
		tok(TokenWord, "a"),
		tok(TokenAndIf, "&&"),
		tok(TokenWord, "b"),
		tok(TokenOrIf, "||"),
		tok(TokenWord, "c"),
	})
	assertNode(t, node, TokenOrIf, 2)
	assertNode(t, node.Kids[0], TokenAndIf, 2)
	assertNode(t, node.Kids[1], 0, 1)
}

func TestParseOrIfChain(t *testing.T) {
	node := mustParse(t, []Token{
		tok(TokenWord, "a"),
		tok(TokenOrIf, "||"),
		tok(TokenWord, "b"),
		tok(TokenOrIf, "||"),
		tok(TokenWord, "c"),
	})
	assertNode(t, node, TokenOrIf, 3)
}

func TestParsePipeline(t *testing.T) {
	node := mustParse(t, []Token{
		tok(TokenWord, "a"),
		tok(TokenPipe, "|"),
		tok(TokenWord, "b"),
		tok(TokenPipe, "|"),
		tok(TokenWord, "c"),
	})
	assertNode(t, node, TokenPipe, 3)
	for _, kid := range node.Kids {
		assertNode(t, kid, 0, 1)
	}
}

func TestParsePipelineTwo(t *testing.T) {
	// cat foo.txt | grep bar
	node := mustParse(t, []Token{
		tok(TokenWord, "cat"),
		tok(TokenWord, "foo.txt"),
		tok(TokenPipe, "|"),
		tok(TokenWord, "grep"),
		tok(TokenWord, "bar"),
	})
	assertNode(t, node, TokenPipe, 2)
	assertNode(t, node.Kids[0], 0, 2) // cat + foo.txt
	assertNode(t, node.Kids[1], 0, 2) // grep + bar
}

// --- errors ---

func TestParseErrorNoCommand(t *testing.T) {
	// just a redirect with no command word
	mustFail(t, []Token{
		tok(TokenOut, ">"),
		tok(TokenWord, "out.txt"),
	})
}

func TestParseErrorFdBeforeIn(t *testing.T) {
	mustFail(t, []Token{
		tok(TokenWord, "cmd"),
		tok(TokenFd, "2"),
		tok(TokenIn, "<"),
		tok(TokenWord, "input.txt"),
	})
}

func TestParseErrorFdBeforeAppend(t *testing.T) {
	mustFail(t, []Token{
		tok(TokenWord, "cmd"),
		tok(TokenFd, "2"),
		tok(TokenAppend, ">>"),
		tok(TokenWord, "log.txt"),
	})
}

func TestParseErrorFdNotFollowedByRedirect(t *testing.T) {
	mustFail(t, []Token{
		tok(TokenWord, "cmd"),
		tok(TokenFd, "2"),
		tok(TokenWord, "foo"),
	})
}

func TestParseErrorMissingRedirectTarget(t *testing.T) {
	mustFail(t, []Token{
		tok(TokenWord, "echo"),
		tok(TokenOut, ">"),
	})
}

func TestParseErrorDupOutNeedsFd(t *testing.T) {
	mustFail(t, []Token{
		tok(TokenWord, "echo"),
		tok(TokenDupOut, ">&"),
		tok(TokenWord, "notanfd"),
	})
}

func TestParseErrorTrailingJunk(t *testing.T) {
	mustFail(t, []Token{
		tok(TokenWord, "echo"),
		tok(TokenOrIf, "||"),
	})
}
