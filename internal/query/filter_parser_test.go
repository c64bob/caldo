package query

import (
	"errors"
	"testing"
)

func TestParseFilter_SingleNodeToday(t *testing.T) {
	node, err := ParseFilter(LexFilter("today"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	leaf, ok := node.(FilterNode)
	if !ok {
		t.Fatalf("node type=%T want FilterNode", node)
	}
	if leaf.Operator != TokenToday {
		t.Fatalf("operator=%s want %s", leaf.Operator, TokenToday)
	}
}

func TestParseFilter_NodeTypesCovered(t *testing.T) {
	node, err := ParseFilter(LexFilter("not today and overdue or #work"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	orNode, ok := node.(OrNode)
	if !ok {
		t.Fatalf("root type=%T want OrNode", node)
	}

	andNode, ok := orNode.Left.(AndNode)
	if !ok {
		t.Fatalf("left type=%T want AndNode", orNode.Left)
	}

	notNode, ok := andNode.Left.(NotNode)
	if !ok {
		t.Fatalf("and left type=%T want NotNode", andNode.Left)
	}
	if _, ok := notNode.Expr.(FilterNode); !ok {
		t.Fatalf("not expr type=%T want FilterNode", notNode.Expr)
	}

	if _, ok := andNode.Right.(FilterNode); !ok {
		t.Fatalf("and right type=%T want FilterNode", andNode.Right)
	}

	if _, ok := orNode.Right.(FilterNode); !ok {
		t.Fatalf("or right type=%T want FilterNode", orNode.Right)
	}
}

func TestParseFilter_OperatorPrecedence(t *testing.T) {
	node, err := ParseFilter(LexFilter("today or overdue and not upcoming"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	orNode := mustType[OrNode](t, node)
	if mustType[FilterNode](t, orNode.Left).Operator != TokenToday {
		t.Fatalf("unexpected left operator")
	}

	andNode := mustType[AndNode](t, orNode.Right)
	if mustType[FilterNode](t, andNode.Left).Operator != TokenOverdue {
		t.Fatalf("unexpected and-left operator")
	}
	notNode := mustType[NotNode](t, andNode.Right)
	if mustType[FilterNode](t, notNode.Expr).Operator != TokenUpcoming {
		t.Fatalf("unexpected not expr operator")
	}
}

func TestParseFilter_ParenthesesOverridePrecedence(t *testing.T) {
	node, err := ParseFilter(LexFilter("(today or overdue) and upcoming"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	andNode := mustType[AndNode](t, node)
	orNode := mustType[OrNode](t, andNode.Left)
	if mustType[FilterNode](t, orNode.Left).Operator != TokenToday {
		t.Fatalf("unexpected or-left operator")
	}
	if mustType[FilterNode](t, orNode.Right).Operator != TokenOverdue {
		t.Fatalf("unexpected or-right operator")
	}
	if mustType[FilterNode](t, andNode.Right).Operator != TokenUpcoming {
		t.Fatalf("unexpected and-right operator")
	}
}

func TestParseFilter_MissingClosingParen(t *testing.T) {
	_, err := ParseFilter(LexFilter("(today and overdue"))
	if err == nil {
		t.Fatal("expected error")
	}

	var parseErr *ParseError
	if !errors.As(err, &parseErr) {
		t.Fatalf("error type=%T want *ParseError", err)
	}
}

func TestParseFilter_UnknownTokenUnexpectedPosition(t *testing.T) {
	_, err := ParseFilter([]Token{
		{Type: TokenToday, Literal: "today"},
		{Type: TokenRParen, Literal: ")"},
		{Type: TokenEOF},
	})
	if err == nil {
		t.Fatal("expected error")
	}

	var parseErr *ParseError
	if !errors.As(err, &parseErr) {
		t.Fatalf("error type=%T want *ParseError", err)
	}
}

func mustType[T any](t *testing.T, node Node) T {
	t.Helper()
	v, ok := node.(T)
	if !ok {
		t.Fatalf("node type=%T does not match expected", node)
	}
	return v
}
