package shell

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"strings"
)

type Interpreter struct {
	Cwd    string
	Stdin  *os.File
	Stdout *os.File
	Stderr *os.File
}

func NewInterpreter(cwd string) *Interpreter {
	return &Interpreter{
		Cwd:    cwd,
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
}

func (inter *Interpreter) Exec(node *Node) error {
	switch node.Token.Type {
	case TokenAndIf:
		if err := inter.Exec(node.Kids[0]); err != nil {
			return err
		}
		return inter.Exec(node.Kids[1])

	case TokenOrIf:
		err := inter.Exec(node.Kids[0])
		if err == nil {
			return nil
		}
		return inter.Exec(node.Kids[1])

	default:
		// zero token type = cmd node
		return inter.ExecCmd(node)
	}
}

func (inter *Interpreter) ExecCmd(node *Node) error {
	cmd := &exec.Cmd{
		Dir:    inter.Cwd,
		Stdin:  inter.Stdin,
		Stdout: inter.Stdout,
		Stderr: inter.Stderr,
	}

	for _, kid := range node.Kids {
		switch kid.Token.Type {
		case TokenWord:
			expanded, err := inter.ExpandWord(kid.Token.Content)
			if err != nil {
				return err
			}
			cmd.Args = append(cmd.Args, expanded...)

		case TokenString:
			cmd.Args = append(cmd.Args, kid.Token.Content)

		case TokenOut:
			if err := inter.ApplyOut(cmd, kid); err != nil {
				return err
			}

		case TokenAppend:
			if err := inter.ApplyAppend(cmd, kid); err != nil {
				return err
			}

		case TokenIn:
			if err := inter.ApplyIn(cmd, kid); err != nil {
				return err
			}

		case TokenDupOut:
			if err := inter.ApplyDupOut(cmd, kid); err != nil {
				return err
			}
		}
	}

	if len(cmd.Args) == 0 {
		return fmt.Errorf("empty command")
	}
	cmd.Path, _ = exec.LookPath(cmd.Args[0])
	return cmd.Run()
}

// sets cmd.Stdout or cmd.Stderr for a ">" redirect node.
func (inter *Interpreter) ApplyOut(cmd *exec.Cmd, kid *Node) error {
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
func (inter *Interpreter) ApplyAppend(cmd *exec.Cmd, kid *Node) error {
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
func (inter *Interpreter) ApplyIn(cmd *exec.Cmd, kid *Node) error {
	target := kid.Kids[0].Token.Content
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
func (inter *Interpreter) ApplyDupOut(cmd *exec.Cmd, kid *Node) error {
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

// returns (fd, absolute path) from a > or >> node.
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

	target := targetNode.Token.Content
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

func (inter *Interpreter) ExpandWord(word string) ([]string, error) {
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
