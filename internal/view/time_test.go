package view

import (
	"database/sql"
	"testing"
	"time"
)

func TestLocalDateTimeFromNullReturnsEmptyLabelWhenInvalid(t *testing.T) {
	t.Parallel()

	got := LocalDateTimeFromNull(sql.NullTime{}, "nie")
	if got.Text != "nie" || got.ISO != "" {
		t.Fatalf("expected empty label without ISO, got %#v", got)
	}
}

func TestLocalDateTimeFromTimeUsesUTCISOAndFallbackText(t *testing.T) {
	t.Parallel()

	loc := time.FixedZone("UTC+2", 2*60*60)
	got := LocalDateTimeFromTime(time.Date(2026, time.January, 2, 5, 4, 0, 0, loc), "nie")
	if got.ISO != "2026-01-02T03:04:00Z" {
		t.Fatalf("unexpected ISO timestamp: %q", got.ISO)
	}
	if got.Text != "02.01.2026 03:04" {
		t.Fatalf("unexpected fallback text: %q", got.Text)
	}
}
