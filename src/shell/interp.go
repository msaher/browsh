package shell

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"strings"
)

type BuiltinFunc func(inter *Interpreter, cmd *Cmd)

var Builtins = map[string]BuiltinFunc{
	"cd":   builtinCd,
	"pwd":  builtinPwd,
	"echo": builtinEcho,
	":lua":  builtinLua,
}

type Cmd struct {
	exec.Cmd
	IsBuiltin bool
	Done      chan int
	ExitCode  int
}

type Interpreter struct {
	Cwd    string
	Env    []string
	Stdin  *os.File
	Stdout *os.File
	Stderr *os.File
}

func NewInterpreter(cwd string) *Interpreter {

	return &Interpreter{
		Cwd:    cwd,
		Env: 	os.Environ(),
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
}

// closes cmd's stdout and stderr if they are files that don't belong to the
// interpreter's own streams. handles both redirect files and pipe write ends.
func closeOutput(inter *Interpreter, cmd *Cmd) {
	if f, ok := cmd.Stdout.(*os.File); ok && f != nil {
		if f != inter.Stdout && f != inter.Stderr {
			f.Close()
		}
	}

	if cmd.Stdout == cmd.Stderr {
		return
	}

	if f, ok := cmd.Stderr.(*os.File); ok && f != nil {
		if f != inter.Stdout && f != inter.Stderr {
			f.Close()
		}
	}
}

func (inter *Interpreter) Exec(node *Node) error {
	switch node.Token.Type {
	case TokenAndIf:
		for _, kid := range node.Kids {
			if err := inter.Exec(kid); err != nil {
				return err
			}
		}
		return nil

	case TokenOrIf:
		var err error
		for _, kid := range node.Kids {
			err = inter.Exec(kid)
			if err == nil {
				return nil
			}
		}
		return err

	case TokenPipe:
		var cmds []*Cmd
		for _, kid := range node.Kids {
			cmd, err := inter.BuildCmd(kid)
			if err != nil {
				return err
			}
			cmds = append(cmds, cmd)
		}
		return inter.RunPipe(cmds)

	default:
		cmd, err := inter.BuildCmd(node)
		if err != nil {
			return err
		}
		return inter.CmdRun(cmd)
	}
}

// builds a Cmd from a cmd node, applying args and redirects.
func (inter *Interpreter) BuildCmd(node *Node) (*Cmd, error) {
	cmd := &Cmd{
		Cmd: exec.Cmd{
			Dir:    inter.Cwd,
			Env: 	inter.Env,
			Stdin:  inter.Stdin,
			Stdout: inter.Stdout,
			Stderr: inter.Stderr,
		},
	}

	for _, kid := range node.Kids {
		switch kid.Token.Type {
		case TokenWord:
			expanded, err := inter.ExpandWord(kid.Token.Content)
			if err != nil {
				return nil, err
			}
			cmd.Args = append(cmd.Args, expanded...)

		case TokenBlock:
			cmd.Args = append(cmd.Args, kid.Token.Content)

		case TokenString:
			// skip quotes
			content := kid.Token.Content
			str := content[1:len(content)-1]
			cmd.Args = append(cmd.Args, str)

		case TokenOut:
			if err := inter.ApplyOut(cmd, kid); err != nil {
				return nil, err
			}

		case TokenAppend:
			if err := inter.ApplyAppend(cmd, kid); err != nil {
				return nil, err
			}

		case TokenIn:
			if err := inter.ApplyIn(cmd, kid); err != nil {
				return nil, err
			}

		case TokenDupOut:
			if err := inter.ApplyDupOut(cmd, kid); err != nil {
				return nil, err
			}
		}
	}

	if len(cmd.Args) == 0 {
		return nil, fmt.Errorf("empty command")
	}

	_, cmd.IsBuiltin = Builtins[cmd.Args[0]]
	if !cmd.IsBuiltin {
		cmd.Path, _ = exec.LookPath(cmd.Args[0])
	}

	return cmd, nil
}

func (inter *Interpreter) ExecCmd(node *Node) error {
	cmd, err := inter.BuildCmd(node)
	if err != nil {
		return err
	}
	return inter.CmdRun(cmd)
}

// starts the command. external commands call cmd.Start; builtins run in a goroutine.
func (inter *Interpreter) CmdStart(cmd *Cmd) error {
	if !cmd.IsBuiltin {
		return cmd.Start()
	}
	fn := Builtins[cmd.Args[0]]
	cmd.Done = make(chan int, 1)
	go func() {
		fn(inter, cmd)
		closeOutput(inter, cmd)
		close(cmd.Done)
	}()
	return nil
}

// waits for the command to finish, populates cmd.ExitCode, and returns any error.
func (inter *Interpreter) CmdWait(cmd *Cmd) error {
	if !cmd.IsBuiltin {
		err := cmd.Wait()
		closeOutput(inter, cmd)
		if err != nil {
			if exit, ok := err.(*exec.ExitError); ok {
				cmd.ExitCode = exit.ExitCode()
			}
		}
		return err
	}
	code := <-cmd.Done
	cmd.ExitCode = code
	if code != 0 {
		return fmt.Errorf("exit status %d", code)
	}
	return nil
}

// runs the command to completion.
func (inter *Interpreter) CmdRun(cmd *Cmd) error {
	if err := inter.CmdStart(cmd); err != nil {
		return err
	}
	return inter.CmdWait(cmd)
}

// connects each cmd's stdout to the next cmd's stdin using os.Pipe.
// we use os.Pipe directly rather than cmd.StdoutPipe so we control
// when the write end closes — closeOutput handles that uniformly.
func WirePipe(cmds []*Cmd) error {
	for i := 0; i < len(cmds)-1; i++ {
		r, w, err := os.Pipe()
		if err != nil {
			return err
		}
		cmds[i].Stdout = w
		cmds[i+1].Stdin = r
	}
	return nil
}

// starts and waits for a pipeline, returning the exit status of the last command.
func (inter *Interpreter) RunPipe(cmds []*Cmd) error {
	if err := WirePipe(cmds); err != nil {
		return err
	}

	for _, cmd := range cmds {
		if err := inter.CmdStart(cmd); err != nil {
			return err
		}
	}

	var lastErr error
	for _, cmd := range cmds {
		lastErr = inter.CmdWait(cmd)
	}
	return lastErr
}

// ApplyOut sets cmd.Stdout or cmd.Stderr for a ">" redirect node.
func (inter *Interpreter) ApplyOut(cmd *Cmd, kid *Node) error {
	fd, target, err := inter.ResolveOutTarget(kid)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	switch fd {
	case 1:
		cmd.Stdout = f
	case 2:
		cmd.Stderr = f
	default:
		f.Close()
		return fmt.Errorf("unsupported fd %d for >", fd)
	}
	return nil
}

// sets cmd.Stdout for a ">>" redirect node.
func (inter *Interpreter) ApplyAppend(cmd *Cmd, kid *Node) error {
	_, target, err := inter.ResolveOutTarget(kid)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	cmd.Stdout = f
	return nil
}

// sets cmd.Stdin for a "<" redirect node.
func (inter *Interpreter) ApplyIn(cmd *Cmd, kid *Node) error {
	target := ExpandTilde(kid.Kids[0].Token.Content)
	if !IsAbs(target) {
		target = inter.Cwd + "/" + target
	}
	f, err := os.Open(target)
	if err != nil {
		return err
	}
	cmd.Stdin = f
	return nil
}

// wires one of cmd's streams to another for a ">&" node.
func (inter *Interpreter) ApplyDupOut(cmd *Cmd, kid *Node) error {
	srcFd, dstFd, err := inter.ResolveDupOut(kid)
	if err != nil {
		return err
	}
	var dst *os.File
	switch dstFd {
	case 1:
		if f, ok := cmd.Stdout.(*os.File); ok {
			dst = f
		} else {
			dst = inter.Stdout
		}
	case 2:
		if f, ok := cmd.Stderr.(*os.File); ok {
			dst = f
		} else {
			dst = inter.Stderr
		}
	default:
		return fmt.Errorf("unsupported dup target fd %d", dstFd)
	}
	switch srcFd {
	case 1:
		cmd.Stdout = dst
	case 2:
		cmd.Stderr = dst
	default:
		return fmt.Errorf("unsupported dup source fd %d", srcFd)
	}
	return nil
}

// ResolveOutTarget returns (fd, absolute path) from a > or >> node.
// kids are either [target] or [fd, target].
func (inter *Interpreter) ResolveOutTarget(kid *Node) (int, string, error) {
	fd := 1
	var targetNode *Node

	if len(kid.Kids) == 2 {
		n, err := ParseFd(kid.Kids[0].Token.Content)
		if err != nil {
			return 0, "", err
		}
		fd = n
		targetNode = kid.Kids[1]
	} else {
		targetNode = kid.Kids[0]
	}

	target := ExpandTilde(targetNode.Token.Content)
	if !IsAbs(target) {
		target = inter.Cwd + "/" + target
	}
	return fd, target, nil
}

// returns (srcFd, dstFd) from a >& node.
// kids are either [dstFd] or [srcFd, dstFd].
func (inter *Interpreter) ResolveDupOut(kid *Node) (int, int, error) {
	srcFd := 1
	var dstNode *Node

	if len(kid.Kids) == 2 {
		n, err := ParseFd(kid.Kids[0].Token.Content)
		if err != nil {
			return 0, 0, err
		}
		srcFd = n
		dstNode = kid.Kids[1]
	} else {
		dstNode = kid.Kids[0]
	}

	dstFd, err := ParseFd(dstNode.Token.Content)
	if err != nil {
		return 0, 0, err
	}
	return srcFd, dstFd, nil
}

func ExpandTilde(s string) string {
	if s == "~" {
		return os.Getenv("HOME")
	}
	if strings.HasPrefix(s, "~/") {
		return os.Getenv("HOME") + s[1:]
	}
	return s
}

func (inter *Interpreter) ExpandWord(word string) ([]string, error) {
	word = ExpandTilde(word)
	if !ContainsGlob(word) {
		return []string{word}, nil
	}

	orig, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	if err := os.Chdir(inter.Cwd); err != nil {
		return nil, err
	}
	matches, globErr := fs.Glob(os.DirFS("."), word)
	_ = os.Chdir(orig)
	if globErr != nil {
		return nil, globErr
	}
	if len(matches) == 0 {
		return []string{word}, nil
	}
	return matches, nil
}

func ContainsGlob(s string) bool {
	return strings.ContainsAny(s, "*?[")
}

func ParseFd(s string) (int, error) {
	var n int
	if _, err := fmt.Sscan(s, &n); err != nil {
		return 0, fmt.Errorf("invalid fd %q", s)
	}
	return n, nil
}

func IsAbs(path string) bool {
	return len(path) > 0 && path[0] == '/'
}
