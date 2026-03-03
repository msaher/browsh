package shell

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// run parses and executes a token list, returning the interpreter and any error.
// stdout and stderr are redirected to temp files so tests can inspect them.
func run(t *testing.T, tokens []Token) (stdout, stderr string, err error) {
	t.Helper()

	dir := t.TempDir()
	outFile := filepath.Join(dir, "stdout")
	errFile := filepath.Join(dir, "stderr")

	outF, e := os.Create(outFile)
	if e != nil {
		t.Fatal(e)
	}
	errF, e := os.Create(errFile)
	if e != nil {
		t.Fatal(e)
	}

	inter := NewInterpreter(dir)
	inter.Stdout = outF
	inter.Stderr = errF

	tokens = append(tokens, tok(TokenEOF, ""))
	node, parseErr := NewParser(tokens).Parse()
	if parseErr != nil {
		t.Fatalf("parse error: %v", parseErr)
	}

	err = inter.Exec(node)

	outF.Close()
	errF.Close()

	outBytes, _ := os.ReadFile(outFile)
	errBytes, _ := os.ReadFile(errFile)
	return string(outBytes), string(errBytes), err
}

// runInDir is like run but lets the caller populate the temp dir first.
func runInDir(t *testing.T, dir string, tokens []Token) (stdout, stderr string, err error) {
	t.Helper()

	outFile := filepath.Join(dir, "_stdout")
	errFile := filepath.Join(dir, "_stderr")

	outF, e := os.Create(outFile)
	if e != nil {
		t.Fatal(e)
	}
	errF, e := os.Create(errFile)
	if e != nil {
		t.Fatal(e)
	}

	inter := NewInterpreter(dir)
	inter.Stdout = outF
	inter.Stderr = errF

	tokens = append(tokens, tok(TokenEOF, ""))
	node, parseErr := NewParser(tokens).Parse()
	if parseErr != nil {
		t.Fatalf("parse error: %v", parseErr)
	}

	err = inter.Exec(node)

	outF.Close()
	errF.Close()

	outBytes, _ := os.ReadFile(outFile)
	errBytes, _ := os.ReadFile(errFile)
	return string(outBytes), string(errBytes), err
}

// --- simple commands ---

func TestExecEcho(t *testing.T) {
	stdout, _, err := run(t, []Token{
		tok(TokenWord, "echo"),
		tok(TokenWord, "hello"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "hello") {
		t.Errorf("want 'hello' in stdout, got %q", stdout)
	}
}

func TestExecStringArg(t *testing.T) {
	stdout, _, err := run(t, []Token{
		tok(TokenWord, "echo"),
		tok(TokenString, "hello world"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "hello world") {
		t.Errorf("want 'hello world' in stdout, got %q", stdout)
	}
}

func TestExecCommandNotFound(t *testing.T) {
	_, _, err := run(t, []Token{
		tok(TokenWord, "totallymadeupcommand"),
	})
	if err == nil {
		t.Fatal("expected error for unknown command, got nil")
	}
}

func TestExecNonZeroExit(t *testing.T) {
	_, _, err := run(t, []Token{
		tok(TokenWord, "false"),
	})
	if err == nil {
		t.Fatal("expected error from false, got nil")
	}
}

// --- redirect out ---

func TestRedirectOutCreatesFile(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "out.txt")

	_, _, err := runInDir(t, dir, []Token{
		tok(TokenWord, "echo"),
		tok(TokenWord, "hello"),
		tok(TokenOut, ">"),
		tok(TokenWord, "out.txt"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, readErr := os.ReadFile(target)
	if readErr != nil {
		t.Fatalf("output file not created: %v", readErr)
	}
	if !strings.Contains(string(data), "hello") {
		t.Errorf("want 'hello' in file, got %q", string(data))
	}
}

func TestRedirectOutTruncates(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "out.txt")
	os.WriteFile(target, []byte("old content\n"), 0644)

	runInDir(t, dir, []Token{
		tok(TokenWord, "echo"),
		tok(TokenWord, "new"),
		tok(TokenOut, ">"),
		tok(TokenWord, "out.txt"),
	})

	data, _ := os.ReadFile(target)
	if strings.Contains(string(data), "old") {
		t.Errorf("file should be truncated, got %q", string(data))
	}
}

func TestRedirectOutStderr(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "err.txt")

	// use ls on a nonexistent path to generate stderr output
	runInDir(t, dir, []Token{
		tok(TokenWord, "ls"),
		tok(TokenWord, "/nonexistent_path_xyz"),
		tok(TokenFd, "2"),
		tok(TokenOut, ">"),
		tok(TokenWord, "err.txt"),
	})

	data, readErr := os.ReadFile(target)
	if readErr != nil {
		t.Fatalf("stderr redirect file not created: %v", readErr)
	}
	if len(data) == 0 {
		t.Error("expected stderr output in file, got empty")
	}
}

// --- redirect append ---

func TestRedirectAppend(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "log.txt")
	os.WriteFile(target, []byte("first\n"), 0644)

	_, _, err := runInDir(t, dir, []Token{
		tok(TokenWord, "echo"),
		tok(TokenWord, "second"),
		tok(TokenAppend, ">>"),
		tok(TokenWord, "log.txt"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, _ := os.ReadFile(target)
	if !strings.Contains(string(data), "first") || !strings.Contains(string(data), "second") {
		t.Errorf("want both lines, got %q", string(data))
	}
}

// --- redirect in ---

func TestRedirectIn(t *testing.T) {
	dir := t.TempDir()
	inFile := filepath.Join(dir, "input.txt")
	os.WriteFile(inFile, []byte("hello from file\n"), 0644)

	stdout, _, err := runInDir(t, dir, []Token{
		tok(TokenWord, "cat"),
		tok(TokenIn, "<"),
		tok(TokenWord, "input.txt"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "hello from file") {
		t.Errorf("want file content in stdout, got %q", stdout)
	}
}

func TestRedirectInMissingFile(t *testing.T) {
	_, _, err := run(t, []Token{
		tok(TokenWord, "cat"),
		tok(TokenIn, "<"),
		tok(TokenWord, "doesnotexist.txt"),
	})
	if err == nil {
		t.Fatal("expected error for missing input file, got nil")
	}
}

// --- globs ---

func TestGlobExpansion(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a"), 0644)
	os.WriteFile(filepath.Join(dir, "b.txt"), []byte("b"), 0644)

	stdout, _, err := runInDir(t, dir, []Token{
		tok(TokenWord, "ls"),
		tok(TokenWord, "*.txt"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "a.txt") || !strings.Contains(stdout, "b.txt") {
		t.Errorf("want both .txt files in output, got %q", stdout)
	}
}

func TestGlobNoMatch(t *testing.T) {
	// no match: literal word should be passed through
	stdout, _, _ := run(t, []Token{
		tok(TokenWord, "echo"),
		tok(TokenWord, "*.xyz"),
	})
	if !strings.Contains(stdout, "*.xyz") {
		t.Errorf("want literal '*.xyz' passed through, got %q", stdout)
	}
}

// --- ContainsGlob ---

func TestContainsGlob(t *testing.T) {
	cases := []struct {
		s    string
		want bool
	}{
		{"*.txt", true},
		{"file.txt", false},
		{"foo?bar", true},
		{"[abc]", true},
		{"normal", false},
		{"", false},
	}
	for _, c := range cases {
		if got := ContainsGlob(c.s); got != c.want {
			t.Errorf("ContainsGlob(%q) = %v, want %v", c.s, got, c.want)
		}
	}
}
