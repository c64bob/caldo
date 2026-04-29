package handler

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"caldo/internal/db"
	"caldo/internal/logging"
	"github.com/go-chi/chi/v5"
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

func TestResolveConflictLocalSuccess(t *testing.T) {
	t.Parallel()
	database, err := db.OpenSQLite(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	seedConflictData(t, database)
	key := bytes.Repeat([]byte{0x33}, 32)
	if err := database.SaveCalDAVCredentials(context.Background(), key, db.CalDAVCredentials{URL: "https://dav.example", Username: "a", Password: "b"}); err != nil {
		t.Fatal(err)
	}

	h := ResolveConflict(taskUpdateDependencies{database: database, encryptionKey: key, todos: &stubTaskUpdateTodoClient{updateETag: `"etag-new"`}})
	form := strings.NewReader("resolution=local")
	req := httptest.NewRequest(http.MethodPost, "/conflicts/open-1/resolve", form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("conflictID", "open-1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var resolution string
	if err := database.Conn.QueryRow(`SELECT resolution FROM conflicts WHERE id='open-1'`).Scan(&resolution); err != nil {
		t.Fatal(err)
	}
	if resolution != "local" {
		t.Fatalf("resolution=%q", resolution)
	}
}

func TestResolveConflictKeepsUnresolvedOnWriteFailure(t *testing.T) {
	t.Parallel()
	database, err := db.OpenSQLite(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	seedConflictData(t, database)
	key := bytes.Repeat([]byte{0x44}, 32)
	if err := database.SaveCalDAVCredentials(context.Background(), key, db.CalDAVCredentials{URL: "https://dav.example", Username: "a", Password: "b"}); err != nil {
		t.Fatal(err)
	}

	h := ResolveConflict(taskUpdateDependencies{database: database, encryptionKey: key, todos: &stubTaskUpdateTodoClient{updateErr: errors.New("boom")}})
	req := httptest.NewRequest(http.MethodPost, "/conflicts/open-1/resolve", strings.NewReader("resolution=remote"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("conflictID", "open-1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadGateway {
		t.Fatalf("status=%d", rr.Code)
	}
	var count int
	if err := database.Conn.QueryRow(`SELECT COUNT(*) FROM conflicts WHERE id='open-1' AND resolved_at IS NULL`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("count=%d", count)
	}
}
