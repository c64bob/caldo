package handler

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"caldo/internal/logging"
)

func TestTaskFragmentRouteRendersCurrentTaskRow(t *testing.T) {
	t.Parallel()

	database := openDateViewRouteDB(t)
	seedDateViewRouteTasks(t, database)

	logger := logging.New(bytes.NewBuffer(nil), "production", "info")
	req := httptest.NewRequest(http.MethodGet, "/tasks/task-today-active", nil)
	req.Header.Set("X-Forwarded-User", "alice")
	rr := httptest.NewRecorder()

	NewRouter(logger, "X-Forwarded-User", testManifest(t), true, []byte("12345678901234567890123456789012"), database, context.Background(), nil).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("unexpected status: got %d body=%q", rr.Code, rr.Body.String())
	}
	body := rr.Body.String()
	for _, want := range []string{
		`data-task-id="task-today-active"`,
		`data-server-version="3"`,
		`name="expected_version" value="3"`,
		`Heute Aufgabe`,
		`Heute Beschreibung`,
		`Work`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("fragment missing %q in %s", want, body)
		}
	}
}

func TestTaskSubtasksFragmentRouteRendersAllDirectChildren(t *testing.T) {
	t.Parallel()

	database := openDateViewRouteDB(t)
	seedDateViewRouteTasks(t, database)
	if _, err := database.Conn.Exec(`
INSERT INTO tasks (
	id, project_id, uid, href, etag, server_version, title, description, status, raw_vtodo, base_vtodo,
	priority, label_names, project_name, sync_status, due_date, parent_id, created_at, updated_at
) VALUES
('task-today-child-completed','project-1','uid-today-child-completed','/calendars/work/task-today-child-completed.ics','"etag-child-completed"',2,'Erledigte Unteraufgabe','','completed','BEGIN:VTODO\nUID:uid-today-child-completed\nEND:VTODO','BEGIN:VTODO\nUID:uid-today-child-completed\nEND:VTODO',NULL,'','Work','synced',NULL,'task-today-active',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
('task-grandchild','project-1','uid-grandchild','/calendars/work/task-grandchild.ics','"etag-grandchild"',1,'Nicht direktes Kind','','needs-action','BEGIN:VTODO\nUID:uid-grandchild\nEND:VTODO','BEGIN:VTODO\nUID:uid-grandchild\nEND:VTODO',NULL,'','Work','synced',NULL,'task-today-child',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP);
`); err != nil {
		t.Fatalf("seed additional subtasks: %v", err)
	}

	logger := logging.New(bytes.NewBuffer(nil), "production", "info")
	req := httptest.NewRequest(http.MethodGet, "/tasks/task-today-active/subtasks", nil)
	req.Header.Set("X-Forwarded-User", "alice")
	rr := httptest.NewRecorder()

	NewRouter(logger, "X-Forwarded-User", testManifest(t), true, []byte("12345678901234567890123456789012"), database, context.Background(), nil).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("unexpected status: got %d body=%q", rr.Code, rr.Body.String())
	}
	body := rr.Body.String()
	for _, want := range []string{
		`data-task-id="task-today-child"`,
		`data-task-id="task-today-child-completed"`,
		`data-task-detail-subtask`,
		`hx-post="/tasks/task-today-child/complete"`,
		`hx-post="/tasks/task-today-child-completed/reopen"`,
		`aria-controls="task-detail-parent-detail-task-today-active-task-today-child"`,
		`Heute Unteraufgabe`,
		`Erledigte Unteraufgabe`,
		`Offen`,
		`Erledigt`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("subtask fragment missing %q in %s", want, body)
		}
	}
	if strings.Contains(body, "Nicht direktes Kind") {
		t.Fatalf("subtask fragment contains a non-direct child: %s", body)
	}
}

func TestTaskFragmentRouteReturnsNotFoundForMissingTask(t *testing.T) {
	t.Parallel()

	database := openDateViewRouteDB(t)
	logger := logging.New(bytes.NewBuffer(nil), "production", "info")
	req := httptest.NewRequest(http.MethodGet, "/tasks/missing", nil)
	req.Header.Set("X-Forwarded-User", "alice")
	rr := httptest.NewRecorder()

	NewRouter(logger, "X-Forwarded-User", testManifest(t), true, []byte("12345678901234567890123456789012"), database, context.Background(), nil).ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("unexpected status: got %d body=%q", rr.Code, rr.Body.String())
	}
}
