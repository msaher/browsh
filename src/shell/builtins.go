package shell

import (
	"fmt"
	"os"
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
