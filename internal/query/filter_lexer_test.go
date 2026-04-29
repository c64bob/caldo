package query

import "testing"

func TestLexFilter_AllTokenTypes(t *testing.T) {
	input := "today overdue upcoming no_date completed priority text #work @home before after and or not ( ) : foo"
	tokens := LexFilter(input)

	wantTypes := []TokenType{
		TokenToday,
		TokenOverdue,
		TokenUpcoming,
		TokenNoDate,
		TokenCompleted,
		TokenPriority,
		TokenText,
		TokenProject,
		TokenLabel,
		TokenBefore,
		TokenAfter,
		TokenAnd,
		TokenOr,
		TokenNot,
		TokenLParen,
		TokenRParen,
		TokenColon,
		TokenString,
		TokenEOF,
	}

	if len(tokens) != len(wantTypes) {
		t.Fatalf("len(tokens)=%d want %d", len(tokens), len(wantTypes))
	}

	for i := range wantTypes {
		if tokens[i].Type != wantTypes[i] {
			t.Fatalf("token %d type=%s want %s", i, tokens[i].Type, wantTypes[i])
		}
	}
}

func TestLexFilter_CaseInsensitiveKeywords(t *testing.T) {
	tokens := LexFilter("today TODAY ToDaY")
	for i := 0; i < 3; i++ {
		if tokens[i].Type != TokenToday {
			t.Fatalf("token %d type=%s want %s", i, tokens[i].Type, TokenToday)
		}
	}
}

func TestLexFilter_WhitespaceSkipped(t *testing.T) {
	tokens := LexFilter("   today\n\t  and\r\n overdue  ")
	if tokens[0].Type != TokenToday || tokens[1].Type != TokenAnd || tokens[2].Type != TokenOverdue || tokens[3].Type != TokenEOF {
		t.Fatalf("unexpected tokens: %#v", tokens)
	}
}

func TestLexFilter_UnknownAsStringAndSpecialChars(t *testing.T) {
	tokens := LexFilter("abc-123 ümlaut value.with.dot")
	if tokens[0].Type != TokenString || tokens[1].Type != TokenString || tokens[2].Type != TokenString || tokens[3].Type != TokenEOF {
		t.Fatalf("unexpected tokens: %#v", tokens)
	}
}

func TestLexFilter_EmptyInput(t *testing.T) {
	tokens := LexFilter("")
	if len(tokens) != 1 {
		t.Fatalf("len(tokens)=%d want 1", len(tokens))
	}
	if tokens[0].Type != TokenEOF {
		t.Fatalf("token[0]=%s want %s", tokens[0].Type, TokenEOF)
	}
}

func TestLexFilter_NoDatePhrase(t *testing.T) {
	tokens := LexFilter("no date")
	if tokens[0].Type != TokenNoDate || tokens[1].Type != TokenEOF {
		t.Fatalf("unexpected tokens: %#v", tokens)
	}
}

func TestLexFilter_NoDatePhraseCaseInsensitive(t *testing.T) {
	tokens := LexFilter("No	DaTe")
	if tokens[0].Type != TokenNoDate || tokens[1].Type != TokenEOF {
		t.Fatalf("unexpected tokens: %#v", tokens)
	}
}
