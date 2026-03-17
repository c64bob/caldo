package service

import "testing"

func TestSlugify(t *testing.T) {
	if got := slugify("  Mein Filter 2026 "); got != "mein-filter-2026" {
		t.Fatalf("unexpected slug: %s", got)
	}
}

func TestDedupe(t *testing.T) {
	values := dedupe([]string{"A", "a", "", " B "})
	if len(values) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(values))
	}
}
