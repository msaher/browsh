package shell

import (
	"testing"
	"fmt"
)

func tokenTypes(tokens []Token) []TokenType {
	types := make([]TokenType, len(tokens))
