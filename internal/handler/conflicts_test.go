package handler

import (
	"bytes"
	"context"
	"database/sql"
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

func TestResolveConflictManualOmitsMissingCoreFields(t *testing.T) {
	t.Parallel()
	database, err := db.OpenSQLite(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	seedConflictData(t, database)
	key := bytes.Repeat([]byte{0x55}, 32)
	if err := database.SaveCalDAVCredentials(context.Background(), key, db.CalDAVCredentials{URL: "https://dav.example", Username: "a", Password: "b"}); err != nil {
		t.Fatal(err)
	}
	remote := "BEGIN:VTODO\r\nSUMMARY:Keep me\r\nDESCRIPTION:Keep desc\r\nSTATUS:NEEDS-ACTION\r\nDUE;VALUE=DATE:20260501\r\nEND:VTODO\r\n"
	if _, err := database.Conn.Exec(`UPDATE conflicts SET remote_vtodo=? WHERE id='open-1'`, remote); err != nil {
		t.Fatal(err)
	}
	h := ResolveConflict(taskUpdateDependencies{database: database, encryptionKey: key, todos: &stubTaskUpdateTodoClient{updateETag: `"etag-new"`}})
	form := strings.NewReader("resolution=manual&due_date=2026-05-20")
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
	var resolved string
	if err := database.Conn.QueryRow(`SELECT resolved_vtodo FROM conflicts WHERE id='open-1'`).Scan(&resolved); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(resolved, "SUMMARY:Keep me") || !strings.Contains(resolved, "DESCRIPTION:Keep desc") || !strings.Contains(resolved, "STATUS:NEEDS-ACTION") {
		t.Fatalf("missing preserved core fields in resolved payload: %s", resolved)
	}
	if !strings.Contains(resolved, "DUE;VALUE=DATE:20260520") {
		t.Fatalf("expected due date patch in resolved payload: %s", resolved)
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

func TestResolveConflictSplitCreatesSecondTaskAndMarksResolved(t *testing.T) {
	t.Parallel()
	database, err := db.OpenSQLite(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	seedConflictData(t, database)
	key := bytes.Repeat([]byte{0x45}, 32)
	if err := database.SaveCalDAVCredentials(context.Background(), key, db.CalDAVCredentials{URL: "https://dav.example", Username: "a", Password: "b"}); err != nil {
		t.Fatal(err)
	}
	remote := "BEGIN:VCALENDAR\r\nBEGIN:VTODO\r\nUID:uid-remote\r\nSUMMARY:Remote version\r\nSTATUS:NEEDS-ACTION\r\nRELATED-TO;RELTYPE=PARENT:uid-parent\r\nEND:VTODO\r\nEND:VCALENDAR\r\n"
	if _, err := database.Conn.Exec(`UPDATE conflicts SET remote_vtodo=? WHERE id='open-1'`, remote); err != nil {
		t.Fatal(err)
	}

	todos := &stubTaskUpdateTodoClient{createETag: `"etag-split"`}
	h := ResolveConflict(taskUpdateDependencies{database: database, encryptionKey: key, todos: todos})
	req := httptest.NewRequest(http.MethodPost, "/conflicts/open-1/resolve", strings.NewReader("resolution=split"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("conflictID", "open-1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if todos.updateCalls != 0 || todos.createCalls != 1 {
		t.Fatalf("unexpected caldav calls: update=%d create=%d", todos.updateCalls, todos.createCalls)
	}
	if strings.Contains(strings.ToUpper(todos.lastRawVTODO), "RELATED-TO;RELTYPE=PARENT") {
		t.Fatalf("split payload must not include parent link: %s", todos.lastRawVTODO)
	}
	if !strings.Contains(todos.lastRawVTODO, "UID:"+splitConflictUID("open-1")) {
		t.Fatalf("split payload uid not rewritten deterministically: %s", todos.lastRawVTODO)
	}
	if todos.lastHref != "/"+splitConflictUID("open-1")+".ics" {
		t.Fatalf("split href=%q", todos.lastHref)
	}
	var resolution string
	if err := database.Conn.QueryRow(`SELECT resolution FROM conflicts WHERE id='open-1'`).Scan(&resolution); err != nil {
		t.Fatal(err)
	}
	if resolution != "split" {
		t.Fatalf("resolution=%q", resolution)
	}
	var taskCount int
	if err := database.Conn.QueryRow(`SELECT COUNT(*) FROM tasks WHERE project_id='project-1'`).Scan(&taskCount); err != nil {
		t.Fatal(err)
	}
	if taskCount != 2 {
		t.Fatalf("taskCount=%d", taskCount)
	}
	var localUID, splitUID string
	if err := database.Conn.QueryRow(`SELECT uid FROM tasks WHERE id='task-1'`).Scan(&localUID); err != nil {
		t.Fatal(err)
	}
	if err := database.Conn.QueryRow(`SELECT uid FROM tasks WHERE id!='task-1'`).Scan(&splitUID); err != nil {
		t.Fatal(err)
	}
	if localUID != "uid-1" || splitUID == localUID {
		t.Fatalf("uids local=%q split=%q", localUID, splitUID)
	}
	var splitParentID sql.NullString
	if err := database.Conn.QueryRow(`SELECT parent_id FROM tasks WHERE id!='task-1'`).Scan(&splitParentID); err != nil {
		t.Fatal(err)
	}
	if splitParentID.Valid {
		t.Fatalf("expected no parent link, got %q", splitParentID.String)
	}
}
