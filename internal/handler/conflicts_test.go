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

func TestConflictsPageShowsOnlyUnresolved(t *testing.T) {
	database, err := db.OpenSQLite(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	seedConflictData(t, database)

	logger := logging.New(bytes.NewBuffer(nil), "production", "info")
	r := NewRouter(logger, "X-User", testManifest(), true, []byte("12345678901234567890123456789012"), database, context.Background(), nil)
	req := httptest.NewRequest(http.MethodGet, "/conflicts", nil)
	req.Header.Set("X-User", "u")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status: %d", rr.Code)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "open-1") || strings.Contains(body, "resolved-1") {
		t.Fatalf("unexpected body: %s", body)
	}
}

func seedConflictData(t *testing.T, database *db.Database) {
	t.Helper()
	_, err := database.Conn.ExecContext(context.Background(), `
INSERT INTO projects (id, calendar_href, display_name, sync_strategy, created_at, updated_at)
VALUES ('project-1','/p','Inbox','ctag',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP);
INSERT INTO tasks (id, project_id, uid, href, etag, title, status, raw_vtodo, sync_status, created_at, updated_at)
VALUES ('task-1','project-1','uid-1','/t','e','Task 1','needs-action','raw','conflict',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP);
INSERT INTO conflicts (id, task_id, project_id, conflict_type, created_at, base_vtodo, local_vtodo, remote_vtodo)
VALUES ('open-1','task-1','project-1','field_conflict',CURRENT_TIMESTAMP,'b','l','r');
INSERT INTO conflicts (id, task_id, project_id, conflict_type, created_at, resolved_at, resolution)
VALUES ('resolved-1','task-1','project-1','field_conflict',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,'local');
`)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
}
