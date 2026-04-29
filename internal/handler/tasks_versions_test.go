package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"caldo/internal/db"
	"caldo/internal/logging"
)

func TestTaskVersionsRouteReturnsVersions(t *testing.T) {
	t.Parallel()
	database, err := db.OpenSQLite(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	seedTaskVersionsRouteData(t, database)

	logger := logging.New(bytes.NewBuffer(nil), "production", "info")
	req := httptest.NewRequest(http.MethodGet, "/api/tasks/versions?ids=task-1,missing&ids=task-2", nil)
	req.Header.Set("X-Forwarded-User", "alice")
	rr := httptest.NewRecorder()
	NewRouter(logger, "X-Forwarded-User", testManifest(), true, []byte("12345678901234567890123456789012"), database, context.Background(), nil).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("unexpected status: got %d body=%q", rr.Code, rr.Body.String())
	}

	var payload struct {
		Tasks []struct {
			TaskID        string `json:"task_id"`
			ServerVersion int    `json:"server_version"`
		} `json:"tasks"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if len(payload.Tasks) != 2 {
		t.Fatalf("unexpected task count: got %d", len(payload.Tasks))
	}
}

func TestTaskVersionsRouteAcceptsLegacyTaskIDParam(t *testing.T) {
	t.Parallel()
	database, err := db.OpenSQLite(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	seedTaskVersionsRouteData(t, database)

	logger := logging.New(bytes.NewBuffer(nil), "production", "info")
	req := httptest.NewRequest(http.MethodGet, "/api/tasks/versions?task_id=task-1&task_id=task-2", nil)
	req.Header.Set("X-Forwarded-User", "alice")
	rr := httptest.NewRecorder()
	NewRouter(logger, "X-Forwarded-User", testManifest(), true, []byte("12345678901234567890123456789012"), database, context.Background(), nil).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("unexpected status: got %d body=%q", rr.Code, rr.Body.String())
	}
}

func TestTaskVersionsRouteRejectsMissingIDs(t *testing.T) {
	t.Parallel()
	database, err := db.OpenSQLite(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	logger := logging.New(bytes.NewBuffer(nil), "production", "info")
	req := httptest.NewRequest(http.MethodGet, "/api/tasks/versions", nil)
	req.Header.Set("X-Forwarded-User", "alice")
	rr := httptest.NewRecorder()
	NewRouter(logger, "X-Forwarded-User", testManifest(), true, []byte("12345678901234567890123456789012"), database, context.Background(), nil).ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status: got %d body=%q", rr.Code, rr.Body.String())
	}
}

func seedTaskVersionsRouteData(t *testing.T, database *db.Database) {
	t.Helper()
	if _, err := database.Conn.ExecContext(context.Background(), `
INSERT INTO projects (id, calendar_href, display_name, sync_strategy, created_at, updated_at)
VALUES ('project-1', '/cal/work/', 'Work', 'fullscan', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
INSERT INTO tasks (
 id, project_id, uid, href, etag, server_version, title, status, raw_vtodo, base_vtodo, project_name, sync_status, created_at, updated_at
) VALUES
('task-1', 'project-1', 'uid-1', '/cal/work/task-1.ics', '"e1"', 7, 'One', 'needs-action', 'BEGIN:VTODO\nUID:uid-1\nEND:VTODO', 'BEGIN:VTODO\nUID:uid-1\nEND:VTODO', 'Work', 'synced', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('task-2', 'project-1', 'uid-2', '/cal/work/task-2.ics', '"e2"', 3, 'Two', 'needs-action', 'BEGIN:VTODO\nUID:uid-2\nEND:VTODO', 'BEGIN:VTODO\nUID:uid-2\nEND:VTODO', 'Work', 'synced', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
`); err != nil {
		t.Fatalf("seed task versions route: %v", err)
	}
}
