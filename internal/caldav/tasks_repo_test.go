package caldav

import (
	"testing"
	"time"
)

func TestParseVTODO_UnfoldsFoldedLines(t *testing.T) {
	raw := "BEGIN:VCALENDAR\r\nBEGIN:VTODO\r\nUID:task-1\r\nSUMMARY:This is a very long\r\n description line\r\nDESCRIPTION:Line one\r\n\tand line two\r\nEND:VTODO\r\nEND:VCALENDAR\r\n"

	task := parseVTODO(raw)

	if task.Summary != "This is a very longdescription line" {
		t.Fatalf("unexpected summary: %q", task.Summary)
	}
	if task.Description != "Line oneand line two" {
		t.Fatalf("unexpected description: %q", task.Description)
	}
}

func TestParseVTODO_PreservesTZIDDateTime(t *testing.T) {
	raw := "BEGIN:VCALENDAR\nBEGIN:VTODO\nUID:task-2\nDUE;TZID=Europe/Berlin:20260316T090000\nEND:VTODO\nEND:VCALENDAR"

	task := parseVTODO(raw)
	if task.Due == nil {
		t.Fatal("expected due date to be parsed")
	}
	if got := task.Due.Location().String(); got != "Europe/Berlin" {
		t.Fatalf("expected Europe/Berlin location, got %s", got)
	}
	if task.DueKind != "datetime" {
		t.Fatalf("expected datetime kind, got %s", task.DueKind)
	}
	if task.Due.Hour() != 9 {
		t.Fatalf("expected hour 9, got %d", task.Due.Hour())
	}
}

func TestParseICalTime_FloatingDateTimeUsesLocal(t *testing.T) {
	orig := time.Local
	time.Local = time.FixedZone("TestLocal", 2*60*60)
	t.Cleanup(func() { time.Local = orig })

	parsed, kind, ok := parseICalTime("20260316T090000", "")
	if !ok {
		t.Fatal("expected floating datetime to parse")
	}
	if kind != "datetime" {
		t.Fatalf("expected datetime kind, got %s", kind)
	}
	if parsed.Location().String() != "TestLocal" {
		t.Fatalf("expected TestLocal, got %s", parsed.Location())
	}
}
