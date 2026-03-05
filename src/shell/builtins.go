package shell

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func builtinEcho(inter *Interpreter, cmd *Cmd) {
	out := strings.Join(cmd.Args[1:], " ")
	fmt.Fprintln(cmd.Stdout, out)
	cmd.Done <- 0
}

func builtinPwd(inter *Interpreter, cmd *Cmd) {
	fmt.Fprintln(cmd.Stdout, inter.Cwd)
	cmd.Done <- 0
}

func builtinCd(inter *Interpreter, cmd *Cmd) {
	var dir string
	switch len(cmd.Args) {
	case 1:
		// cd with no args goes home
		dir = os.Getenv("HOME")
		if dir == "" {
			fmt.Fprintln(cmd.Stderr, "cd: HOME not set")
			cmd.Done <- 1
			return
		}
	case 2:
		dir = cmd.Args[1]
	default:
		fmt.Fprintln(cmd.Stderr, "cd: too many arguments")
		cmd.Done <- 1
		return
	}

	if !IsAbs(dir) {
		dir = filepath.Join(inter.Cwd, dir)
	}

	info, err := os.Stat(dir)
	if err != nil {
		fmt.Fprintf(cmd.Stderr, "cd: %v\n", err)
		cmd.Done <- 1
		return
	}
	if !info.IsDir() {
		fmt.Fprintf(cmd.Stderr, "cd: %s: not a directory\n", dir)
		cmd.Done <- 1
		return
	}

	// dont change unless we're not in a pipeline
	if cmd.Stdout == inter.Stdout {
		inter.Cwd = dir
	}
	cmd.Done <- 0
}

func builtinPy(inter *Interpreter, cmd *Cmd) {
	src := cmd.Args[1]
	src = src[1:len(src)-1] // strip braces
	src = Dedent(src) // python is whitespace sensitive

	py := exec.Command("python")
	py.Stdin = strings.NewReader(src)
	py.Stdout = cmd.Stdout
	py.Stderr = cmd.Stderr
	py.Run()
	cmd.Done <- py.ProcessState.ExitCode()
}

func Dedent(src string) string {
	lines := strings.Split(src, "\n")

	minIndent := -1
	for _, line := range lines {
		trimmed := strings.TrimLeft(line, " \t")
		if trimmed == "" {
			continue
		}
		indent := len(line) - len(trimmed)
		if minIndent == -1 || indent < minIndent {
			minIndent = indent
		}
	}

	if minIndent <= 0 {
		return src
	}

	for i, line := range lines {
		if len(line) >= minIndent {
			lines[i] = line[minIndent:]
		}
	}

	return strings.TrimSpace(strings.Join(lines, "\n"))
}
