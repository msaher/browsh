package shell

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"syscall"
	"strings"
	"time"
	"errors"
)

type BuiltinFunc func(inter *Interpreter, cmd *Cmd, stdio Stdio)

var Builtins = map[string]BuiltinFunc{
	"cd":   builtinCd,
	"pwd":  builtinPwd,
	"echo": builtinEcho,
	":lua":  builtinLua,
}

// might want to move duration to Result
type Cmd struct {
	exec.Cmd
	Id int
	// for builtins
	IsBuiltin bool
	Done      chan int
}

type Stdio struct {
	Stdin io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

func NewStdio() Stdio {
	return Stdio {
		Stdin: os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
}

type Interpreter struct {
	Cwd    string
	Env    []string
	LastCmdId int
	CmdTable map[int]*Cmd
}

func NewInterpreter(cwd string) *Interpreter {

	return &Interpreter{
		Cwd:    cwd,
		Env: 	os.Environ(),
		CmdTable: make(map[int]*Cmd),
	}
}

func closeOutput(cmd *Cmd, stdio Stdio) {
    if cmd.Stdout != stdio.Stdout {
        if c, ok := cmd.Stdout.(io.Closer); ok {
            c.Close()
        }
    }
    if cmd.Stderr != stdio.Stderr && cmd.Stderr != cmd.Stdout {
        if c, ok := cmd.Stderr.(io.Closer); ok {
            c.Close()
        }
    }
}

func (inter *Interpreter) Exec(node *Node, stdio Stdio) *Result {
	result := Result{}
	inter.exec(node, stdio, &result)
	return &result
}

func (inter *Interpreter) ExecRes(node *Node, stdio Stdio, result *Result) {
	inter.exec(node, stdio, result)
}

func (inter *Interpreter) exec(node *Node, stdio Stdio, result *Result) {
	switch node.Token.Type {
	case TokenAndIf:
		for _, kid := range node.Kids {
			if inter.exec(kid, stdio, result); result.IsErr() {
				return
			}
		}
		return

	case TokenOrIf:
		for _, kid := range node.Kids {
			inter.exec(kid, stdio, result)
			if result.IsErr() {
				return
			}
		}
		return

	case TokenPipe:
		var cmds []*Cmd
		for _, kid := range node.Kids {
			cmd, err := inter.BuildCmd(kid, stdio)
			if err != nil {
				result.err = err
				return
			}
			cmds = append(cmds, cmd)
		}
		//  only the first cmd is registered
		if len(cmds) >= 1 {
			inter.RegisterCmd(cmds[0])
		}
		inter.RunPipe(cmds, stdio, result)
		return

	default:
		cmd, err := inter.BuildCmd(node, stdio)
		if err != nil {
			result.err = err
			return
		}
		inter.RegisterCmd(cmd)
		inter.CmdRun(cmd, stdio, result)
		return
	}
}

// TODO: add mutex
func (inter *Interpreter) RegisterCmd(cmd *Cmd) {
	cmd.Id = inter.LastCmdId
	inter.LastCmdId++
	inter.CmdTable[cmd.Id] = cmd
}

func (inter *Interpreter) NewCmd(stdio Stdio) *Cmd {
	return &Cmd{
		Cmd: exec.Cmd{
			Dir:    inter.Cwd,
			Env: 	inter.Env,
			Stdin:  stdio.Stdin,
			Stdout: stdio.Stdout,
			Stderr: stdio.Stderr,
		},
	}
}

// builds a Cmd from a cmd node, applying args and redirects.
func (inter *Interpreter) BuildCmd(node *Node, stdio Stdio) (*Cmd, error) {
	cmd := inter.NewCmd(stdio)
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
			if err := inter.ApplyDupOut(cmd, kid, stdio); err != nil {
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

// starts the command. external commands call cmd.Start; builtins run in a goroutine.
func (inter *Interpreter) CmdStart(cmd *Cmd, stdio Stdio, result *Result) error {
	result.SetStartedAt(time.Now())
	if !cmd.IsBuiltin {
		return cmd.Start()
	}
	fn := Builtins[cmd.Args[0]]
	cmd.Done = make(chan int, 1)
	go func() {
		fn(inter, cmd, stdio)
		closeOutput(cmd, stdio)
		close(cmd.Done)
	}()
	return nil
}

// waits for the command to finish, populates cmd.ExitCode, and returns any error.
func (inter *Interpreter) CmdWait(cmd *Cmd, stdio Stdio, result *Result) {
	result.SetCurrentCmd(cmd)
	if !cmd.IsBuiltin {
		err := cmd.Wait()
		result.SetExitedAt(time.Now())
		result.SetExitCode(cmd.ProcessState.ExitCode())
		closeOutput(cmd, stdio)

		// dont consider ExitError as a real error
		var exitErr *exec.ExitError
		if err != nil && !errors.As(err, &exitErr) {
			result.SetErr(err)
			return
		}
		return
	}
	code := <-cmd.Done
	result.SetExitedAt(time.Now())
	result.SetExitCode(code)
}

// runs the command to completion.
func (inter *Interpreter) CmdRun(cmd *Cmd, stdio Stdio, result *Result) {
	if inter.CmdStart(cmd, stdio, result); result.IsErr() {
		return
	}
	inter.CmdWait(cmd, stdio, result)
	return
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
func (inter *Interpreter) RunPipe(cmds []*Cmd, stdio Stdio, result *Result) {
	if err := WirePipe(cmds); err != nil {
		result.err = err
		return
	}

	// TODO: what about builtins?
	// first command creates a process group
	cmds[0].SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if inter.CmdStart(cmds[0], stdio, result); result.IsErr() {
		return
	}
	pgid := cmds[0].SysProcAttr.Pgid
	for _, cmd := range cmds[1:] {
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true, Pgid: pgid}
		if inter.CmdStart(cmd, stdio, result); result.IsErr() {
			return
		}
	}

	for _, cmd := range cmds {
		inter.CmdWait(cmd, stdio, result)
		if result.IsErr() {
			return
		}
	}
	return
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
func (inter *Interpreter) ApplyDupOut(cmd *Cmd, kid *Node, stdio Stdio) error {
	srcFd, dstFd, err := inter.ResolveDupOut(kid)
	if err != nil {
		return err
	}
	var dst io.Writer
	switch dstFd {
	case 1:
		if f, ok := cmd.Stdout.(*os.File); ok {
			dst = f
		} else {
			dst = stdio.Stdout
		}
	case 2:
		if f, ok := cmd.Stderr.(*os.File); ok {
			dst = f
		} else {
			dst = stdio.Stderr
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

func (inter *Interpreter) ExecStrRes(src string, stdio Stdio, result *Result) {
	root, err := Parse(src)
	if err != nil {
		result.err = err
		return
	}
	inter.exec(root, stdio, result)
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
