package shell

import (
	"fmt"
)

//go:generate go run ./gen/token_types_gen.go

type Token struct {
	Type TokenType
	Content string
	Line int
}
