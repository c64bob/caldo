package parser

import (
	"testing"
	"time"
)

func TestParseQuickAddTrimsTitle(t *testing.T) {
	draft := ParseQuickAdd("  test aufgabe  ")
	if draft.Title != "test aufgabe" {
		t.Fatalf("unexpected title: %q", draft.Title)
	}
}

func TestParseQuickAddParsesProjectLabelAndPriorityTokens(t *testing.T) {
	draft := ParseQuickAdd("Review #Work @urgent @backend !high")

	if draft.Title != "Review" {
		t.Fatalf("unexpected title: %q", draft.Title)
	}
	if draft.Project != "Work" {
		t.Fatalf("unexpected project: %q", draft.Project)
	}
	if got := len(draft.Labels); got != 2 {
		t.Fatalf("unexpected label count: %d", got)
	}
	if draft.Labels[0] != "urgent" || draft.Labels[1] != "backend" {
		t.Fatalf("unexpected labels: %#v", draft.Labels)
	}
	if draft.Priority != "high" {
		t.Fatalf("unexpected priority: %q", draft.Priority)
	}
}

func TestParseQuickAddParsesNumericPriorityTokens(t *testing.T) {
	tests := map[string]string{"!1": "high", "!2": "medium", "!3": "low"}
	for input, want := range tests {
		draft := ParseQuickAdd("Task " + input)
		if draft.Priority != want {
			t.Fatalf("priority for %s: got %q want %q", input, draft.Priority, want)
		}
	}
}

func TestParseQuickAddNaturalDueDateEnglishGerman(t *testing.T) {
	now := time.Date(2026, time.March, 4, 10, 0, 0, 0, time.UTC)

	tests := []struct {
		in      string
		lang    string
		wantDue string
		wantRem string
	}{
		{"Task heute", "de", "2026-03-04", "Task"},
		{"Task tomorrow", "en", "2026-03-05", "Task"},
		{"Task übermorgen", "de", "2026-03-06", "Task"},
		{"Task in 3 Tagen", "de", "2026-03-07", "Task"},
		{"Task in 3 days", "en", "2026-03-07", "Task"},
		{"Task nächsten Montag", "de", "2026-03-09", "Task"},
		{"Task next monday", "en", "2026-03-09", "Task"},
		{"Task Mittwoch", "de", "2026-03-11", "Task"},
		{"Task friday", "en", "2026-03-06", "Task"},
	}

	for _, tc := range tests {
		draft := parseQuickAddAt(tc.in, now, tc.lang)
		if draft.Due != tc.wantDue {
			t.Fatalf("%q due: got %q want %q", tc.in, draft.Due, tc.wantDue)
		}
		if draft.Title != tc.wantRem {
			t.Fatalf("%q title: got %q want %q", tc.in, draft.Title, tc.wantRem)
		}
	}
}

func TestParseQuickAddUnknownTokensRemainInTitle(t *testing.T) {
	draft := ParseQuickAdd("Task maybe nextmon")
	if draft.Due != "" {
		t.Fatalf("unexpected due: %q", draft.Due)
	}
	if draft.Title != "Task maybe nextmon" {
		t.Fatalf("unexpected title: %q", draft.Title)
	}
}

func TestParseQuickAddRecurrencePatterns(t *testing.T) {
	tests := []struct {
		in        string
		lang      string
		wantRRule string
		wantTitle string
	}{
		{"Task jeden Montag", "de", "FREQ=WEEKLY;BYDAY=MO", "Task"},
		{"Task every monday", "en", "FREQ=WEEKLY;BYDAY=MO", "Task"},
		{"Task täglich", "de", "FREQ=DAILY", "Task"},
		{"Task daily", "en", "FREQ=DAILY", "Task"},
		{"Task wöchentlich", "de", "FREQ=WEEKLY", "Task"},
		{"Task weekly", "en", "FREQ=WEEKLY", "Task"},
		{"Task monatlich", "de", "FREQ=MONTHLY", "Task"},
		{"Task monthly", "en", "FREQ=MONTHLY", "Task"},
		{"Task jährlich", "de", "FREQ=YEARLY", "Task"},
		{"Task yearly", "en", "FREQ=YEARLY", "Task"},
		{"Task werktags", "de", "FREQ=WEEKLY;BYDAY=MO,TU,WE,TH,FR", "Task"},
		{"Task weekdays", "en", "FREQ=WEEKLY;BYDAY=MO,TU,WE,TH,FR", "Task"},
		{"Task alle 3 tage", "de", "FREQ=DAILY;INTERVAL=3", "Task"},
		{"Task alle 2 wochen", "de", "FREQ=WEEKLY;INTERVAL=2", "Task"},
		{"Task alle 4 monate", "de", "FREQ=MONTHLY;INTERVAL=4", "Task"},
	}

	for _, tc := range tests {
		draft := ParseQuickAddWithLanguage(tc.in, tc.lang)
		if draft.Recurrence != tc.wantRRule {
			t.Fatalf("%q recurrence: got %q want %q", tc.in, draft.Recurrence, tc.wantRRule)
		}
		if draft.Title != tc.wantTitle {
			t.Fatalf("%q title: got %q want %q", tc.in, draft.Title, tc.wantTitle)
		}
	}
}

func TestParseQuickAddUnsupportedComplexRecurrenceStaysNonRecurrence(t *testing.T) {
	now := time.Date(2026, time.March, 4, 10, 0, 0, 0, time.UTC)
	draft := parseQuickAddAt("Task every second monday", now, "en")
	if draft.Recurrence != "" {
		t.Fatalf("expected no recurrence, got %q", draft.Recurrence)
	}
	if draft.Title != "Task every second" {
		t.Fatalf("unexpected title: %q", draft.Title)
	}
}

func TestParseQuickAddLanguageSwitchesNaturalParsing(t *testing.T) {
	now := time.Date(2026, time.March, 4, 10, 0, 0, 0, time.UTC)

	deDraft := parseQuickAddAt("Task morgen", now, "de")
	if deDraft.Due != "2026-03-05" {
		t.Fatalf("de due: got %q want %q", deDraft.Due, "2026-03-05")
	}

	enDraft := parseQuickAddAt("Task tomorrow", now, "en")
	if enDraft.Due != "2026-03-05" {
		t.Fatalf("en due: got %q want %q", enDraft.Due, "2026-03-05")
	}
	if enDraft.Title != "Task" {
		t.Fatalf("en title: got %q want %q", enDraft.Title, "Task")
	}
}
