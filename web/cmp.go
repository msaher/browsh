package main

import (
	"os/exec"
	"strings"
	"bufio"
)

func complete(s string) ([]string, error) {
	cmd := exec.Command("./cmp.exp", s)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	lines := splitLines(string(out))
	return lines, nil
}

func splitLines(s string) []string {
    var lines []string
    sc := bufio.NewScanner(strings.NewReader(s))
    for sc.Scan() {
        lines = append(lines, sc.Text())
    }
    return lines
}
