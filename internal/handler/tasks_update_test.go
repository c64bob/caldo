package handler

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"testing"

	"caldo/internal/caldav"
	"caldo/internal/db"
	"github.com/go-chi/chi/v5"
)

type stubTaskUpdateTodoClient struct {
	updateETag  string
	updateErr   error
	createETag  string
	createErr   error
	deleteErr   error
	updateCalls int
}

func (s *stubTaskUpdateTodoClient) PutVTODOUpdate(_ context.Context, _ caldav.Credentials, _ string, _ string, _ string) (string, error) {
	s.updateCalls++
	if s.updateErr != nil {
		return "", s.updateErr
	}
	return s.updateETag, nil
}

func (s *stubTaskUpdateTodoClient) PutVTODOCreate(_ context.Context, _ caldav.Credentials, _ string, _ string) (string, error) {
	if s.createErr != nil {
		return "", s.createErr
	}
	return s.createETag, nil
}

func (s *stubTaskUpdateTodoClient) DeleteVTODO(_ context.Context, _ caldav.Credentials, _ string, _ string) error {
	return s.deleteErr
}

func TestTaskUpdateSuccess(t *testing.T) {
	t.Parallel()
	database := openSQLiteForTaskUpdateHandlerTest(t)
	seedTaskUpdateHandlerData(t, database)

	key := bytes.Repeat([]byte{0x66}, 32)
	if err := database.SaveCalDAVCredentials(context.Background(), key, db.CalDAVCredentials{URL: "https://dav.example", Username: "alice", Password: "secret"}); err != nil {
		t.Fatalf("save credentials: %v", err)
	}

	h := TaskUpdate(taskUpdateDependencies{database: database, encryptionKey: key, todos: &stubTaskUpdateTodoClient{updateETag: `"etag-2"`}})
	form := url.Values{
		"expected_version": {"2"},
		"title":            {"new title"},
		"description":      {"updated"},
		"status":           {"needs-action"},
		"priority":         {"4"},
		"due_date":         {"2026-07-10"},
		"labels":           {"home,urgent"},
	}
	req := httptest.NewRequest(http.MethodPatch, "/tasks/task-1", bytes.NewBufferString(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Tab-ID", "tab-1")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("taskID", "task-1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("unexpected status: got %d body=%q", rr.Code, rr.Body.String())
	}

	var syncStatus, etag string
	var version int
	if err := database.Conn.QueryRowContext(context.Background(), `SELECT sync_status, etag, server_version FROM tasks WHERE id = 'task-1';`).Scan(&syncStatus, &etag, &version); err != nil {
		t.Fatalf("query task: %v", err)
	}
	if syncStatus != "synced" || etag != `"etag-2"` || version != 4 {
		t.Fatalf("unexpected row: status=%q etag=%q version=%d", syncStatus, etag, version)
	}
}

func TestTaskUpdateVersionConflict(t *testing.T) {
	t.Parallel()
	database := openSQLiteForTaskUpdateHandlerTest(t)
	seedTaskUpdateHandlerData(t, database)

	key := bytes.Repeat([]byte{0x77}, 32)
	if err := database.SaveCalDAVCredentials(context.Background(), key, db.CalDAVCredentials{URL: "https://dav.example", Username: "alice", Password: "secret"}); err != nil {
		t.Fatalf("save credentials: %v", err)
	}

	h := TaskUpdate(taskUpdateDependencies{database: database, encryptionKey: key, todos: &stubTaskUpdateTodoClient{updateETag: `"etag-2"`}})
	form := url.Values{"expected_version": {"9"}, "title": {"new title"}}
	req := httptest.NewRequest(http.MethodPatch, "/tasks/task-1", bytes.NewBufferString(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Tab-ID", "tab-1")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("taskID", "task-1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusConflict {
		t.Fatalf("unexpected status: got %d body=%q", rr.Code, rr.Body.String())
	}
}

func TestTaskUpdatePreservesDescriptionWhenOmitted(t *testing.T) {
	t.Parallel()
	database := openSQLiteForTaskUpdateHandlerTest(t)
	seedTaskUpdateHandlerData(t, database)

	key := bytes.Repeat([]byte{0x61}, 32)
	if err := database.SaveCalDAVCredentials(context.Background(), key, db.CalDAVCredentials{URL: "https://dav.example", Username: "alice", Password: "secret"}); err != nil {
		t.Fatalf("save credentials: %v", err)
	}

	h := TaskUpdate(taskUpdateDependencies{database: database, encryptionKey: key, todos: &stubTaskUpdateTodoClient{updateETag: `"etag-3"`}})
	form := url.Values{
		"expected_version": {"2"},
		"title":            {"updated title"},
	}
	req := httptest.NewRequest(http.MethodPatch, "/tasks/task-1", bytes.NewBufferString(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Tab-ID", "tab-1")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("taskID", "task-1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("unexpected status: got %d body=%q", rr.Code, rr.Body.String())
	}

	var rawVTODO string
	if err := database.Conn.QueryRowContext(context.Background(), `SELECT raw_vtodo FROM tasks WHERE id = 'task-1';`).Scan(&rawVTODO); err != nil {
		t.Fatalf("query task: %v", err)
	}
	if !bytes.Contains([]byte(rawVTODO), []byte("DESCRIPTION:old-desc")) {
		t.Fatalf("unexpected raw vtodo: %q", rawVTODO)
	}
}

func TestTaskUpdatePreconditionFailedMarksConflict(t *testing.T) {
	t.Parallel()
	database := openSQLiteForTaskUpdateHandlerTest(t)
	seedTaskUpdateHandlerData(t, database)

	key := bytes.Repeat([]byte{0x62}, 32)
	if err := database.SaveCalDAVCredentials(context.Background(), key, db.CalDAVCredentials{URL: "https://dav.example", Username: "alice", Password: "secret"}); err != nil {
		t.Fatalf("save credentials: %v", err)
	}

	h := TaskUpdate(taskUpdateDependencies{database: database, encryptionKey: key, todos: &stubTaskUpdateTodoClient{updateErr: caldav.ErrPreconditionFailed}})
	form := url.Values{"expected_version": {"2"}, "title": {"new title"}}
	req := httptest.NewRequest(http.MethodPatch, "/tasks/task-1", bytes.NewBufferString(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Tab-ID", "tab-1")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("taskID", "task-1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusConflict {
		t.Fatalf("unexpected status: got %d body=%q", rr.Code, rr.Body.String())
	}

	var syncStatus string
	var version int
	if err := database.Conn.QueryRowContext(context.Background(), `SELECT sync_status, server_version FROM tasks WHERE id = 'task-1';`).Scan(&syncStatus, &version); err != nil {
		t.Fatalf("query task: %v", err)
	}
	if syncStatus != "conflict" || version != 3 {
		t.Fatalf("unexpected row: status=%q version=%d", syncStatus, version)
	}
}

func TestTaskUpdateMoveDeleteFailurePersistsNewETag(t *testing.T) {
	t.Parallel()
	database := openSQLiteForTaskUpdateHandlerTest(t)
	seedTaskUpdateHandlerData(t, database)

	if _, err := database.Conn.ExecContext(context.Background(), `
INSERT INTO projects (id, calendar_href, display_name, sync_strategy, is_default, created_at, updated_at)
VALUES ('project-2', '/cal/work/', 'Work', 'fullscan', FALSE, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
`); err != nil {
		t.Fatalf("seed second project: %v", err)
	}

	key := bytes.Repeat([]byte{0x63}, 32)
	if err := database.SaveCalDAVCredentials(context.Background(), key, db.CalDAVCredentials{URL: "https://dav.example", Username: "alice", Password: "secret"}); err != nil {
		t.Fatalf("save credentials: %v", err)
	}

	h := TaskUpdate(taskUpdateDependencies{
		database:      database,
		encryptionKey: key,
		todos:         &stubTaskUpdateTodoClient{createETag: `"etag-new"`, deleteErr: context.DeadlineExceeded},
	})
	form := url.Values{"expected_version": {"2"}, "project_id": {"project-2"}, "title": {"new title"}}
	req := httptest.NewRequest(http.MethodPatch, "/tasks/task-1", bytes.NewBufferString(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Tab-ID", "tab-1")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("taskID", "task-1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadGateway {
		t.Fatalf("unexpected status: got %d body=%q", rr.Code, rr.Body.String())
	}

	var syncStatus, etag, href string
	var version int
	if err := database.Conn.QueryRowContext(context.Background(), `SELECT sync_status, etag, href, server_version FROM tasks WHERE id = 'task-1';`).Scan(&syncStatus, &etag, &href, &version); err != nil {
		t.Fatalf("query task: %v", err)
	}
	if syncStatus != "error" || etag != `"etag-new"` || href != "/cal/work/uid-1.ics" || version != 3 {
		t.Fatalf("unexpected row: status=%q etag=%q href=%q version=%d", syncStatus, etag, href, version)
	}
}

func openSQLiteForTaskUpdateHandlerTest(t *testing.T) *db.Database {
	t.Helper()
	database, err := db.OpenSQLite(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	return database
}

func seedTaskUpdateHandlerData(t *testing.T, database *db.Database) {
	t.Helper()
	if _, err := database.Conn.ExecContext(context.Background(), `
INSERT INTO projects (id, calendar_href, display_name, sync_strategy, is_default, created_at, updated_at)
VALUES ('project-1', '/cal/inbox/', 'Inbox', 'fullscan', TRUE, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
INSERT INTO tasks (
    id, project_id, uid, href, etag, server_version, title, description, status, raw_vtodo, base_vtodo,
    label_names, project_name, sync_status, created_at, updated_at
) VALUES (
    'task-1', 'project-1', 'uid-1', '/cal/inbox/uid-1.ics', '"etag-1"', 2, 'old', 'old-desc', 'needs-action',
    'BEGIN:VCALENDAR\nBEGIN:VTODO\nUID:uid-1\nSUMMARY:old\nDESCRIPTION:old-desc\nSTATUS:NEEDS-ACTION\nEND:VTODO\nEND:VCALENDAR',
    'BEGIN:VCALENDAR\nBEGIN:VTODO\nUID:uid-1\nSUMMARY:old\nDESCRIPTION:old-desc\nSTATUS:NEEDS-ACTION\nEND:VTODO\nEND:VCALENDAR',
    'home', 'Inbox', 'synced', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
);
`); err != nil {
		t.Fatalf("seed handler update data: %v", err)
	}
}
