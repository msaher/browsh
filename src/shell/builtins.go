package shell

import (
	"fmt"
	"os"
	"io"
	"bufio"
	"path/filepath"
	"strings"

	"github.com/yuin/gopher-lua"
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

func builtinLua(inter *Interpreter, cmd *Cmd) {
	src := cmd.Args[1]
	src = src[1:len(src)-1] // strip braces

	L := lua.NewState()
	L.OpenLibs()
	defer L.Close()
	registerSh(L, inter, cmd)

	if err := L.DoString(src); err != nil {
		// "exit" is caused sh.exit() which writes cmd.Done for us already
		if err.Error() != "exit" {
			fmt.Fprintln(cmd.Stderr, err)
			cmd.Done <- 1
		}
		return
	}
	cmd.Done <- 0
}

func registerSh(L *lua.LState, inter *Interpreter, cmd *Cmd) {
	sh := L.NewTable()

	sh.RawSetString("print", L.NewFunction(func(L *lua.LState) int {
		n := L.GetTop()
		parts := make([]string, n)
		for i := 1; i <= n; i++ {
			parts[i-1] = L.ToStringMeta(L.Get(i)).String()
		}
		fmt.Fprintln(cmd.Stdout, strings.Join(parts, "\t"))
		return 0
	}))

	sh.RawSetString("write", L.NewFunction(func(L *lua.LState) int {
		s := L.CheckString(1)
		io.WriteString(cmd.Stdout, s)
		return 0
	}))

	sh.RawSetString("cwd", lua.LString(inter.Cwd))

	env := L.NewTable()
	for _, e := range os.Environ() {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			env.RawSetString(parts[0], lua.LString(parts[1]))
		}
	}
	sh.RawSetString("env", env)

	reader := bufio.NewReader(cmd.Stdin)
	sh.RawSetString("stdin", L.NewFunction(func(L *lua.LState) int {
		line, err := reader.ReadString('\n')
		line = strings.TrimRight(line, "\n")
		if line != "" {
			L.Push(lua.LString(line))
			return 1
		}
		if err != nil {
			L.Push(lua.LNil)
			return 1
		}
		L.Push(lua.LNil)
		return 1
	}))

	sh.RawSetString("exit", L.NewFunction(func(L *lua.LState) int {
		code := L.OptInt(1, 0)
		cmd.Done <- code
		L.RaiseError("exit")
		return 0
	}))

	L.SetGlobal("sh", sh)
}
