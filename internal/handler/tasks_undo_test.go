package handler

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"caldo/internal/caldav"
	"caldo/internal/db"
)

func TestTaskUndoSuccessDeletesSnapshot(t *testing.T) {
	t.Parallel()
	database := openSQLiteForTaskUndoHandlerTest(t)
	seedTaskUndoHandlerData(t, database)
	key := bytes.Repeat([]byte{0x6a}, 32)
	_ = database.SaveCalDAVCredentials(context.Background(), key, db.CalDAVCredentials{URL: "https://dav.example", Username: "alice", Password: "secret"})

	stub := &stubTaskUpdateTodoClient{updateETag: `"etag-3"`}
	h := TaskUndo(taskUpdateDependencies{database: database, encryptionKey: key, todos: stub})
	req := httptest.NewRequest(http.MethodPost, "/tasks/undo", nil)
	req.Header.Set("X-Tab-ID", "tab-1")
	req = requestWithSession(req, "session-1")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%q", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "Rückgängig ausgeführt.") {
		t.Fatalf("response missing undo result fragment: %q", rr.Body.String())
	}
	if stub.updateCalls != 1 || stub.lastUpdateETag != `"etag-2"` {
		t.Fatalf("undo should write once with current etag, calls=%d etag=%q", stub.updateCalls, stub.lastUpdateETag)
	}
	assertSingleIntResult(t, database, `SELECT COUNT(*) FROM undo_snapshots WHERE session_id='session-1' AND tab_id='tab-1';`, 0)
	assertSingleTextResult(t, database, `SELECT etag FROM tasks WHERE id='task-1';`, `"etag-3"`)
}

func TestTaskUndoPreconditionFailedMarksConflictKeepsSnapshot(t *testing.T) {
	t.Parallel()
	database := openSQLiteForTaskUndoHandlerTest(t)
	seedTaskUndoHandlerData(t, database)
	key := bytes.Repeat([]byte{0x6b}, 32)
	_ = database.SaveCalDAVCredentials(context.Background(), key, db.CalDAVCredentials{URL: "https://dav.example", Username: "alice", Password: "secret"})

	stub := &stubTaskUpdateTodoClient{updateErr: caldav.ErrPreconditionFailed}
	h := TaskUndo(taskUpdateDependencies{database: database, encryptionKey: key, todos: stub})
	req := httptest.NewRequest(http.MethodPost, "/tasks/undo", nil)
	req.Header.Set("X-Tab-ID", "tab-1")
	req = requestWithSession(req, "session-1")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusConflict {
		t.Fatalf("status=%d body=%q", rr.Code, rr.Body.String())
	}
	if stub.updateCalls != 1 || stub.lastUpdateETag != `"etag-2"` {
		t.Fatalf("undo conflict should be detected by caldav write with current etag, calls=%d etag=%q", stub.updateCalls, stub.lastUpdateETag)
	}
	assertSingleTextResult(t, database, `SELECT sync_status FROM tasks WHERE id='task-1';`, "conflict")
	assertSingleIntResult(t, database, `SELECT COUNT(*) FROM undo_snapshots WHERE session_id='session-1' AND tab_id='tab-1';`, 1)
}

func TestTaskUndoDeletedTaskRecreatesResourceAndTaskRow(t *testing.T) {
	t.Parallel()
	database := openSQLiteForTaskUndoHandlerTest(t)
	seedTaskUndoHandlerData(t, database)
	key := bytes.Repeat([]byte{0x6c}, 32)
	_ = database.SaveCalDAVCredentials(context.Background(), key, db.CalDAVCredentials{URL: "https://dav.example", Username: "alice", Password: "secret"})

	h := TaskUndo(taskUpdateDependencies{database: database, encryptionKey: key, todos: &stubTaskUpdateTodoClient{createETag: `"etag-del"`}})
	req := httptest.NewRequest(http.MethodPost, "/tasks/undo", nil)
	req.Header.Set("X-Tab-ID", "tab-del")
	req = requestWithSession(req, "session-del")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%q", rr.Code, rr.Body.String())
	}
	assertSingleIntResult(t, database, `SELECT COUNT(*) FROM undo_snapshots WHERE session_id='session-del' AND tab_id='tab-del';`, 0)
	assertSingleIntResult(t, database, `SELECT COUNT(*) FROM tasks WHERE uid='uid-del' AND sync_status='synced' AND etag='"etag-del"';`, 1)
}

func TestTaskUndoStatusRendersValidSnapshot(t *testing.T) {
	t.Parallel()
	database := openSQLiteForTaskUndoHandlerTest(t)
	seedTaskUndoHandlerData(t, database)

	h := TaskUndoStatus(database)
	req := httptest.NewRequest(http.MethodGet, "/tasks/undo/status", nil)
	req.Header.Set("X-Tab-ID", "tab-1")
	req = requestWithSession(req, "session-1")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%q", rr.Code, rr.Body.String())
	}
	for _, want := range []string{
		`data-undo-toast`,
		`data-undo-expires-at=`,
		`Letzte Änderung gespeichert.`,
		`Rückgängig ist kurz verfügbar.`,
		`data-undo-action`,
		`Rückgängig`,
	} {
		if !strings.Contains(rr.Body.String(), want) {
			t.Fatalf("response missing %q: %q", want, rr.Body.String())
		}
	}
}

func TestTaskUndoStatusHidesMissingExpiredAndHeaderlessSnapshots(t *testing.T) {
	t.Parallel()
	database := openSQLiteForTaskUndoHandlerTest(t)
	seedTaskUndoHandlerData(t, database)

	if _, err := database.Conn.ExecContext(context.Background(), `
INSERT INTO undo_snapshots (id, session_id, tab_id, task_id, action_type, snapshot_vtodo, snapshot_fields, etag_at_snapshot, created_at, expires_at)
VALUES ('undo-exp','session-exp','tab-exp','task-1','task_updated','BEGIN:VTODO\nUID:uid-1\nEND:VTODO',json_object('title','before'),'"etag-1"',CURRENT_TIMESTAMP,DATETIME(CURRENT_TIMESTAMP,'-1 minutes'));
`); err != nil {
		t.Fatalf("seed expired undo snapshot: %v", err)
	}

	h := TaskUndoStatus(database)
	for _, tt := range []struct {
		name      string
		sessionID string
		tabID     string
	}{
		{name: "missing", sessionID: "session-1", tabID: "missing-tab"},
		{name: "expired", sessionID: "session-exp", tabID: "tab-exp"},
		{name: "no tab", sessionID: "session-1"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/tasks/undo/status", nil)
			req = requestWithSession(req, tt.sessionID)
			if tt.tabID != "" {
				req.Header.Set("X-Tab-ID", tt.tabID)
			}
			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, req)
			if rr.Code != http.StatusNoContent {
				t.Fatalf("status=%d body=%q", rr.Code, rr.Body.String())
			}
		})
	}
}

func requestWithSession(req *http.Request, sessionID string) *http.Request {
	return req.WithContext(context.WithValue(req.Context(), sessionIDKey, sessionID))
}

func openSQLiteForTaskUndoHandlerTest(t *testing.T) *db.Database {
	t.Helper()
	database, err := db.OpenSQLite(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	return database
}

func seedTaskUndoHandlerData(t *testing.T, database *db.Database) {
	t.Helper()
	if _, err := database.Conn.ExecContext(context.Background(), `
INSERT INTO projects (id, calendar_href, display_name, sync_strategy, is_default, created_at, updated_at)
VALUES ('project-1', '/cal/inbox/', 'Inbox', 'fullscan', TRUE, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
INSERT INTO tasks (
    id, project_id, uid, href, etag, server_version, title, status, raw_vtodo, base_vtodo,
    project_name, sync_status, created_at, updated_at
) VALUES (
    'task-1', 'project-1', 'uid-1', '/cal/inbox/uid-1.ics', '"etag-2"', 2, 'new', 'needs-action',
    'BEGIN:VTODO\nUID:uid-1\nSUMMARY:new\nEND:VTODO', 'BEGIN:VTODO\nUID:uid-1\nSUMMARY:new\nEND:VTODO',
    'Inbox', 'synced', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
);
INSERT INTO undo_snapshots (id, session_id, tab_id, task_id, action_type, snapshot_vtodo, snapshot_fields, etag_at_snapshot, created_at, expires_at)
VALUES ('undo-1','session-1','tab-1','task-1','task_updated','BEGIN:VTODO\nUID:uid-1\nSUMMARY:before\nEND:VTODO',json_object('title','before','description','before description','status','completed','due_date','2026-05-01','due_at','2026-05-01T09:00:00Z','priority',3,'label_names','alpha,beta'),'"etag-1"',CURRENT_TIMESTAMP,DATETIME(CURRENT_TIMESTAMP,'+5 minutes'));
INSERT INTO undo_snapshots (id, session_id, tab_id, task_id, action_type, snapshot_vtodo, snapshot_fields, etag_at_snapshot, created_at, expires_at)
VALUES ('undo-del','session-del','tab-del','task-gone','task_deleted','BEGIN:VTODO\nUID:uid-del\nSUMMARY:before\nEND:VTODO',json_object('project_id','project-1','title','before','description','before description','status','needs-action','label_names','alpha'),'"etag-1"',CURRENT_TIMESTAMP,DATETIME(CURRENT_TIMESTAMP,'+5 minutes'));
`); err != nil {
		t.Fatalf("seed undo handler data: %v", err)
	}
}

func assertSingleIntResult(t *testing.T, database *db.Database, query string, expected int) {
	t.Helper()
	var got int
	if err := database.Conn.QueryRowContext(context.Background(), query).Scan(&got); err != nil {
		t.Fatalf("query int: %v", err)
	}
	if got != expected {
		t.Fatalf("unexpected int result: got=%d want=%d", got, expected)
	}
}

func assertSingleTextResult(t *testing.T, database *db.Database, query string, expected string) {
	t.Helper()
	var got string
	if err := database.Conn.QueryRowContext(context.Background(), query).Scan(&got); err != nil {
		t.Fatalf("query text: %v", err)
	}
	if got != expected {
		t.Fatalf("unexpected text result: got=%q want=%q", got, expected)
	}
}
