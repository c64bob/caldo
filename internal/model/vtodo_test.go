package model

import "testing"

func TestParseVTODOFieldsExtractsNormalizedFields(t *testing.T) {
	t.Parallel()

	raw := "BEGIN:VCALENDAR\r\nBEGIN:VTODO\r\nUID:uid-1\r\nSUMMARY:Task title\r\nDESCRIPTION:Task description\r\nSTATUS:COMPLETED\r\nCOMPLETED:20260102T150405Z\r\nDUE:20260203T070809Z\r\nPRIORITY:5\r\nRRULE:FREQ=WEEKLY;BYDAY=MO\r\nRELATED-TO;RELTYPE=PARENT:parent-uid\r\nCATEGORIES:home,STARRED, errands\r\nEND:VTODO\r\nEND:VCALENDAR\r\n"

	parsed := ParseVTODOFields(raw)

	if parsed.UID != "uid-1" {
		t.Fatalf("unexpected uid: got %q", parsed.UID)
	}
	if parsed.Title != "Task title" {
		t.Fatalf("unexpected title: got %q", parsed.Title)
	}
	if parsed.Description != "Task description" {
		t.Fatalf("unexpected description: got %q", parsed.Description)
	}
	if parsed.Status != "completed" {
		t.Fatalf("unexpected status: got %q", parsed.Status)
	}
	if parsed.CompletedAt == nil || parsed.CompletedAt.Format("2006-01-02T15:04:05Z") != "2026-01-02T15:04:05Z" {
		t.Fatalf("unexpected completed_at: %#v", parsed.CompletedAt)
	}
	if parsed.DueAt == nil || parsed.DueAt.Format("2006-01-02T15:04:05Z") != "2026-02-03T07:08:09Z" {
		t.Fatalf("unexpected due_at: %#v", parsed.DueAt)
	}
	if parsed.Priority == nil || *parsed.Priority != 5 {
		t.Fatalf("unexpected priority: %#v", parsed.Priority)
	}
	if parsed.RRule != "FREQ=WEEKLY;BYDAY=MO" {
		t.Fatalf("unexpected rrule: got %q", parsed.RRule)
	}
	if parsed.ParentUID != "parent-uid" {
		t.Fatalf("unexpected parent uid: got %q", parsed.ParentUID)
	}
	if len(parsed.Categories) != 3 {
		t.Fatalf("unexpected categories length: got %d", len(parsed.Categories))
	}
}

func TestParseVTODOFieldsParsesDateDue(t *testing.T) {
	t.Parallel()

	parsed := ParseVTODOFields("BEGIN:VTODO\nUID:uid-2\nDUE;value=date:20260310\nEND:VTODO")
	if parsed.DueDate == nil || *parsed.DueDate != "2026-03-10" {
		t.Fatalf("unexpected due_date: %#v", parsed.DueDate)
	}
	if parsed.DueAt != nil {
		t.Fatalf("expected due_at nil, got %#v", parsed.DueAt)
	}
}

func TestParseVTODOFieldsParsesParentWithQuotedReltype(t *testing.T) {
	t.Parallel()

	parsed := ParseVTODOFields("BEGIN:VTODO\nUID:uid-3\nRELATED-TO;RELTYPE=\"PARENT\":parent-3\nEND:VTODO")
	if parsed.ParentUID != "parent-3" {
		t.Fatalf("unexpected parent uid: got %q", parsed.ParentUID)
	}
}
