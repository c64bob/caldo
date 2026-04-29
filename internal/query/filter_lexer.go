package query

import "strings"

// TokenType identifies a lexical token in a filter expression.
type TokenType string

const (
	TokenToday     TokenType = "TODAY"
	TokenOverdue   TokenType = "OVERDUE"
	TokenUpcoming  TokenType = "UPCOMING"
	TokenNoDate    TokenType = "NO_DATE"
	TokenCompleted TokenType = "COMPLETED"
	TokenPriority  TokenType = "PRIORITY"
	TokenText      TokenType = "TEXT"
	TokenProject   TokenType = "PROJECT"
	TokenLabel     TokenType = "LABEL"
	TokenBefore    TokenType = "BEFORE"
	TokenAfter     TokenType = "AFTER"
	TokenAnd       TokenType = "AND"
	TokenOr        TokenType = "OR"
	TokenNot       TokenType = "NOT"
	TokenLParen    TokenType = "LPAREN"
	TokenRParen    TokenType = "RPAREN"
	TokenColon     TokenType = "COLON"
	TokenString    TokenType = "STRING"
	TokenEOF       TokenType = "EOF"
)

// Token represents one tokenized value from a filter expression.
type Token struct {
	Type    TokenType
	Literal string
}

// LexFilter converts a filter expression string into lexical tokens.
func LexFilter(input string) []Token {
	tokens := make([]Token, 0)
	for i := 0; i < len(input); {
		ch := input[i]
		if isWhitespace(ch) {
			i++
			continue
		}

		switch ch {
		case '(':
			tokens = append(tokens, Token{Type: TokenLParen, Literal: "("})
			i++
		case ')':
			tokens = append(tokens, Token{Type: TokenRParen, Literal: ")"})
			i++
		case ':':
			tokens = append(tokens, Token{Type: TokenColon, Literal: ":"})
			i++
		case '#':
			literal, next := readPrefixed(input, i, '#')
			tokens = append(tokens, Token{Type: TokenProject, Literal: literal})
			i = next
		case '@':
			literal, next := readPrefixed(input, i, '@')
			tokens = append(tokens, Token{Type: TokenLabel, Literal: literal})
			i = next
		default:
			literal, next := readWord(input, i)

			if strings.EqualFold(literal, "no") {
				if combined, end, ok := readNoDatePhrase(input, i, next); ok {
					tokens = append(tokens, Token{Type: TokenNoDate, Literal: combined})
					i = end
					continue
				}
			}

			tokens = append(tokens, Token{Type: classifyWord(literal), Literal: literal})
			i = next
		}
	}
	tokens = append(tokens, Token{Type: TokenEOF})
	return tokens
}

func classifyWord(word string) TokenType {
	switch strings.ToUpper(word) {
	case "TODAY":
		return TokenToday
	case "OVERDUE":
		return TokenOverdue
	case "UPCOMING":
		return TokenUpcoming
	case "NO_DATE":
		return TokenNoDate
	case "COMPLETED":
		return TokenCompleted
	case "PRIORITY":
		return TokenPriority
	case "TEXT":
		return TokenText
	case "BEFORE":
		return TokenBefore
	case "AFTER":
		return TokenAfter
	case "AND":
		return TokenAnd
	case "OR":
		return TokenOr
	case "NOT":
		return TokenNot
	default:
		return TokenString
	}
}

func readNoDatePhrase(input string, wordStart, wordEnd int) (string, int, bool) {
	i := wordEnd
	for i < len(input) && isWhitespace(input[i]) {
		i++
	}

	if i == wordEnd {
		return "", 0, false
	}

	nextWord, nextEnd := readWord(input, i)
	if !strings.EqualFold(nextWord, "date") {
		return "", 0, false
	}

	return input[wordStart:nextEnd], nextEnd, true
}

func readWord(input string, start int) (string, int) {
	i := start
	for i < len(input) && !isDelimiter(input[i]) {
		i++
	}
	return input[start:i], i
}

func readPrefixed(input string, start int, prefix byte) (string, int) {
	i := start + 1
	for i < len(input) && !isDelimiter(input[i]) {
		i++
	}
	if i == start+1 {
		return string(prefix), i
	}
	return input[start:i], i
}

func isDelimiter(ch byte) bool {
	return isWhitespace(ch) || ch == '(' || ch == ')' || ch == ':' || ch == '#' || ch == '@'
}

func isWhitespace(ch byte) bool {
	return ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r'
}
