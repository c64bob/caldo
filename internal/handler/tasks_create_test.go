package handler

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"caldo/internal/caldav"
	"caldo/internal/db"
)

type stubTaskCreateTodoClient struct {
	etag  string
	err   error
	href  string
	raw   string
	after func()
}

func (s *stubTaskCreateTodoClient) PutVTODOCreate(_ context.Context, _ caldav.Credentials, todoHref string, rawVTODO string) (string, error) {
	s.href = todoHref
	s.raw = rawVTODO
	if s.after != nil {
		s.after()
	}
	if s.err != nil {
		return "", s.err
	}
	return s.etag, nil
}

func TestTaskCreateSuccessPersistsSyncedTask(t *testing.T) {
	t.Parallel()
	database := openSQLiteForTaskCreateHandlerTest(t)
	seedTaskCreateHandlerProject(t, database)

	key := bytes.Repeat([]byte{0x11}, 32)
	if err := database.SaveCalDAVCredentials(context.Background(), key, db.CalDAVCredentials{URL: "https://dav.example", Username: "alice", Password: "secret"}); err != nil {
		t.Fatalf("save credentials: %v", err)
	}

	stub := &stubTaskCreateTodoClient{etag: `"etag-1"`}
	h := TaskCreate(taskCreateDependencies{database: database, encryptionKey: key, todos: stub})

	form := url.Values{"title": {"Buy milk"}, "labels": {"finance,home"}, "priority": {"high"}, "recurrence": {"FREQ=WEEKLY;BYDAY=MO"}}
	req := httptest.NewRequest(http.MethodPost, "/tasks", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("unexpected status: got %d body=%q", rr.Code, rr.Body.String())
	}
	if !strings.Contains(stub.raw, "SUMMARY:Buy milk") {
		t.Fatalf("expected summary in raw payload: %q", stub.raw)
	}
	if !strings.Contains(stub.raw, "CATEGORIES:finance,home") {
		t.Fatalf("expected categories in raw payload: %q", stub.raw)
	}
	if !strings.Contains(stub.raw, "PRIORITY:1") {
		t.Fatalf("expected priority in raw payload: %q", stub.raw)
	}
	if !strings.Contains(stub.raw, "RRULE:FREQ=WEEKLY;BYDAY=MO") {
		t.Fatalf("expected recurrence in raw payload: %q", stub.raw)
	}
	if !strings.HasPrefix(stub.href, "/cal/inbox/") || !strings.HasSuffix(stub.href, ".ics") {
		t.Fatalf("unexpected href: %q", stub.href)
	}

	var syncStatus string
	var etag string
	var serverVersion int
	if err := database.Conn.QueryRowContext(context.Background(), `SELECT sync_status, etag, server_version FROM tasks LIMIT 1;`).Scan(&syncStatus, &etag, &serverVersion); err != nil {
		t.Fatalf("query task: %v", err)
	}
	if syncStatus != "synced" || etag != `"etag-1"` || serverVersion != 2 {
		t.Fatalf("unexpected task state: status=%q etag=%q version=%d", syncStatus, etag, serverVersion)
	}
}

func TestTaskCreateCalDAVFailureMarksTaskError(t *testing.T) {
	t.Parallel()
	database := openSQLiteForTaskCreateHandlerTest(t)
	seedTaskCreateHandlerProject(t, database)

	key := bytes.Repeat([]byte{0x22}, 32)
	if err := database.SaveCalDAVCredentials(context.Background(), key, db.CalDAVCredentials{URL: "https://dav.example", Username: "alice", Password: "secret"}); err != nil {
		t.Fatalf("save credentials: %v", err)
	}

	stub := &stubTaskCreateTodoClient{err: context.DeadlineExceeded}
	h := TaskCreate(taskCreateDependencies{database: database, encryptionKey: key, todos: stub})

	form := url.Values{"title": {"Buy milk"}}
	req := httptest.NewRequest(http.MethodPost, "/tasks", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadGateway {
		t.Fatalf("unexpected status: got %d body=%q", rr.Code, rr.Body.String())
	}

	var syncStatus string
	var serverVersion int
	if err := database.Conn.QueryRowContext(context.Background(), `SELECT sync_status, server_version FROM tasks LIMIT 1;`).Scan(&syncStatus, &serverVersion); err != nil {
		t.Fatalf("query task: %v", err)
	}
	if syncStatus != "error" || serverVersion != 1 {
		t.Fatalf("unexpected task state after error: status=%q version=%d", syncStatus, serverVersion)
	}
}

func TestTaskCreatePersistsErrorStateAfterRequestCancellation(t *testing.T) {
	t.Parallel()
	database := openSQLiteForTaskCreateHandlerTest(t)
	seedTaskCreateHandlerProject(t, database)

	key := bytes.Repeat([]byte{0x44}, 32)
	if err := database.SaveCalDAVCredentials(context.Background(), key, db.CalDAVCredentials{URL: "https://dav.example", Username: "alice", Password: "secret"}); err != nil {
		t.Fatalf("save credentials: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	stub := &stubTaskCreateTodoClient{err: context.DeadlineExceeded, after: cancel}
	h := TaskCreate(taskCreateDependencies{database: database, encryptionKey: key, todos: stub})

	form := url.Values{"title": {"Buy milk"}}
	req := httptest.NewRequest(http.MethodPost, "/tasks", strings.NewReader(form.Encode())).WithContext(ctx)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadGateway {
		t.Fatalf("unexpected status: got %d body=%q", rr.Code, rr.Body.String())
	}

	var syncStatus string
	if err := database.Conn.QueryRowContext(context.Background(), `SELECT sync_status FROM tasks LIMIT 1;`).Scan(&syncStatus); err != nil {
		t.Fatalf("query task: %v", err)
	}
	if syncStatus != "error" {
		t.Fatalf("unexpected task state after cancellation: status=%q", syncStatus)
	}
}

func TestTaskCreatePersistsSyncedStateAfterRequestCancellation(t *testing.T) {
	t.Parallel()
	database := openSQLiteForTaskCreateHandlerTest(t)
	seedTaskCreateHandlerProject(t, database)

	key := bytes.Repeat([]byte{0x55}, 32)
	if err := database.SaveCalDAVCredentials(context.Background(), key, db.CalDAVCredentials{URL: "https://dav.example", Username: "alice", Password: "secret"}); err != nil {
		t.Fatalf("save credentials: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	stub := &stubTaskCreateTodoClient{etag: `"etag-cancel"`, after: cancel}
	h := TaskCreate(taskCreateDependencies{database: database, encryptionKey: key, todos: stub})

	form := url.Values{"title": {"Buy milk"}}
	req := httptest.NewRequest(http.MethodPost, "/tasks", strings.NewReader(form.Encode())).WithContext(ctx)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("unexpected status: got %d body=%q", rr.Code, rr.Body.String())
	}

	var syncStatus string
	var etag string
	if err := database.Conn.QueryRowContext(context.Background(), `SELECT sync_status, etag FROM tasks LIMIT 1;`).Scan(&syncStatus, &etag); err != nil {
		t.Fatalf("query task: %v", err)
	}
	if syncStatus != "synced" || etag != `"etag-cancel"` {
		t.Fatalf("unexpected task state after cancellation: status=%q etag=%q", syncStatus, etag)
	}
}

func TestTaskCreateWithoutValidDefaultProjectIsBlocked(t *testing.T) {
	t.Parallel()
	database := openSQLiteForTaskCreateHandlerTest(t)

	key := bytes.Repeat([]byte{0x33}, 32)
	if err := database.SaveCalDAVCredentials(context.Background(), key, db.CalDAVCredentials{URL: "https://dav.example", Username: "alice", Password: "secret"}); err != nil {
		t.Fatalf("save credentials: %v", err)
	}

	stub := &stubTaskCreateTodoClient{etag: `"etag-1"`}
	h := TaskCreate(taskCreateDependencies{database: database, encryptionKey: key, todos: stub})

	form := url.Values{"title": {"Buy milk"}}
	req := httptest.NewRequest(http.MethodPost, "/tasks", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusConflict {
		t.Fatalf("unexpected status: got %d body=%q", rr.Code, rr.Body.String())
	}

	var count int
	if err := database.Conn.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM tasks;`).Scan(&count); err != nil {
		t.Fatalf("count tasks: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected no local task rows, got %d", count)
	}
}

func seedTaskCreateHandlerProject(t *testing.T, database *db.Database) {
	t.Helper()
	if _, err := database.Conn.ExecContext(context.Background(), `
INSERT INTO projects (id, calendar_href, display_name, sync_strategy, is_default, created_at, updated_at)
VALUES ('project-default', '/cal/inbox/', 'Inbox', 'fullscan', TRUE, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
UPDATE settings SET default_project_id = 'project-default' WHERE id = 'default';
`); err != nil {
		t.Fatalf("seed project: %v", err)
	}
}

func openSQLiteForTaskCreateHandlerTest(t *testing.T) *db.Database {
	t.Helper()
	database, err := db.OpenSQLite(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	return database
}
