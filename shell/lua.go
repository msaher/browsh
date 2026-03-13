package shell

import (
	"fmt"
	"os"
	"os/exec"
	"io"
	"bufio"
	"strings"
	"bytes"

	"github.com/yuin/gopher-lua"
)

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
	    if err != nil && line == "" {
	        L.Push(lua.LNil)
	        return 1
	    }
	    L.Push(lua.LString(line))
	    return 1
	}))

	sh.RawSetString("exit", L.NewFunction(func(L *lua.LState) int {
		code := L.OptInt(1, 0)
		cmd.Done <- code
		L.RaiseError("exit")
		return 0
	}))

	sh.RawSetString("run", L.NewFunction(func(L *lua.LState) int {
		cmdStr := L.CheckString(1)
		env := inter.Env

		if tbl := L.OptTable(2, nil); tbl != nil {
			merged := make([]string, len(inter.Env))
			copy(merged, inter.Env)
			tbl.ForEach(func(k, v lua.LValue) {
				merged = append(merged, k.String()+"="+v.String())
			})
			env = merged
		}

		var stdout, stderr bytes.Buffer
		c := exec.Command("bash", "-c", cmdStr)
		c.Dir = inter.Cwd
		c.Env = env
		c.Stdout = &stdout
		c.Stderr = &stderr
		c.Run()

		L.Push(lua.LString(stdout.String()))
		L.Push(lua.LNumber(c.ProcessState.ExitCode()))
		L.Push(lua.LString(stderr.String()))
		return 3
	}))

	sh.RawSetString("setenv", L.NewFunction(func(L *lua.LState) int {
		key := L.CheckString(1)
		value := L.CheckString(2)
		keyValue := fmt.Sprintf("%s=%s", key, value)
		inter.Env = append(inter.Env, keyValue)
		_ = os.Setenv(key, value) // ignore error
		return 0
	}))

	L.SetGlobal("sh", sh)
}
