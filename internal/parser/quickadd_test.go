package parser

import "testing"

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
