package parser

import (
	"strconv"
	"strings"
	"time"
)

var nowFunc = time.Now

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

	dueDate, remaining := parseNaturalDue(titleTokens, nowFunc().UTC())
	draft.Due = dueDate
	draft.Title = strings.Join(remaining, " ")
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

func parseNaturalDue(tokens []string, now time.Time) (string, []string) {
	remaining := make([]string, 0, len(tokens))
	for i := 0; i < len(tokens); {
		matched, due, consumed := matchDueToken(tokens, i, now)
		if matched {
			return due, append(remaining, tokens[i+consumed:]...)
		}
		remaining = append(remaining, tokens[i])
		i++
	}
	return "", remaining
}

func matchDueToken(tokens []string, i int, now time.Time) (bool, string, int) {
	n := len(tokens)
	lower := strings.ToLower(tokens[i])
	switch lower {
	case "heute", "today":
		return true, now.Format("2006-01-02"), 1
	case "morgen", "tomorrow":
		return true, now.AddDate(0, 0, 1).Format("2006-01-02"), 1
	case "übermorgen":
		return true, now.AddDate(0, 0, 2).Format("2006-01-02"), 1
	}

	if i+2 < n && lower == "in" {
		if num, err := strconv.Atoi(tokens[i+1]); err == nil && num >= 0 {
			unit := strings.ToLower(tokens[i+2])
			if unit == "tagen" || unit == "tage" || unit == "days" || unit == "day" {
				return true, now.AddDate(0, 0, num).Format("2006-01-02"), 3
			}
		}
	}

	if i+1 < n && (lower == "next" || lower == "nächsten") {
		if wd, ok := weekdayToken(tokens[i+1]); ok {
			return true, nextWeekday(now, wd).Format("2006-01-02"), 2
		}
	}

	if wd, ok := weekdayToken(tokens[i]); ok {
		return true, nextWeekday(now, wd).Format("2006-01-02"), 1
	}
	return false, "", 0
}

func weekdayToken(token string) (time.Weekday, bool) {
	switch strings.ToLower(token) {
	case "montag", "monday":
		return time.Monday, true
	case "dienstag", "tuesday":
		return time.Tuesday, true
	case "mittwoch", "wednesday":
		return time.Wednesday, true
	case "donnerstag", "thursday":
		return time.Thursday, true
	case "freitag", "friday":
		return time.Friday, true
	case "samstag", "saturday":
		return time.Saturday, true
	case "sonntag", "sunday":
		return time.Sunday, true
	}
	return 0, false
}

func nextWeekday(now time.Time, target time.Weekday) time.Time {
	days := (int(target) - int(now.Weekday()) + 7) % 7
	if days == 0 {
		days = 7
	}
	return now.AddDate(0, 0, days)
}
