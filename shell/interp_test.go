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
	stdio := Stdio {
		Stdout: outF,
		Stderr: errF,
	}

	tokens = append(tokens, tok(TokenEOF, ""))
	node, parseErr := NewParser(tokens).Parse()
	if parseErr != nil {
		t.Fatalf("parse error: %v", parseErr)
	}

	result := inter.Exec(node, stdio)
	err = result.Err()

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
	stdio := Stdio {
		Stdout: outF,
		Stderr: errF,
	}

	tokens = append(tokens, tok(TokenEOF, ""))
	node, parseErr := NewParser(tokens).Parse()
	if parseErr != nil {
		t.Fatalf("parse error: %v", parseErr)
	}

	result := inter.Exec(node, stdio)
	err = result.Err()

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
		tok(TokenString, `"hello world"`),
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

// runStr parses a source string and executes it, returning stdout, stderr, and any error.
func runStr(t *testing.T, dir, src string) (stdout, stderr string, err error) {
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

	tokens, scanErr := Scan(src)
	if scanErr != nil {
		t.Fatalf("scan error: %v", scanErr)
	}

	node, parseErr := NewParser(tokens).Parse()
	if parseErr != nil {
		t.Fatalf("parse error: %v", parseErr)
	}

	inter := NewInterpreter(dir)
	stdio := Stdio {
		Stdout: outF,
		Stderr: errF,
	}
	err = inter.Exec(node, stdio).Err()

	outF.Close()
	errF.Close()

	outBytes, _ := os.ReadFile(outFile)
	errBytes, _ := os.ReadFile(errFile)
	return string(outBytes), string(errBytes), err
}

func mustRunStr(t *testing.T, dir, s string) (string, string) {
	t.Helper()
	stdout, stderr, err := runStr(t, dir, s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return stdout, stderr
}

// --- builtin: echo ---

func TestBuiltinEcho(t *testing.T) {
	dir := t.TempDir()
	stdout, _, err := runStr(t, dir, "echo hello world")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "hello world") {
		t.Errorf("want 'hello world', got %q", stdout)
	}
}

func TestBuiltinEchoNoArgs(t *testing.T) {
	dir := t.TempDir()
	stdout, _, err := runStr(t, dir, "echo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// should output a blank line
	if stdout != "\n" {
		t.Errorf("want blank line, got %q", stdout)
	}
}

func TestBuiltinEchoRedirect(t *testing.T) {
	dir := t.TempDir()
	_, _, err := runStr(t, dir, "echo hello > out.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, _ := os.ReadFile(filepath.Join(dir, "out.txt"))
	if !strings.Contains(string(data), "hello") {
		t.Errorf("want 'hello' in file, got %q", string(data))
	}
}

// --- builtin: pwd ---

func TestBuiltinPwd(t *testing.T) {
	dir := t.TempDir()
	stdout, _, err := runStr(t, dir, "pwd")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, dir) {
		t.Errorf("want %q in pwd output, got %q", dir, stdout)
	}
}

func TestBuiltinPwdRedirect(t *testing.T) {
	dir := t.TempDir()
	_, _, err := runStr(t, dir, "pwd > here.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, _ := os.ReadFile(filepath.Join(dir, "here.txt"))
	if !strings.Contains(string(data), dir) {
		t.Errorf("want cwd in file, got %q", string(data))
	}
}

// --- builtin: cd ---

func TestBuiltinCd(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "sub")
	os.Mkdir(sub, 0755)

	inter := NewInterpreter(dir)
	stdio := Stdio {
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
	tokens, _ := Scan("cd sub")
	node, _ := NewParser(tokens).Parse()
	if err := inter.Exec(node, stdio).Err(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inter.Cwd != sub {
		t.Errorf("want cwd %q, got %q", sub, inter.Cwd)
	}
}

func TestBuiltinCdAbsolute(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "sub")
	os.Mkdir(sub, 0755)

	inter := NewInterpreter(dir)
	stdio := Stdio{Stdout: os.Stdout, Stderr: os.Stderr}
	tokens, _ := Scan("cd " + sub)
	node, _ := NewParser(tokens).Parse()
	if err := inter.Exec(node, stdio).Err(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inter.Cwd != sub {
		t.Errorf("want cwd %q, got %q", sub, inter.Cwd)
	}
}

// --- pipelines ---

func TestPipeBasic(t *testing.T) {
	dir := t.TempDir()
	stdout, _, err := runStr(t, dir, "echo hello | cat")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "hello") {
		t.Errorf("want 'hello' in stdout, got %q", stdout)
	}
}

func TestPipeThreeStages(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "f.txt"), []byte("foo\nbar\nbaz\n"), 0644)
	stdout, _, err := runStr(t, dir, "cat f.txt | grep ba | cat")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "bar") || !strings.Contains(stdout, "baz") {
		t.Errorf("want 'bar' and 'baz', got %q", stdout)
	}
	if strings.Contains(stdout, "foo") {
		t.Errorf("want 'foo' filtered out, got %q", stdout)
	}
}

func TestPipeLastExitCode(t *testing.T) {
	dir := t.TempDir()
	_, _, err := runStr(t, dir, "echo hello | false")
	if err == nil {
		t.Fatal("want error from false at end of pipe, got nil")
	}
}

func TestPipeIntoRedirect(t *testing.T) {
	dir := t.TempDir()
	_, _, err := runStr(t, dir, "echo hello | cat > out.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, _ := os.ReadFile(filepath.Join(dir, "out.txt"))
	if !strings.Contains(string(data), "hello") {
		t.Errorf("want 'hello' in out.txt, got %q", string(data))
	}
}

func TestPipeBuiltinLeft(t *testing.T) {
	dir := t.TempDir()
	stdout, _, err := runStr(t, dir, "echo hello world | cat")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "hello world") {
		t.Errorf("want 'hello world', got %q", stdout)
	}
}

func TestPipeBuiltinRight(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "sub")
	os.Mkdir(sub, 0755)
	stdout, _, err := runStr(t, dir, "echo sub | cat")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "sub") {
		t.Errorf("want 'sub' in output, got %q", stdout)
	}
}

// --- && ---

func TestAndIfBothSucceed(t *testing.T) {
	dir := t.TempDir()
	stdout, _, err := runStr(t, dir, "echo foo && echo bar")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "foo") || !strings.Contains(stdout, "bar") {
		t.Errorf("want both outputs, got %q", stdout)
	}
}

func TestAndIfShortCircuit(t *testing.T) {
	dir := t.TempDir()
	stdout, _, err := runStr(t, dir, "false && echo bar")
	if err == nil {
		t.Fatal("want error from false, got nil")
	}
	if strings.Contains(stdout, "bar") {
		t.Errorf("want second command skipped, got %q", stdout)
	}
}

func TestAndIfChainStopsOnFailure(t *testing.T) {
	dir := t.TempDir()
	stdout, _, err := runStr(t, dir, "echo a && false && echo c")
	if err == nil {
		t.Fatal("want error, got nil")
	}
	if !strings.Contains(stdout, "a") {
		t.Errorf("want first command to run, got %q", stdout)
	}
	if strings.Contains(stdout, "c") {
		t.Errorf("want third command skipped, got %q", stdout)
	}
}

func TestAndIfAllThreeSucceed(t *testing.T) {
	dir := t.TempDir()
	stdout, _, err := runStr(t, dir, "echo a && echo b && echo c")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "a") || !strings.Contains(stdout, "b") || !strings.Contains(stdout, "c") {
		t.Errorf("want all three outputs, got %q", stdout)
	}
}

// --- || ---

// func TestOrIfFirstSucceeds(t *testing.T) {
// 	dir := t.TempDir()
// 	stdout, _, err := runStr(t, dir, "echo foo || echo bar")
// 	if err != nil {
// 		t.Fatalf("unexpected error: %v", err)
// 	}
// 	if !strings.Contains(stdout, "foo") {
// 		t.Errorf("want 'foo', got %q", stdout)
// 	}
// 	if strings.Contains(stdout, "bar") {
// 		t.Errorf("want second command skipped, got %q", stdout)
// 	}
// }

// func TestOrIfFirstFails(t *testing.T) {
// 	dir := t.TempDir()
// 	stdout, _, err := runStr(t, dir, "false || echo bar")
// 	if err != nil {
// 		t.Fatalf("unexpected error: %v", err)
// 	}
// 	if !strings.Contains(stdout, "bar") {
// 		t.Errorf("want 'bar', got %q", stdout)
// 	}
// }

// func TestOrIfChainStopsOnFirstSuccess(t *testing.T) {
// 	dir := t.TempDir()
// 	stdout, _, err := runStr(t, dir, "false || echo b || echo c")
// 	if err != nil {
// 		t.Fatalf("unexpected error: %v", err)
// 	}
// 	if !strings.Contains(stdout, "b") {
// 		t.Errorf("want 'b', got %q", stdout)
// 	}
// 	if strings.Contains(stdout, "c") {
// 		t.Errorf("want third command skipped, got %q", stdout)
// 	}
// }

// func TestOrIfAllFail(t *testing.T) {
// 	dir := t.TempDir()
// 	_, _, err := runStr(t, dir, "false || false || false")
// 	if err == nil {
// 		t.Fatal("want error when all fail, got nil")
// 	}
// }

// --- && and || combined ---

// func TestAndIfThenOrIf(t *testing.T) {
// 	dir := t.TempDir()
// 	stdout, _, err := runStr(t, dir, "false && echo a || echo b")
// 	if err != nil {
// 		t.Fatalf("unexpected error: %v", err)
// 	}
// 	if strings.Contains(stdout, "a") {
// 		t.Errorf("want 'a' skipped, got %q", stdout)
// 	}
// 	if !strings.Contains(stdout, "b") {
// 		t.Errorf("want 'b', got %q", stdout)
// 	}
// }

// --- tilde expansion ---

// func TestTildeAlone(t *testing.T) {
// 	dir := t.TempDir()
// 	t.Setenv("HOME", dir)
// 	stdout, _, err := runStr(t, dir, "echo ~")
// 	if err != nil {
// 		t.Fatalf("unexpected error: %v", err)
// 	}
// 	if !strings.Contains(stdout, dir) {
// 		t.Errorf("want %q, got %q", dir, stdout)
// 	}
// }

func TestTildeSlash(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	stdout, _, err := runStr(t, dir, "echo ~/foo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := dir + "/foo"
	if !strings.Contains(stdout, want) {
		t.Errorf("want %q, got %q", want, stdout)
	}
}

// func TestTildeCd(t *testing.T) {
// 	dir := t.TempDir()
// 	t.Setenv("HOME", dir)
// 	inter := NewInterpreter("/")
// 	tokens, _ := Scan("cd ~")
// 	node, _ := NewParser(tokens).Parse()
// 	if err := inter.Exec(node, Stdio{}); err != nil {
// 		t.Fatalf("unexpected error: %v", err)
// 	}
// 	if inter.Cwd != dir {
// 		t.Errorf("want cwd %q, got %q", dir, inter.Cwd)
// 	}
// }

func TestTildeNoHome(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", "")
	stdout, _, err := runStr(t, dir, "echo ~")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(stdout, dir) {
		t.Errorf("with empty HOME, ~ should not expand to %q", dir)
	}
}

// shouldn't expand in strings
func TestTildeInString(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	stdout, _, err := runStr(t, dir, `echo "~/foo"`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "~/foo"
	if !strings.Contains(stdout, want) {
		t.Errorf("want %q, got %q", want, stdout)
	}
}

// --- blocks ---

func TestBlock(t *testing.T) {
	dir := t.TempDir()
	stdout, _ := mustRunStr(t, dir, `:lua {
		sh.print("hello")
	}`)
	t.Log(stdout)
	if !strings.Contains(stdout, "hello") {
		t.Errorf("want 'hello' in stdout, got %q", stdout)
	}
}

func TestLua(t *testing.T) {

	t.Run("cwd", func(t *testing.T) {
		dir := t.TempDir()
		stdout, _ := mustRunStr(t, dir, `:lua {
			sh.print(sh.cwd)
		}`)
		if !strings.Contains(stdout, dir) {
			t.Errorf("want %q in stdout, got %q", dir, stdout)
		}
	})

	t.Run("stdin", func(t *testing.T) {
		dir := t.TempDir()
		stdout, _ := mustRunStr(t, dir, `printf "1\n2\n3" | :lua {
			x = {}
			for line in sh.stdin do
				table.insert(x, line)
			end
			for i, v in ipairs(x) do
				sh.write(v)
			end
		}`)
		if stdout != "123" {
			t.Errorf("want 123 in stdout, got %q", stdout)
		}
	})

	t.Run("env", func(t *testing.T) {
		t.Setenv("XYZ", "123")
		stdout, _ := mustRunStr(t, t.TempDir(), `:lua {
			sh.print(sh.env.XYZ)
		}`)
		if stdout != "123\n" {
			t.Errorf("want 123 in stdout, got %q", stdout)
		}
	})

	t.Run("run", func(t *testing.T) {
		stdout, _ := mustRunStr(t, t.TempDir(), `:lua {
			out, code = sh.run("echo hi")
			sh.write(out)
		}`)
		if stdout != "hi\n" {
			t.Errorf("want hi in stdout, got %q", stdout)
		}
	})

	t.Run("setenv", func(t *testing.T) {
		stdout, _ := mustRunStr(t, t.TempDir(), `:lua {
			sh.setenv("XYZ", "123")
		} && env | tail -n 1`)
		if !strings.Contains(stdout, "XYZ") {
			t.Errorf("want XYZ in stdout, got %q", stdout)
		}
	})


}
