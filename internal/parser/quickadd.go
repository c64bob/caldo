package parser

import "strings"

// QuickAddDraft contains parsed quick-add values for preview and persistence.
type QuickAddDraft struct {
	Title      string
	ProjectID  string
	Project    string
	Labels     []string
	Due        string
	Recurrence string
	Priority   string
}

// ParseQuickAdd extracts the free-text title from quick-add input.
func ParseQuickAdd(input string) QuickAddDraft {
	return QuickAddDraft{Title: strings.TrimSpace(input)}
}
