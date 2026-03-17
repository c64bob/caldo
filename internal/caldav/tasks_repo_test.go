package caldav

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"caldo/internal/domain"
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

func TestTasksRepo_CreateTaskUsesIfNoneMatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Fatalf("expected PUT, got %s", r.Method)
		}
		if got := r.Header.Get("If-None-Match"); got != "*" {
			t.Fatalf("expected If-None-Match '*', got %q", got)
		}
		if !strings.HasSuffix(r.URL.Path, ".ics") {
			t.Fatalf("expected .ics target, got %s", r.URL.Path)
		}
		w.Header().Set("ETag", `"etag-new"`)
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	repo := NewTasksRepo(NewClient())
	created, err := repo.CreateTask(context.Background(), server.URL, "alice", "pw", Collection{ID: "tasks", Href: "/tasks/", SupportsVTODO: true}, domain.Task{UID: "abc", Summary: "hello"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if created.ETag != `"etag-new"` {
		t.Fatalf("expected etag from server, got %q", created.ETag)
	}
}

func TestTasksRepo_UpdateTaskReturnsConflictOn412(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("If-Match"); got != `"v1"` {
			t.Fatalf("expected If-Match header, got %q", got)
		}
		w.WriteHeader(http.StatusPreconditionFailed)
	}))
	defer server.Close()

	repo := NewTasksRepo(NewClient())
	_, err := repo.UpdateTask(context.Background(), server.URL, "alice", "pw", domain.Task{Href: "/tasks/abc.ics", ETag: `"v1"`, UID: "abc", Summary: "x"})
	if err == nil || err != ErrPreconditionFailed {
		t.Fatalf("expected ErrPreconditionFailed, got %v", err)
	}
}

func TestTasksRepo_DeleteTaskReturnsConflictOn412(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Fatalf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusPreconditionFailed)
	}))
	defer server.Close()

	repo := NewTasksRepo(NewClient())
	err := repo.DeleteTask(context.Background(), server.URL, "alice", "pw", "/tasks/abc.ics", `"v1"`)
	if err == nil || err != ErrPreconditionFailed {
		t.Fatalf("expected ErrPreconditionFailed, got %v", err)
	}
}

func TestTasksRepo_UpdateTaskFailsWhenETagMissing(t *testing.T) {
	repo := NewTasksRepo(NewClient())
	_, err := repo.UpdateTask(context.Background(), "https://caldav.example.com", "alice", "pw", domain.Task{Href: "/tasks/abc.ics", UID: "abc", Summary: "x"})
	if err == nil || err != ErrMissingETag {
		t.Fatalf("expected ErrMissingETag, got %v", err)
	}
}

func TestTasksRepo_DeleteTaskRejectsAbsoluteHref(t *testing.T) {
	repo := NewTasksRepo(NewClient())
	err := repo.DeleteTask(context.Background(), "https://caldav.example.com", "alice", "pw", "https://evil.example.com/pwn.ics", `"v1"`)
	if err == nil || err != ErrInvalidTaskHref {
		t.Fatalf("expected ErrInvalidTaskHref, got %v", err)
	}
}

func TestTasksRepo_DeleteTaskFailsWhenETagMissing(t *testing.T) {
	repo := NewTasksRepo(NewClient())
	err := repo.DeleteTask(context.Background(), "https://caldav.example.com", "alice", "pw", "/tasks/abc.ics", "")
	if err == nil || err != ErrMissingETag {
		t.Fatalf("expected ErrMissingETag, got %v", err)
	}
}

func TestTasksRepo_DeleteTaskRejectsHrefWithQuery(t *testing.T) {
	repo := NewTasksRepo(NewClient())
	err := repo.DeleteTask(context.Background(), "https://caldav.example.com", "alice", "pw", "/tasks/abc.ics?x=1", `"v1"`)
	if err == nil || err != ErrInvalidTaskHref {
		t.Fatalf("expected ErrInvalidTaskHref, got %v", err)
	}
}

func TestParseVTODO_ParsesParentAndGoal(t *testing.T) {
	raw := "BEGIN:VCALENDAR\nBEGIN:VTODO\nUID:task-3\nRELATED-TO:parent-1\nX-CALDO-GOAL:Long-term\nEND:VTODO\nEND:VCALENDAR"
	task := parseVTODO(raw)
	if task.ParentUID != "parent-1" {
		t.Fatalf("expected parent uid, got %q", task.ParentUID)
	}
	if task.Goal != "Long-term" {
		t.Fatalf("expected goal, got %q", task.Goal)
	}
}

func TestBuildVTODOCalendar_IncludesParentAndGoal(t *testing.T) {
	cal := buildVTODOCalendar(domain.Task{UID: "u-1", Summary: "x", ParentUID: "parent-1", Goal: "Lifetime"})
	if !strings.Contains(cal, "RELATED-TO:parent-1") {
		t.Fatalf("expected RELATED-TO in calendar: %s", cal)
	}
	if !strings.Contains(cal, "X-CALDO-GOAL:Lifetime") {
		t.Fatalf("expected X-CALDO-GOAL in calendar: %s", cal)
	}
}
