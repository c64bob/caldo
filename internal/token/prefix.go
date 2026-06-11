package token

// ReadPrefixed reads a prefixed token beginning at start and returns the literal and next offset.
func ReadPrefixed(input string, start int, prefix byte) (string, int) {
	i := start + 1
	for i < len(input) && !IsDelimiter(input[i]) {
		i++
	}
	if i == start+1 {
		return string(prefix), i
	}
	return input[start:i], i
}

// IsDelimiter reports whether ch separates project and label tokens.
func IsDelimiter(ch byte) bool {
	return IsWhitespace(ch) || ch == '(' || ch == ')' || ch == ':' || ch == '#' || ch == '@'
}

// IsWhitespace reports whether ch is ASCII whitespace used by token parsers.
func IsWhitespace(ch byte) bool {
	return ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r'
}
