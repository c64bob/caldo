package parser

import "testing"

func TestParseQuickAddTrimsTitle(t *testing.T) {
	draft := ParseQuickAdd("  test aufgabe  ")
	if draft.Title != "test aufgabe" {
		t.Fatalf("unexpected title: %q", draft.Title)
	}
}
