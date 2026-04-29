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
	oldNow := nowFunc
	nowFunc = func() time.Time { return time.Date(2026, time.March, 4, 10, 0, 0, 0, time.UTC) }
	t.Cleanup(func() { nowFunc = oldNow })

	tests := []struct {
		in      string
		wantDue string
		wantRem string
	}{
		{"Task heute", "2026-03-04", "Task"},
		{"Task tomorrow", "2026-03-05", "Task"},
		{"Task übermorgen", "2026-03-06", "Task"},
		{"Task in 3 Tagen", "2026-03-07", "Task"},
		{"Task in 3 days", "2026-03-07", "Task"},
		{"Task nächsten Montag", "2026-03-09", "Task"},
		{"Task next monday", "2026-03-09", "Task"},
		{"Task Mittwoch", "2026-03-11", "Task"},
		{"Task friday", "2026-03-06", "Task"},
	}

	for _, tc := range tests {
		draft := ParseQuickAdd(tc.in)
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
