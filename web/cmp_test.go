package main

import (
	"testing"
)

func TestFiles(t *testing.T) {
	output, err := complete("p")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(output)
}
