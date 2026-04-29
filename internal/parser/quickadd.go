package parser

import "strings"

// QuickAddDraft contains parsed quick-add values for preview and persistence.
type QuickAddDraft struct {
	Title             string
	ProjectID         string
	Project           string
	ProjectNew        bool
	ProjectUnresolved bool
	Labels            []string
	Due               string
	Recurrence        string
	Priority          string
}

// ParseQuickAdd extracts supported quick-add tokens and remaining title text.
func ParseQuickAdd(input string) QuickAddDraft {
	tokens := strings.Fields(strings.TrimSpace(input))
	draft := QuickAddDraft{Labels: make([]string, 0)}
	titleTokens := make([]string, 0, len(tokens))

	for _, token := range tokens {
		switch {
		case strings.HasPrefix(token, "#"):
			project := strings.TrimSpace(strings.TrimPrefix(token, "#"))
			if project != "" {
				draft.Project = project
				continue
			}
		case strings.HasPrefix(token, "@"):
			label := strings.TrimSpace(strings.TrimPrefix(token, "@"))
			if label != "" {
				draft.Labels = append(draft.Labels, label)
				continue
			}
		case strings.HasPrefix(token, "!"):
			if priority, ok := normalizePriorityToken(token); ok {
				draft.Priority = priority
				continue
			}
		}

		titleTokens = append(titleTokens, token)
	}

	draft.Title = strings.Join(titleTokens, " ")
	return draft
}

func normalizePriorityToken(token string) (string, bool) {
	switch strings.ToLower(strings.TrimPrefix(strings.TrimSpace(token), "!")) {
	case "high", "1":
		return "high", true
	case "medium", "2":
		return "medium", true
	case "low", "3":
		return "low", true
	default:
		return "", false
	}
}
