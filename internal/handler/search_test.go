package handler

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"caldo/internal/db"
	"caldo/internal/logging"
)

func TestSearchRouteReturnsActiveTasksOnly(t *testing.T) {
	t.Parallel()

	database, err := db.OpenSQLite(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() {
		if err := database.Close(); err != nil {
			t.Fatalf("close sqlite: %v", err)
		}
	})

	seedSearchRouteProjectAndTasks(t, database)

	logger := logging.New(bytes.NewBuffer(nil), "production", "info")
	request := httptest.NewRequest(http.MethodGet, "/search?q=rechnung", nil)
	request.Header.Set("X-Forwarded-User", "alice")
	responseRecorder := httptest.NewRecorder()

	NewRouter(logger, "X-Forwarded-User", testManifest(), true, []byte("12345678901234567890123456789012"), database, context.Background(), nil).ServeHTTP(responseRecorder, request)

	if responseRecorder.Code != http.StatusOK {
		t.Fatalf("unexpected status code: got %d want %d", responseRecorder.Code, http.StatusOK)
	}

	body := responseRecorder.Body.String()
	if !strings.Contains(body, "Überweisung Rechnung") {
		t.Fatalf("response body missing active task title: %q", body)
	}
	if strings.Contains(body, "Überfällige Rechnung") {
		t.Fatalf("response body unexpectedly contains completed task title: %q", body)
	}
	if !strings.Contains(body, "Globale Suche") {
		t.Fatalf("response body missing search heading")
	}
}

func seedSearchRouteProjectAndTasks(t *testing.T, database *db.Database) {
	t.Helper()

	if _, err := database.Conn.Exec(`
INSERT INTO projects (
    id, calendar_href, display_name, sync_strategy, server_version, created_at, updated_at
) VALUES (
    'project-1', '/calendars/work', 'Work', 'fullscan', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
);

INSERT INTO tasks (
    id, project_id, uid, href, etag, server_version, title, description, status, raw_vtodo, base_vtodo,
    label_names, project_name, sync_status, created_at, updated_at
) VALUES
(
    'task-active', 'project-1', 'uid-active', '/calendars/work/task-active.ics', '"etag-active"', 1,
    'Überweisung Rechnung', 'Prüfen', 'needs-action', 'BEGIN:VTODO\nUID:uid-active\nEND:VTODO',
    'BEGIN:VTODO\nUID:uid-active\nEND:VTODO', 'Büro dringend', 'Finanzen', 'synced', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
),
(
    'task-completed', 'project-1', 'uid-completed', '/calendars/work/task-completed.ics', '"etag-completed"', 1,
    'Überfällige Rechnung', 'Archiv', 'completed', 'BEGIN:VTODO\nUID:uid-completed\nEND:VTODO',
    'BEGIN:VTODO\nUID:uid-completed\nEND:VTODO', 'Büro erledigt', 'Finanzen', 'synced', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
);
`); err != nil {
		t.Fatalf("seed search route data: %v", err)
	}
}
