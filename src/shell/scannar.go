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

type Scanner struct {
	Src string
	Start int
	Current int
	Line int
	PreviousType TokenType
	ExpectBlock bool
}

func NewScannar(src string) *Scanner {
	return &Scanner {
		Src: src,
		Line: 1,
		PreviousType: TokenError,
	}
}

func (s *Scanner) IsAtEnd() bool {
	return len(s.Src) == s.Current
}

func (s *Scanner) Content() string {
	return s.Src[s.Start:s.Current]
}

func (s *Scanner) Advance() byte {
	c := s.Src[s.Current]
	s.Current++
	return c
}

func (s *Scanner) MakeToken(tt TokenType) Token {
	s.PreviousType = tt
	return Token {
		Type: tt,
		Line: s.Line,
		Content: s.Content(),
	}
}

func (s *Scanner) Error(msg string) Token {
	t := s.MakeToken(TokenError)
	t.Content = msg
	return t
}

func (s *Scanner) Peek() byte {
	if s.IsAtEnd() {
		return 0
	}
	return s.Src[s.Current]
}

func (s *Scanner) SkipWhitespace() {
	for {
		if s.IsAtEnd() {
			return
		}

		switch s.Peek() {

		case ' ', '\r', '\t':
			s.Advance()

		case '\n':
			s.Line++
			s.Advance()

		case '#':
			// skip comment until newline or EOF
			for !s.IsAtEnd() && s.Peek() != '\n' {
				s.Advance()
			}

		default:
			return
		}
	}
}

func (s *Scanner) ScanString() Token {
	for {
		if s.IsAtEnd() {
			t := s.Error("unterminated string")
			return t
		}

		if s.Peek() == '"' {
			break
		}

		if s.Peek() == '\n' {
			s.Line++
		}

		s.Advance()
	}

	// consume closing quote
	s.Advance()
	return s.MakeToken(TokenString)
}

func IsWordChar(ch byte) bool {
	switch ch {
	case ' ', '\r', '\t', '\n',
		'>', '<', '|', '&', '"', 0:
		return false
	default:
		return true
	}
}

func (s *Scanner) ScanWord() Token {
	for IsWordChar(s.Peek()) {
		s.Advance()
	}

	w := s.Content()
	// case A: 1> ...
	if (w == "1" || w == "2") && s.Peek() == '>' {
		return s.MakeToken(TokenFd)

	// case B: ... >&2
	} else if s.PreviousType == TokenDupOut {
		return s.MakeToken(TokenFd)
	}

	// :py {}
	if w == ":py" {
		s.ExpectBlock = true
	}

	return s.MakeToken(TokenWord)
}

func (s *Scanner) ScanBlock() Token {
	c := s.Advance()
	if c != '{' {
		return s.Error("expected a block")
	}

	depth := 1
	for depth != 0 && !s.IsAtEnd() {
		c = s.Advance()
		switch c {
		case '\n': s.Line++
		case '{': depth++
		case '}': depth--
		}
	}

	if depth != 0 {
		return s.Error("unclosed block")
	}

	return s.MakeToken(TokenBlock)
}

func (s *Scanner) Next() Token {
	s.SkipWhitespace()
	s.Start = s.Current

	if s.IsAtEnd() {
		return s.MakeToken(TokenEOF)
	}

	if s.ExpectBlock {
		s.ExpectBlock = false
		return s.ScanBlock()
	}

	c := s.Advance()

	switch c {

	case '"':
		return s.ScanString()

	case '>':
		if s.Match('>') {
			return s.MakeToken(TokenAppend)
		} else if s.Match('&') {
			return s.MakeToken(TokenDupOut)
		}
		return s.MakeToken(TokenOut)

	case '<':
		return s.MakeToken(TokenIn)

	case '|':
		if s.Match('|') {
			return s.MakeToken(TokenOrIf)
		}
		return s.MakeToken(TokenPipe)

	case '&':
		if s.Match('&') {
			return s.MakeToken(TokenAndIf)
		}
		return s.Error("expected a second &")
	}

	if IsWordChar(c) {
		return s.ScanWord()
	}

	return s.MakeToken(TokenError)
}

func (s *Scanner) Match(expected byte) bool {
	if s.IsAtEnd() {
		return false
	}
	if s.Src[s.Current] != expected {
		return false
	}
	s.Advance()
	return true
}

func Scan(src string) ([]Token, error) {
	s := NewScannar(src)
	var tokens []Token

	for {
		tok := s.Next()
		tokens = append(tokens, tok)

		if tok.Type == TokenError {
			return tokens, fmt.Errorf("%s", tok.Content)
		}

		if tok.Type == TokenEOF {
			break
		}
	}

	return tokens, nil
}

