package model

import (
	"strings"
	"testing"
	"time"
)

func TestBuildTaskVTODO(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.April, 28, 12, 34, 56, 0, time.UTC)
	raw := BuildTaskVTODO("uid-1", `Task, A; B\\C`, now)

	for _, want := range []string{
		"BEGIN:VCALENDAR",
		"BEGIN:VTODO",
		"UID:uid-1",
		"DTSTAMP:20260428T123456Z",
		`SUMMARY:Task\\, A\\; B\\\\C`,
		"STATUS:NEEDS-ACTION",
		"END:VTODO",
		"END:VCALENDAR",
	} {
		if !strings.Contains(raw, want) {
			t.Fatalf("raw vtodo missing %q in %q", want, raw)
		}
	}

	if !strings.Contains(raw, "\r\n") {
		t.Fatalf("expected CRLF separators, got %q", raw)
	}
}
