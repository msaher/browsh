package shell

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func (inter *Interpreter) Complete(src string, cursor int) []string {
	src = src[:cursor]
	word := currentWord(src)
	isFirst := isFirstWord(src)

	if isFirst {
		return inter.completeCommand(word)
	}
	return inter.completePath(word)
}

func (inter *Interpreter) completeCommand(prefix string) []string {
	var results []string

	for name := range Builtins {
		if strings.HasPrefix(name, prefix) {
			results = append(results, name)
		}
	}

	path := os.Getenv("PATH")
	for _, dir := range filepath.SplitList(path) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			name := e.Name()
			if !strings.HasPrefix(name, prefix) {
				continue
			}
			full := filepath.Join(dir, name)
			if _, err := exec.LookPath(full); err == nil {
				results = append(results, name)
			}
		}
	}

	return results
}

func (inter *Interpreter) completePath(word string) []string {
	word = ExpandTilde(word)

	dir := inter.Cwd
	prefix := ""

	if word == "" {
		dir = inter.Cwd
	} else if strings.Contains(word, "/") {
		dir = filepath.Dir(word)
		prefix = filepath.Base(word)
		if !IsAbs(dir) {
			dir = filepath.Join(inter.Cwd, dir)
		}
	} else {
		dir = inter.Cwd
		prefix = word
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	var results []string
	for _, e := range entries {
		name := e.Name()
		if !strings.HasPrefix(name, prefix) {
			continue
		}
		candidate := name
		if strings.Contains(word, "/") {
			candidate = filepath.Join(filepath.Dir(word), name)
		}
		if e.IsDir() {
			candidate += "/"
		}
		results = append(results, candidate)
	}
	return results
}

func currentWord(src string) string {
	fields := strings.Fields(src)
	if len(fields) == 0 {
		return ""
	}
	if src[len(src)-1] == ' ' {
		return ""
	}
	return fields[len(fields)-1]
}

func isFirstWord(src string) bool {
	fields := strings.Fields(src)
	return len(fields) <= 1 && (len(src) == 0 || src[len(src)-1] != ' ')
}
