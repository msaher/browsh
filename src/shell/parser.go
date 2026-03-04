// s -> orif
// orif -> andif ("||" andif)*
// andif -> pipes ("&&" pipes)*
// pipes -> cmd ("|" cmd)*
// cmd -> redirect* (word | string) (word | string | redirect)*

// redirect -> out | dupout | in | append
// out -> [fd] ">" (fd | word | string)
// dupout -> [fd] ">&" fd
// in -> "<" (word | string)
// append -> ">>" (word | string)

package shell

import "fmt"

type Node struct {
	Token Token
	Kids []*Node
}

type Parser struct {
	Tokens []Token
	Pos    int
}

func NewParser(tokens []Token) *Parser {
	return &Parser{Tokens: tokens}
}

func (p *Parser) Peek() Token {
	if p.Pos < len(p.Tokens) {
		return p.Tokens[p.Pos]
	}
	return Token{Type: TokenEOF}
}

func (p *Parser) Consume() Token {
	t := p.Peek()
	if t.Type != TokenEOF {
		p.Pos++
	}
	return t
}

func (p *Parser) Expect(tt TokenType) (Token, error) {
	t := p.Peek()
	if t.Type != tt {
		return t, fmt.Errorf("line %d: expected token %d, got %d (%q)", t.Line, tt, t.Type, t.Content)
	}
	return p.Consume(), nil
}

// parses the full input and returns the root node.
func (p *Parser) Parse() (*Node, error) {
	node, err := p.ParseOrIf()
	if err != nil {
		return nil, err
	}
	if p.Peek().Type != TokenEOF {
		t := p.Peek()
		return nil, fmt.Errorf("line %d: unexpected token %q", t.Line, t.Content)
	}
	return node, nil
}

// orif -> andif ("||" andif)*
func (p *Parser) ParseOrIf() (*Node, error) {
	first, err := p.ParseAndIf()
	if err != nil {
		return nil, err
	}

	if p.Peek().Type != TokenOrIf {
		return first, nil
	}

	node := &Node{Token: p.Consume(), Kids: []*Node{first}}
	for {
		child, err := p.ParseAndIf()
		if err != nil {
			return nil, err
		}
		node.Kids = append(node.Kids, child)
		if p.Peek().Type != TokenOrIf {
			break
		}
		p.Consume()
	}
	return node, nil
}

// andif -> pipeline ("&&" pipeline)*
func (p *Parser) ParseAndIf() (*Node, error) {
	first, err := p.ParsePipeline()
	if err != nil {
		return nil, err
	}

	if p.Peek().Type != TokenAndIf {
		return first, nil
	}

	node := &Node{Token: p.Consume(), Kids: []*Node{first}}
	for {
		child, err := p.ParsePipeline()
		if err != nil {
			return nil, err
		}
		node.Kids = append(node.Kids, child)
		if p.Peek().Type != TokenAndIf {
			break
		}
		p.Consume()
	}
	return node, nil
}

// pipeline -> cmd ("|" cmd)*
func (p *Parser) ParsePipeline() (*Node, error) {
	first, err := p.ParseCmd()
	if err != nil {
		return nil, err
	}

	if p.Peek().Type != TokenPipe {
		return first, nil
	}

	node := &Node{Token: p.Consume(), Kids: []*Node{first}}
	for {
		cmd, err := p.ParseCmd()
		if err != nil {
			return nil, err
		}
		node.Kids = append(node.Kids, cmd)
		if p.Peek().Type != TokenPipe {
			break
		}
		p.Consume()
	}
	return node, nil
}

// cmd -> redirect* (word|string) (word|string|redirect)*
// the cmd node uses a zero token (no meaningful token of its own).
func (p *Parser) ParseCmd() (*Node, error) {
	cmd := &Node{}

	for p.IsRedirectStart() {
		r, err := p.ParseRedirect()
		if err != nil {
			return nil, err
		}
		cmd.Kids = append(cmd.Kids, r)
	}

	if p.Peek().Type != TokenWord && p.Peek().Type != TokenString {
		t := p.Peek()
		return nil, fmt.Errorf("line %d: expected command word, got %q", t.Line, t.Content)
	}
	cmd.Kids = append(cmd.Kids, &Node{Token: p.Consume()})

	for {
		switch p.Peek().Type {
		case TokenWord, TokenString:
			cmd.Kids = append(cmd.Kids, &Node{Token: p.Consume()})
		default:
			if p.IsRedirectStart() {
				r, err := p.ParseRedirect()
				if err != nil {
					return nil, err
				}
				cmd.Kids = append(cmd.Kids, r)
			} else {
				return cmd, nil
			}
		}
	}
}

// returns true when the next token can begin a redirect.
func (p *Parser) IsRedirectStart() bool {
	switch p.Peek().Type {
	case TokenFd, TokenOut, TokenDupOut, TokenIn, TokenAppend:
		return true
	}
	return false
}

// redirect -> out | dupout | in | append
//
// out    -> [fd] ">"  (fd | word | string)
// dupout -> [fd] ">&" fd
// in     -> "<"       (word | string)
// append -> ">>"      (word | string)
func (p *Parser) ParseRedirect() (*Node, error) {
	// optional leading fd for out / dupout.
	var fdNode *Node
	if p.Peek().Type == TokenFd {
		fd := p.Consume()
		switch p.Peek().Type {
		case TokenOut, TokenDupOut:
			fdNode = &Node{Token: fd}
		default:
			return nil, fmt.Errorf("line %d: fd %q not followed by > or >&", fd.Line, fd.Content)
		}
	}

	op := p.Peek()
	switch op.Type {
	case TokenOut:
		return p.ParseOut(fdNode)
	case TokenDupOut:
		return p.ParseDupOut(fdNode)
	case TokenIn:
		if fdNode != nil {
			return nil, fmt.Errorf("line %d: fd not valid before <", op.Line)
		}
		return p.ParseIn()
	case TokenAppend:
		if fdNode != nil {
			return nil, fmt.Errorf("line %d: fd not valid before >>", op.Line)
		}
		return p.ParseAppend()
	default:
		return nil, fmt.Errorf("line %d: expected redirect operator, got %q", op.Line, op.Content)
	}
}

// out -> [fd] ">" (fd | word | string)
func (p *Parser) ParseOut(fdNode *Node) (*Node, error) {
	op, _ := p.Expect(TokenOut)
	node := &Node{Token: op}
	if fdNode != nil {
		node.Kids = append(node.Kids, fdNode)
	}

	t := p.Peek()
	switch t.Type {
	case TokenFd, TokenWord, TokenString:
		node.Kids = append(node.Kids, &Node{Token: p.Consume()})
	default:
		return nil, fmt.Errorf("line %d: expected fd, word, or string after >", t.Line)
	}
	return node, nil
}

// dupout -> [fd] ">&" fd
func (p *Parser) ParseDupOut(fdNode *Node) (*Node, error) {
	op, _ := p.Expect(TokenDupOut)
	node := &Node{Token: op}
	if fdNode != nil {
		node.Kids = append(node.Kids, fdNode)
	}

	fd, err := p.Expect(TokenFd)
	if err != nil {
		return nil, fmt.Errorf("line %d: expected fd after >&", op.Line)
	}
	node.Kids = append(node.Kids, &Node{Token: fd})
	return node, nil
}

// in -> "<" (word | string)
func (p *Parser) ParseIn() (*Node, error) {
	op, _ := p.Expect(TokenIn)
	node := &Node{Token: op}

	t := p.Peek()
	switch t.Type {
	case TokenWord, TokenString:
		node.Kids = append(node.Kids, &Node{Token: p.Consume()})
	default:
		return nil, fmt.Errorf("line %d: expected word or string after <", t.Line)
	}
	return node, nil
}

// append -> ">>" (word | string)
func (p *Parser) ParseAppend() (*Node, error) {
	op, _ := p.Expect(TokenAppend)
	node := &Node{Token: op}

	t := p.Peek()
	switch t.Type {
	case TokenWord, TokenString:
		node.Kids = append(node.Kids, &Node{Token: p.Consume()})
	default:
		return nil, fmt.Errorf("line %d: expected word or string after >>", t.Line)
	}
	return node, nil
}
