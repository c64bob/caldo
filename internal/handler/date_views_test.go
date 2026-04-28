package handler

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"caldo/internal/db"
)

func TestTodayRouteShowsTodayAndOverdueTasks(t *testing.T) {
	t.Parallel()

	database := openDateViewRouteDB(t)
	seedDateViewRouteTasks(t, database)

	request := httptest.NewRequest(http.MethodGet, "/today", nil)
	request.Header.Set("X-Forwarded-User", "alice")
	responseRecorder := httptest.NewRecorder()

	Today(dateViewDependencies{database: database, now: fixedNow}).ServeHTTP(responseRecorder, request)

	if responseRecorder.Code != http.StatusOK {
		t.Fatalf("unexpected status code: got %d want %d", responseRecorder.Code, http.StatusOK)
	}

	body := responseRecorder.Body.String()
	if !strings.Contains(body, "Überfällige Aufgabe") {
		t.Fatalf("response body missing overdue task")
	}
	if !strings.Contains(body, "Heute Aufgabe") {
		t.Fatalf("response body missing today task")
	}
	if strings.Contains(body, "Bald Aufgabe") {
		t.Fatalf("response body unexpectedly contains upcoming task")
	}
	if strings.Contains(body, "Ohne Fälligkeit") {
		t.Fatalf("response body unexpectedly contains task without due date")
	}
}

func TestUpcomingRouteShowsTasksInsideConfiguredWindow(t *testing.T) {
	t.Parallel()

	database := openDateViewRouteDB(t)
	seedDateViewRouteTasks(t, database)

	if _, err := database.Conn.Exec(`UPDATE settings SET upcoming_days = 3 WHERE id = 'default';`); err != nil {
		t.Fatalf("update upcoming days: %v", err)
	}

	request := httptest.NewRequest(http.MethodGet, "/upcoming", nil)
	request.Header.Set("X-Forwarded-User", "alice")
	responseRecorder := httptest.NewRecorder()

	Upcoming(dateViewDependencies{database: database, now: fixedNow}).ServeHTTP(responseRecorder, request)

	if responseRecorder.Code != http.StatusOK {
		t.Fatalf("unexpected status code: got %d want %d", responseRecorder.Code, http.StatusOK)
	}

	body := responseRecorder.Body.String()
	if !strings.Contains(body, "Bald Aufgabe") {
		t.Fatalf("response body missing in-range upcoming task")
	}
	if strings.Contains(body, "In 7 Tagen") {
		t.Fatalf("response body unexpectedly contains out-of-window task")
	}
}

func TestOverdueRouteRespectsShowCompletedSetting(t *testing.T) {
	t.Parallel()

	database := openDateViewRouteDB(t)
	seedDateViewRouteTasks(t, database)

	request := httptest.NewRequest(http.MethodGet, "/overdue", nil)
	responseRecorder := httptest.NewRecorder()

	Overdue(dateViewDependencies{database: database, now: fixedNow}).ServeHTTP(responseRecorder, request)

	if responseRecorder.Code != http.StatusOK {
		t.Fatalf("unexpected status code: got %d want %d", responseRecorder.Code, http.StatusOK)
	}
	if strings.Contains(responseRecorder.Body.String(), "Überfällig erledigt") {
		t.Fatal("completed task should be hidden by default")
	}

	if _, err := database.Conn.Exec(`UPDATE settings SET show_completed = TRUE WHERE id = 'default';`); err != nil {
		t.Fatalf("update show_completed: %v", err)
	}

	responseRecorder = httptest.NewRecorder()
	Overdue(dateViewDependencies{database: database, now: fixedNow}).ServeHTTP(responseRecorder, request)
	if !strings.Contains(responseRecorder.Body.String(), "Überfällig erledigt") {
		t.Fatal("completed task should be visible when show_completed is true")
	}
}

func fixedNow() time.Time {
	return time.Date(2026, 4, 28, 10, 0, 0, 0, time.UTC)
}

func openDateViewRouteDB(t *testing.T) *db.Database {
	t.Helper()

	database, err := db.OpenSQLite(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() {
		if err := database.Close(); err != nil {
			t.Fatalf("close sqlite: %v", err)
		}
	})
	return database
}

func seedDateViewRouteTasks(t *testing.T, database *db.Database) {
	t.Helper()

	if _, err := database.Conn.Exec(`
INSERT INTO projects (
	id, calendar_href, display_name, sync_strategy, server_version, created_at, updated_at
) VALUES (
	'project-1', '/calendars/work', 'Work', 'fullscan', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
);

INSERT INTO tasks (
	id, project_id, uid, href, etag, server_version, title, description, status, raw_vtodo, base_vtodo,
	label_names, project_name, sync_status, due_date, due_at, created_at, updated_at
) VALUES
('task-overdue-active','project-1','uid-overdue-active','/calendars/work/task-overdue-active.ics','"etag-1"',1,'Überfällige Aufgabe','','needs-action','BEGIN:VTODO\nUID:uid-overdue-active\nEND:VTODO','BEGIN:VTODO\nUID:uid-overdue-active\nEND:VTODO','','Work','synced','2026-04-27',NULL,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
('task-overdue-completed','project-1','uid-overdue-completed','/calendars/work/task-overdue-completed.ics','"etag-2"',1,'Überfällig erledigt','','completed','BEGIN:VTODO\nUID:uid-overdue-completed\nEND:VTODO','BEGIN:VTODO\nUID:uid-overdue-completed\nEND:VTODO','','Work','synced','2026-04-26',NULL,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
('task-today-active','project-1','uid-today-active','/calendars/work/task-today-active.ics','"etag-3"',1,'Heute Aufgabe','','needs-action','BEGIN:VTODO\nUID:uid-today-active\nEND:VTODO','BEGIN:VTODO\nUID:uid-today-active\nEND:VTODO','','Work','synced','2026-04-28',NULL,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
('task-upcoming-in-range','project-1','uid-upcoming-in-range','/calendars/work/task-upcoming-in-range.ics','"etag-4"',1,'Bald Aufgabe','','needs-action','BEGIN:VTODO\nUID:uid-upcoming-in-range\nEND:VTODO','BEGIN:VTODO\nUID:uid-upcoming-in-range\nEND:VTODO','','Work','synced','2026-05-01',NULL,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
('task-upcoming-day7','project-1','uid-upcoming-day7','/calendars/work/task-upcoming-day7.ics','"etag-5"',1,'In 7 Tagen','','needs-action','BEGIN:VTODO\nUID:uid-upcoming-day7\nEND:VTODO','BEGIN:VTODO\nUID:uid-upcoming-day7\nEND:VTODO','','Work','synced','2026-05-05',NULL,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
('task-without-due','project-1','uid-without-due','/calendars/work/task-without-due.ics','"etag-6"',1,'Ohne Fälligkeit','','needs-action','BEGIN:VTODO\nUID:uid-without-due\nEND:VTODO','BEGIN:VTODO\nUID:uid-without-due\nEND:VTODO','','Work','synced',NULL,NULL,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP);
`); err != nil {
		t.Fatalf("seed date view tasks: %v", err)
	}
}
