package handler

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"caldo/internal/caldav"
	"caldo/internal/db"
	"github.com/go-chi/chi/v5"
)

func TestTaskDeleteSuccess(t *testing.T) {
	t.Parallel()
	database := openSQLiteForTaskUpdateHandlerTest(t)
	seedTaskUpdateHandlerData(t, database)

	key := bytes.Repeat([]byte{0x71}, 32)
	if err := database.SaveCalDAVCredentials(context.Background(), key, db.CalDAVCredentials{URL: "https://dav.example", Username: "alice", Password: "secret"}); err != nil {
		t.Fatalf("save credentials: %v", err)
	}

	h := TaskDelete(taskUpdateDependencies{database: database, encryptionKey: key, todos: &stubTaskUpdateTodoClient{}})
	form := url.Values{"expected_version": {"2"}}
	req := httptest.NewRequest(http.MethodDelete, "/tasks/task-1", bytes.NewBufferString(form.Encode()))
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

	var count int
	if err := database.Conn.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM tasks WHERE id = 'task-1';`).Scan(&count); err != nil {
		t.Fatalf("count tasks: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected task to be deleted, got %d rows", count)
	}
}

func TestTaskDeletePreconditionFailedMarksConflict(t *testing.T) {
	t.Parallel()
	database := openSQLiteForTaskUpdateHandlerTest(t)
	seedTaskUpdateHandlerData(t, database)

	key := bytes.Repeat([]byte{0x72}, 32)
	if err := database.SaveCalDAVCredentials(context.Background(), key, db.CalDAVCredentials{URL: "https://dav.example", Username: "alice", Password: "secret"}); err != nil {
		t.Fatalf("save credentials: %v", err)
	}

	h := TaskDelete(taskUpdateDependencies{database: database, encryptionKey: key, todos: &stubTaskUpdateTodoClient{deleteErr: caldav.ErrPreconditionFailed}})
	form := url.Values{"expected_version": {"2"}}
	req := httptest.NewRequest(http.MethodDelete, "/tasks/task-1", bytes.NewBufferString(form.Encode()))
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

func TestTaskDeleteCalDAVErrorMarksError(t *testing.T) {
	t.Parallel()
	database := openSQLiteForTaskUpdateHandlerTest(t)
	seedTaskUpdateHandlerData(t, database)

	key := bytes.Repeat([]byte{0x73}, 32)
	if err := database.SaveCalDAVCredentials(context.Background(), key, db.CalDAVCredentials{URL: "https://dav.example", Username: "alice", Password: "secret"}); err != nil {
		t.Fatalf("save credentials: %v", err)
	}

	h := TaskDelete(taskUpdateDependencies{database: database, encryptionKey: key, todos: &stubTaskUpdateTodoClient{deleteErr: context.DeadlineExceeded}})
	form := url.Values{"expected_version": {"2"}}
	req := httptest.NewRequest(http.MethodDelete, "/tasks/task-1", bytes.NewBufferString(form.Encode()))
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

	var syncStatus string
	if err := database.Conn.QueryRowContext(context.Background(), `SELECT sync_status FROM tasks WHERE id = 'task-1';`).Scan(&syncStatus); err != nil {
		t.Fatalf("query task: %v", err)
	}
	if syncStatus != "error" {
		t.Fatalf("unexpected sync status: %q", syncStatus)
	}
}

func TestTaskDeleteCredentialsUnavailableDoesNotPersistPendingDelete(t *testing.T) {
	t.Parallel()
	database := openSQLiteForTaskUpdateHandlerTest(t)
	seedTaskUpdateHandlerData(t, database)

	h := TaskDelete(taskUpdateDependencies{
		database:      database,
		encryptionKey: bytes.Repeat([]byte{0x74}, 32),
		todos:         &stubTaskUpdateTodoClient{},
	})
	form := url.Values{"expected_version": {"2"}}
	req := httptest.NewRequest(http.MethodDelete, "/tasks/task-1", bytes.NewBufferString(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Tab-ID", "tab-1")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("taskID", "task-1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusFailedDependency {
		t.Fatalf("unexpected status: got %d body=%q", rr.Code, rr.Body.String())
	}

	var syncStatus string
	var version int
	if err := database.Conn.QueryRowContext(context.Background(), `SELECT sync_status, server_version FROM tasks WHERE id = 'task-1';`).Scan(&syncStatus, &version); err != nil {
		t.Fatalf("query task: %v", err)
	}
	if syncStatus != "synced" || version != 2 {
		t.Fatalf("unexpected task state: sync_status=%q version=%d", syncStatus, version)
	}
}

func TestTaskDeleteWithDirectSubtasksRequiresConfirmation(t *testing.T) {
	t.Parallel()
	database := openSQLiteForTaskUpdateHandlerTest(t)
	seedTaskUpdateHandlerData(t, database)
	seedOpenSubtask(t, database, "task-child-1", "task-1")

	key := bytes.Repeat([]byte{0x75}, 32)
	if err := database.SaveCalDAVCredentials(context.Background(), key, db.CalDAVCredentials{URL: "https://dav.example", Username: "alice", Password: "secret"}); err != nil {
		t.Fatalf("save credentials: %v", err)
	}

	stub := &stubTaskUpdateTodoClient{}
	h := TaskDelete(taskUpdateDependencies{database: database, encryptionKey: key, todos: stub})
	form := url.Values{"expected_version": {"2"}}
	req := httptest.NewRequest(http.MethodDelete, "/tasks/task-1", bytes.NewBufferString(form.Encode()))
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
	if stub.deleteCalls != 0 {
		t.Fatalf("expected no CalDAV delete calls, got %d", stub.deleteCalls)
	}
}

func TestTaskDeleteWithDirectSubtasksDeleteAllDeletesParentAndSubtasks(t *testing.T) {
	t.Parallel()
	database := openSQLiteForTaskUpdateHandlerTest(t)
	seedTaskUpdateHandlerData(t, database)
	seedOpenSubtask(t, database, "task-child-1", "task-1")
	seedOpenSubtask(t, database, "task-child-2", "task-1")

	key := bytes.Repeat([]byte{0x76}, 32)
	if err := database.SaveCalDAVCredentials(context.Background(), key, db.CalDAVCredentials{URL: "https://dav.example", Username: "alice", Password: "secret"}); err != nil {
		t.Fatalf("save credentials: %v", err)
	}

	stub := &stubTaskUpdateTodoClient{}
	h := TaskDelete(taskUpdateDependencies{database: database, encryptionKey: key, todos: stub})
	form := url.Values{"expected_version": {"2"}, "subtasks_action": {"delete_all"}}
	req := httptest.NewRequest(http.MethodDelete, "/tasks/task-1", bytes.NewBufferString(form.Encode()))
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
	if stub.deleteCalls != 3 {
		t.Fatalf("expected three CalDAV delete calls, got %d", stub.deleteCalls)
	}

	for _, taskID := range []string{"task-1", "task-child-1", "task-child-2"} {
		var count int
		if err := database.Conn.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM tasks WHERE id = ?;`, taskID).Scan(&count); err != nil {
			t.Fatalf("count task %s: %v", taskID, err)
		}
		if count != 0 {
			t.Fatalf("expected task %s deleted, got %d rows", taskID, count)
		}
	}
}

func TestTaskDeleteWithDirectSubtasksDeleteAllVersionConflictDoesNotDeleteSubtasks(t *testing.T) {
	t.Parallel()
	database := openSQLiteForTaskUpdateHandlerTest(t)
	seedTaskUpdateHandlerData(t, database)
	seedOpenSubtask(t, database, "task-child-1", "task-1")
	seedOpenSubtask(t, database, "task-child-2", "task-1")

	key := bytes.Repeat([]byte{0x77}, 32)
	if err := database.SaveCalDAVCredentials(context.Background(), key, db.CalDAVCredentials{URL: "https://dav.example", Username: "alice", Password: "secret"}); err != nil {
		t.Fatalf("save credentials: %v", err)
	}

	stub := &stubTaskUpdateTodoClient{}
	h := TaskDelete(taskUpdateDependencies{database: database, encryptionKey: key, todos: stub})
	form := url.Values{"expected_version": {"1"}, "subtasks_action": {"delete_all"}}
	req := httptest.NewRequest(http.MethodDelete, "/tasks/task-1", bytes.NewBufferString(form.Encode()))
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
	if stub.deleteCalls != 0 {
		t.Fatalf("expected no CalDAV delete calls, got %d", stub.deleteCalls)
	}

	for _, taskID := range []string{"task-1", "task-child-1", "task-child-2"} {
		var count int
		if err := database.Conn.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM tasks WHERE id = ?;`, taskID).Scan(&count); err != nil {
			t.Fatalf("count task %s: %v", taskID, err)
		}
		if count != 1 {
			t.Fatalf("expected task %s to remain, got %d rows", taskID, count)
		}
	}
}
