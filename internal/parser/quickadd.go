package parser

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

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
	return ParseQuickAddWithLanguage(input, "de")
}

// ParseQuickAddWithLanguage parses quick-add text using language-specific natural tokens.
func ParseQuickAddWithLanguage(input string, language string) QuickAddDraft {
	return parseQuickAddAt(input, time.Now().UTC(), normalizeLanguage(language))
}

func parseQuickAddAt(input string, now time.Time, language string) QuickAddDraft {
	tokens := strings.Fields(strings.TrimSpace(input))
	draft := QuickAddDraft{Labels: make([]string, 0)}
	titleTokens := make([]string, 0, len(tokens))
	remainingAfterRecurrence := make([]string, 0, len(tokens))

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
	draft.Recurrence, remainingAfterRecurrence = parseNaturalRecurrence(titleTokens, language)

	dueDate, remaining := parseNaturalDue(remainingAfterRecurrence, now, language)
	draft.Due = dueDate
	draft.Title = strings.Join(remaining, " ")
	return draft
}

func parseNaturalRecurrence(tokens []string, language string) (string, []string) {
	remaining := make([]string, 0, len(tokens))
	for i := 0; i < len(tokens); {
		matched, rrule, consumed := matchRecurrenceToken(tokens, i, language)
		if matched {
			return rrule, append(remaining, tokens[i+consumed:]...)
		}
		remaining = append(remaining, tokens[i])
		i++
	}
	return "", remaining
}

func matchRecurrenceToken(tokens []string, i int, language string) (bool, string, int) {
	n := len(tokens)
	lower := strings.ToLower(tokens[i])
	switch lower {
	case "täglich", "daily":
		return true, "FREQ=DAILY", 1
	case "wöchentlich", "weekly":
		return true, "FREQ=WEEKLY", 1
	case "monatlich", "monthly":
		return true, "FREQ=MONTHLY", 1
	case "jährlich", "yearly":
		return true, "FREQ=YEARLY", 1
	case "werktags", "weekdays":
		return true, "FREQ=WEEKLY;BYDAY=MO,TU,WE,TH,FR", 1
	}

	if i+1 < n && ((language == "de" && lower == "jeden") || (language == "en" && lower == "every")) {
		if wd, ok := weekdayToken(tokens[i+1], language); ok {
			return true, "FREQ=WEEKLY;BYDAY=" + weekdayToICal(wd), 2
		}
	}

	if i+2 < n && lower == "alle" && language == "de" {
		if interval, err := strconv.Atoi(tokens[i+1]); err == nil && interval > 0 {
			switch strings.ToLower(tokens[i+2]) {
			case "tag", "tage", "tagen":
				return true, fmt.Sprintf("FREQ=DAILY;INTERVAL=%d", interval), 3
			case "woche", "wochen":
				return true, fmt.Sprintf("FREQ=WEEKLY;INTERVAL=%d", interval), 3
			case "monat", "monate", "monaten":
				return true, fmt.Sprintf("FREQ=MONTHLY;INTERVAL=%d", interval), 3
			}
		}
	}
	return false, "", 0
}

func weekdayToICal(wd time.Weekday) string {
	switch wd {
	case time.Monday:
		return "MO"
	case time.Tuesday:
		return "TU"
	case time.Wednesday:
		return "WE"
	case time.Thursday:
		return "TH"
	case time.Friday:
		return "FR"
	case time.Saturday:
		return "SA"
	case time.Sunday:
		return "SU"
	default:
		return ""
	}
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

func parseNaturalDue(tokens []string, now time.Time, language string) (string, []string) {
	remaining := make([]string, 0, len(tokens))
	for i := 0; i < len(tokens); {
		matched, due, consumed := matchDueToken(tokens, i, now, language)
		if matched {
			return due, append(remaining, tokens[i+consumed:]...)
		}
		remaining = append(remaining, tokens[i])
		i++
	}
	return "", remaining
}

func matchDueToken(tokens []string, i int, now time.Time, language string) (bool, string, int) {
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
			if (language == "de" && (unit == "tagen" || unit == "tage")) || (language == "en" && (unit == "days" || unit == "day")) {
				return true, now.AddDate(0, 0, num).Format("2006-01-02"), 3
			}
		}
	}

	if i+1 < n && ((language == "en" && lower == "next") || (language == "de" && lower == "nächsten")) {
		if wd, ok := weekdayToken(tokens[i+1], language); ok {
			return true, nextWeekday(now, wd).Format("2006-01-02"), 2
		}
	}

	if wd, ok := weekdayToken(tokens[i], language); ok {
		return true, nextWeekday(now, wd).Format("2006-01-02"), 1
	}
	return false, "", 0
}

func weekdayToken(token string, language string) (time.Weekday, bool) {
	switch strings.ToLower(token) {
	case "montag":
		if language != "de" {
			return 0, false
		}
		return time.Monday, true
	case "monday":
		if language != "en" {
			return 0, false
		}
		return time.Monday, true
	case "dienstag":
		if language != "de" {
			return 0, false
		}
		return time.Tuesday, true
	case "tuesday":
		if language != "en" {
			return 0, false
		}
		return time.Tuesday, true
	case "mittwoch":
		if language != "de" {
			return 0, false
		}
		return time.Wednesday, true
	case "wednesday":
		if language != "en" {
			return 0, false
		}
		return time.Wednesday, true
	case "donnerstag":
		if language != "de" {
			return 0, false
		}
		return time.Thursday, true
	case "thursday":
		if language != "en" {
			return 0, false
		}
		return time.Thursday, true
	case "freitag":
		if language != "de" {
			return 0, false
		}
		return time.Friday, true
	case "friday":
		if language != "en" {
			return 0, false
		}
		return time.Friday, true
	case "samstag":
		if language != "de" {
			return 0, false
		}
		return time.Saturday, true
	case "saturday":
		if language != "en" {
			return 0, false
		}
		return time.Saturday, true
	case "sonntag":
		if language != "de" {
			return 0, false
		}
		return time.Sunday, true
	case "sunday":
		if language != "en" {
			return 0, false
		}
		return time.Sunday, true
	}
	return 0, false
}

func normalizeLanguage(language string) string {
	if strings.EqualFold(strings.TrimSpace(language), "en") {
		return "en"
	}
	return "de"
}

func nextWeekday(now time.Time, target time.Weekday) time.Time {
	days := (int(target) - int(now.Weekday()) + 7) % 7
	if days == 0 {
		days = 7
	}
	return now.AddDate(0, 0, days)
}
