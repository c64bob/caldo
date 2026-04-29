package query

import (
	"fmt"
)

// Node represents a filter expression AST node.
type Node interface {
	node()
}

// AndNode represents a logical AND between two expressions.
type AndNode struct {
	Left  Node
	Right Node
}

func (AndNode) node() {}

// OrNode represents a logical OR between two expressions.
type OrNode struct {
	Left  Node
	Right Node
}

func (OrNode) node() {}

// NotNode represents a logical NOT for one expression.
type NotNode struct {
	Expr Node
}

func (NotNode) node() {}

// FilterNode is a leaf node with an operator and value.
type FilterNode struct {
	Operator TokenType
	Value    string
}

func (FilterNode) node() {}

// ParseError indicates an invalid filter syntax.
type ParseError struct {
	Message string
}

func (e *ParseError) Error() string {
	return e.Message
}

// ParseFilter parses filter tokens into an AST.
func ParseFilter(tokens []Token) (Node, error) {
	p := &filterParser{tokens: tokens}
	node, err := p.parseExpression()
	if err != nil {
		return nil, err
	}

	if p.current().Type != TokenEOF {
		return nil, p.errorf("unexpected token %q", p.current().Literal)
	}

	return node, nil
}

type filterParser struct {
	tokens []Token
	pos    int
}

func (p *filterParser) parseExpression() (Node, error) {
	return p.parseOr()
}

func (p *filterParser) parseOr() (Node, error) {
	left, err := p.parseAnd()
	if err != nil {
		return nil, err
	}

	for p.current().Type == TokenOr {
		p.next()
		right, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		left = OrNode{Left: left, Right: right}
	}

	return left, nil
}

func (p *filterParser) parseAnd() (Node, error) {
	left, err := p.parseNot()
	if err != nil {
		return nil, err
	}

	for p.current().Type == TokenAnd {
		p.next()
		right, err := p.parseNot()
		if err != nil {
			return nil, err
		}
		left = AndNode{Left: left, Right: right}
	}

	return left, nil
}

func (p *filterParser) parseNot() (Node, error) {
	if p.current().Type == TokenNot {
		p.next()
		expr, err := p.parseNot()
		if err != nil {
			return nil, err
		}
		return NotNode{Expr: expr}, nil
	}
	return p.parsePrimary()
}

func (p *filterParser) parsePrimary() (Node, error) {
	tok := p.current()

	switch tok.Type {
	case TokenLParen:
		p.next()
		node, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		if p.current().Type != TokenRParen {
			return nil, p.errorf("missing closing parenthesis")
		}
		p.next()
		return node, nil
	case TokenToday, TokenOverdue, TokenUpcoming, TokenNoDate, TokenCompleted:
		p.next()
		return FilterNode{Operator: tok.Type}, nil
	case TokenProject, TokenLabel:
		p.next()
		return FilterNode{Operator: tok.Type, Value: tok.Literal}, nil
	case TokenPriority, TokenText, TokenBefore, TokenAfter:
		p.next()
		if p.current().Type == TokenColon {
			p.next()
		}
		if p.current().Type != TokenString {
			return nil, p.errorf("expected value after %q", tok.Literal)
		}
		value := p.current().Literal
		p.next()
		return FilterNode{Operator: tok.Type, Value: value}, nil
	case TokenString:
		p.next()
		return FilterNode{Operator: TokenString, Value: tok.Literal}, nil
	default:
		return nil, p.errorf("unexpected token %q", tok.Literal)
	}
}

func (p *filterParser) current() Token {
	if p.pos >= len(p.tokens) {
		return Token{Type: TokenEOF}
	}
	return p.tokens[p.pos]
}

func (p *filterParser) next() {
	if p.pos < len(p.tokens) {
		p.pos++
	}
}

func (p *filterParser) errorf(format string, args ...any) error {
	return &ParseError{Message: fmt.Sprintf(format, args...)}
}
