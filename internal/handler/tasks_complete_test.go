package handler

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"caldo/internal/caldav"
	"caldo/internal/db"
	"github.com/go-chi/chi/v5"
)

func setTaskRawVTODO(t *testing.T, database *db.Database, rawVTODO string) {
	t.Helper()
	if _, err := database.Conn.ExecContext(context.Background(), `
UPDATE tasks
SET raw_vtodo = ?, base_vtodo = ?
WHERE id = 'task-1';
`, rawVTODO, rawVTODO); err != nil {
		t.Fatalf("set raw vtodo: %v", err)
	}
}

func seedOpenSubtask(t *testing.T, database *db.Database, taskID string, parentID string) {
	t.Helper()
	if _, err := database.Conn.ExecContext(context.Background(), `
INSERT INTO tasks (
    id, project_id, uid, href, etag, server_version, title, description, status, parent_id, raw_vtodo, base_vtodo,
    label_names, project_name, sync_status, created_at, updated_at
) VALUES (
    ?, 'project-1', ?, ?, '"etag-child"', 2, 'child', NULL, 'needs-action', ?,
    ?, ?, NULL, 'Inbox', 'synced', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
);
`, taskID, "uid-"+taskID, "/cal/inbox/uid-"+taskID+".ics", parentID,
		"BEGIN:VCALENDAR\nBEGIN:VTODO\nUID:uid-"+taskID+"\nSUMMARY:child\nSTATUS:NEEDS-ACTION\nEND:VTODO\nEND:VCALENDAR",
		"BEGIN:VCALENDAR\nBEGIN:VTODO\nUID:uid-"+taskID+"\nSUMMARY:child\nSTATUS:NEEDS-ACTION\nEND:VTODO\nEND:VCALENDAR"); err != nil {
		t.Fatalf("seed child task: %v", err)
	}
}

func TestTaskCompleteSuccess(t *testing.T) {
	t.Parallel()
	database := openSQLiteForTaskUpdateHandlerTest(t)
	seedTaskUpdateHandlerData(t, database)
	setTaskRawVTODO(t, database, `BEGIN:VCALENDAR
BEGIN:VTODO
UID:uid-1
SUMMARY:old
DESCRIPTION:old-desc
STATUS:NEEDS-ACTION
END:VTODO
END:VCALENDAR`)

	key := bytes.Repeat([]byte{0x51}, 32)
	if err := database.SaveCalDAVCredentials(context.Background(), key, db.CalDAVCredentials{URL: "https://dav.example", Username: "alice", Password: "secret"}); err != nil {
		t.Fatalf("save credentials: %v", err)
	}

	h := TaskComplete(taskUpdateDependencies{database: database, encryptionKey: key, todos: &stubTaskUpdateTodoClient{updateETag: `"etag-2"`}})
	form := url.Values{"expected_version": {"2"}}
	req := httptest.NewRequest(http.MethodPost, "/tasks/task-1/complete", bytes.NewBufferString(form.Encode()))
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

	var status, rawVTODO string
	if err := database.Conn.QueryRowContext(context.Background(), `SELECT status, raw_vtodo FROM tasks WHERE id = 'task-1';`).Scan(&status, &rawVTODO); err != nil {
		t.Fatalf("query task: %v", err)
	}
	if status != "completed" {
		t.Fatalf("unexpected status: %q", status)
	}
	if !strings.Contains(rawVTODO, "STATUS:COMPLETED") || !strings.Contains(rawVTODO, "COMPLETED:") {
		t.Fatalf("expected completed fields in vtodo, got %q", rawVTODO)
	}
}

func TestTaskReopenClearsCompletedAndPreservesRRule(t *testing.T) {
	t.Parallel()
	database := openSQLiteForTaskUpdateHandlerTest(t)
	seedTaskUpdateHandlerData(t, database)

	if _, err := database.Conn.ExecContext(context.Background(), `
UPDATE tasks
SET status = 'completed'
WHERE id = 'task-1';
`); err != nil {
		t.Fatalf("seed completed task status: %v", err)
	}
	setTaskRawVTODO(t, database, `BEGIN:VCALENDAR
BEGIN:VTODO
UID:uid-1
SUMMARY:old
STATUS:COMPLETED
COMPLETED:20260101T120000Z
RRULE:FREQ=MONTHLY;BYDAY=MO,TU,WE,TH,FR;BYSETPOS=1
END:VTODO
END:VCALENDAR`)

	key := bytes.Repeat([]byte{0x52}, 32)
	if err := database.SaveCalDAVCredentials(context.Background(), key, db.CalDAVCredentials{URL: "https://dav.example", Username: "alice", Password: "secret"}); err != nil {
		t.Fatalf("save credentials: %v", err)
	}

	h := TaskReopen(taskUpdateDependencies{database: database, encryptionKey: key, todos: &stubTaskUpdateTodoClient{updateETag: `"etag-3"`}})
	form := url.Values{"expected_version": {"2"}}
	req := httptest.NewRequest(http.MethodPost, "/tasks/task-1/reopen", bytes.NewBufferString(form.Encode()))
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

	var status, rawVTODO string
	if err := database.Conn.QueryRowContext(context.Background(), `SELECT status, raw_vtodo FROM tasks WHERE id = 'task-1';`).Scan(&status, &rawVTODO); err != nil {
		t.Fatalf("query task: %v", err)
	}
	if status != "needs-action" {
		t.Fatalf("unexpected status: %q", status)
	}
	if strings.Contains(rawVTODO, "COMPLETED:") {
		t.Fatalf("expected completed to be removed from vtodo, got %q", rawVTODO)
	}
	if !strings.Contains(rawVTODO, "RRULE:FREQ=MONTHLY;BYDAY=MO,TU,WE,TH,FR;BYSETPOS=1") {
		t.Fatalf("expected rrule to remain unchanged, got %q", rawVTODO)
	}
}

func TestTaskCompleteCalDAVErrorMarksErrorAndReturnsBadGateway(t *testing.T) {
	t.Parallel()
	database := openSQLiteForTaskUpdateHandlerTest(t)
	seedTaskUpdateHandlerData(t, database)
	setTaskRawVTODO(t, database, `BEGIN:VCALENDAR
BEGIN:VTODO
UID:uid-1
SUMMARY:old
STATUS:NEEDS-ACTION
END:VTODO
END:VCALENDAR`)

	key := bytes.Repeat([]byte{0x53}, 32)
	if err := database.SaveCalDAVCredentials(context.Background(), key, db.CalDAVCredentials{URL: "https://dav.example", Username: "alice", Password: "secret"}); err != nil {
		t.Fatalf("save credentials: %v", err)
	}

	h := TaskComplete(taskUpdateDependencies{database: database, encryptionKey: key, todos: &stubTaskUpdateTodoClient{updateErr: context.DeadlineExceeded}})
	form := url.Values{"expected_version": {"2"}}
	req := httptest.NewRequest(http.MethodPost, "/tasks/task-1/complete", bytes.NewBufferString(form.Encode()))
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

func TestTaskReopenPreconditionFailedMarksConflict(t *testing.T) {
	t.Parallel()
	database := openSQLiteForTaskUpdateHandlerTest(t)
	seedTaskUpdateHandlerData(t, database)
	setTaskRawVTODO(t, database, `BEGIN:VCALENDAR
BEGIN:VTODO
UID:uid-1
SUMMARY:old
STATUS:COMPLETED
COMPLETED:20260101T120000Z
END:VTODO
END:VCALENDAR`)

	key := bytes.Repeat([]byte{0x54}, 32)
	if err := database.SaveCalDAVCredentials(context.Background(), key, db.CalDAVCredentials{URL: "https://dav.example", Username: "alice", Password: "secret"}); err != nil {
		t.Fatalf("save credentials: %v", err)
	}

	h := TaskReopen(taskUpdateDependencies{database: database, encryptionKey: key, todos: &stubTaskUpdateTodoClient{updateErr: caldav.ErrPreconditionFailed}})
	form := url.Values{"expected_version": {"2"}}
	req := httptest.NewRequest(http.MethodPost, "/tasks/task-1/reopen", bytes.NewBufferString(form.Encode()))
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
	if err := database.Conn.QueryRowContext(context.Background(), `SELECT sync_status FROM tasks WHERE id = 'task-1';`).Scan(&syncStatus); err != nil {
		t.Fatalf("query task: %v", err)
	}
	if syncStatus != "conflict" {
		t.Fatalf("unexpected sync status: %q", syncStatus)
	}
}

func TestTaskCompleteCredentialsUnavailableDoesNotPersistPendingUpdate(t *testing.T) {
	t.Parallel()
	database := openSQLiteForTaskUpdateHandlerTest(t)
	seedTaskUpdateHandlerData(t, database)
	setTaskRawVTODO(t, database, `BEGIN:VCALENDAR
BEGIN:VTODO
UID:uid-1
SUMMARY:old
STATUS:NEEDS-ACTION
END:VTODO
END:VCALENDAR`)

	h := TaskComplete(taskUpdateDependencies{
		database:      database,
		encryptionKey: bytes.Repeat([]byte{0x55}, 32),
		todos:         &stubTaskUpdateTodoClient{updateETag: `"etag-2"`},
	})
	form := url.Values{"expected_version": {"2"}}
	req := httptest.NewRequest(http.MethodPost, "/tasks/task-1/complete", bytes.NewBufferString(form.Encode()))
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

	var status, syncStatus string
	var version int
	if err := database.Conn.QueryRowContext(context.Background(), `SELECT status, sync_status, server_version FROM tasks WHERE id = 'task-1';`).Scan(&status, &syncStatus, &version); err != nil {
		t.Fatalf("query task: %v", err)
	}
	if status != "needs-action" || syncStatus != "synced" || version != 2 {
		t.Fatalf("unexpected task state: status=%q sync_status=%q version=%d", status, syncStatus, version)
	}
}

func TestTaskCompleteWithOpenSubtasksRequiresConfirmation(t *testing.T) {
	t.Parallel()
	database := openSQLiteForTaskUpdateHandlerTest(t)
	seedTaskUpdateHandlerData(t, database)
	setTaskRawVTODO(t, database, `BEGIN:VCALENDAR
BEGIN:VTODO
UID:uid-1
SUMMARY:old
STATUS:NEEDS-ACTION
END:VTODO
END:VCALENDAR`)
	seedOpenSubtask(t, database, "task-child-1", "task-1")

	key := bytes.Repeat([]byte{0x56}, 32)
	if err := database.SaveCalDAVCredentials(context.Background(), key, db.CalDAVCredentials{URL: "https://dav.example", Username: "alice", Password: "secret"}); err != nil {
		t.Fatalf("save credentials: %v", err)
	}

	h := TaskComplete(taskUpdateDependencies{database: database, encryptionKey: key, todos: &stubTaskUpdateTodoClient{updateETag: `"etag-2"`}})
	form := url.Values{"expected_version": {"2"}}
	req := httptest.NewRequest(http.MethodPost, "/tasks/task-1/complete", bytes.NewBufferString(form.Encode()))
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

	var parentStatus, childStatus string
	if err := database.Conn.QueryRowContext(context.Background(), `SELECT status FROM tasks WHERE id = 'task-1';`).Scan(&parentStatus); err != nil {
		t.Fatalf("query parent task: %v", err)
	}
	if err := database.Conn.QueryRowContext(context.Background(), `SELECT status FROM tasks WHERE id = 'task-child-1';`).Scan(&childStatus); err != nil {
		t.Fatalf("query child task: %v", err)
	}
	if parentStatus != "needs-action" || childStatus != "needs-action" {
		t.Fatalf("unexpected task status after confirmation requirement: parent=%q child=%q", parentStatus, childStatus)
	}
}

func TestTaskCompleteWithOpenSubtasksCancelDoesNotMutate(t *testing.T) {
	t.Parallel()
	database := openSQLiteForTaskUpdateHandlerTest(t)
	seedTaskUpdateHandlerData(t, database)
	setTaskRawVTODO(t, database, `BEGIN:VCALENDAR
BEGIN:VTODO
UID:uid-1
SUMMARY:old
STATUS:NEEDS-ACTION
END:VTODO
END:VCALENDAR`)
	seedOpenSubtask(t, database, "task-child-1", "task-1")

	key := bytes.Repeat([]byte{0x57}, 32)
	if err := database.SaveCalDAVCredentials(context.Background(), key, db.CalDAVCredentials{URL: "https://dav.example", Username: "alice", Password: "secret"}); err != nil {
		t.Fatalf("save credentials: %v", err)
	}

	h := TaskComplete(taskUpdateDependencies{database: database, encryptionKey: key, todos: &stubTaskUpdateTodoClient{updateETag: `"etag-2"`}})
	form := url.Values{"expected_version": {"2"}, "subtasks_action": {"cancel"}}
	req := httptest.NewRequest(http.MethodPost, "/tasks/task-1/complete", bytes.NewBufferString(form.Encode()))
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

	var parentVersion, childVersion int
	if err := database.Conn.QueryRowContext(context.Background(), `SELECT server_version FROM tasks WHERE id = 'task-1';`).Scan(&parentVersion); err != nil {
		t.Fatalf("query parent version: %v", err)
	}
	if err := database.Conn.QueryRowContext(context.Background(), `SELECT server_version FROM tasks WHERE id = 'task-child-1';`).Scan(&childVersion); err != nil {
		t.Fatalf("query child version: %v", err)
	}
	if parentVersion != 2 || childVersion != 2 {
		t.Fatalf("unexpected versions after cancel: parent=%d child=%d", parentVersion, childVersion)
	}
}

func TestTaskCompleteWithOpenSubtasksParentOnlyCompletesParent(t *testing.T) {
	t.Parallel()
	database := openSQLiteForTaskUpdateHandlerTest(t)
	seedTaskUpdateHandlerData(t, database)
	setTaskRawVTODO(t, database, `BEGIN:VCALENDAR
BEGIN:VTODO
UID:uid-1
SUMMARY:old
STATUS:NEEDS-ACTION
END:VTODO
END:VCALENDAR`)
	seedOpenSubtask(t, database, "task-child-1", "task-1")

	key := bytes.Repeat([]byte{0x58}, 32)
	if err := database.SaveCalDAVCredentials(context.Background(), key, db.CalDAVCredentials{URL: "https://dav.example", Username: "alice", Password: "secret"}); err != nil {
		t.Fatalf("save credentials: %v", err)
	}

	stub := &stubTaskUpdateTodoClient{updateETag: `"etag-2"`}
	h := TaskComplete(taskUpdateDependencies{database: database, encryptionKey: key, todos: stub})
	form := url.Values{"expected_version": {"2"}, "subtasks_action": {"parent_only"}}
	req := httptest.NewRequest(http.MethodPost, "/tasks/task-1/complete", bytes.NewBufferString(form.Encode()))
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
	if stub.updateCalls != 1 {
		t.Fatalf("expected one CalDAV write, got %d", stub.updateCalls)
	}

	var parentStatus, childStatus string
	if err := database.Conn.QueryRowContext(context.Background(), `SELECT status FROM tasks WHERE id = 'task-1';`).Scan(&parentStatus); err != nil {
		t.Fatalf("query parent task: %v", err)
	}
	if err := database.Conn.QueryRowContext(context.Background(), `SELECT status FROM tasks WHERE id = 'task-child-1';`).Scan(&childStatus); err != nil {
		t.Fatalf("query child task: %v", err)
	}
	if parentStatus != "completed" || childStatus != "needs-action" {
		t.Fatalf("unexpected task status: parent=%q child=%q", parentStatus, childStatus)
	}
}

func TestTaskCompleteWithOpenSubtasksCompleteOpenStaleVersionDoesNotMutateSubtasks(t *testing.T) {
	t.Parallel()
	database := openSQLiteForTaskUpdateHandlerTest(t)
	seedTaskUpdateHandlerData(t, database)
	setTaskRawVTODO(t, database, `BEGIN:VCALENDAR
BEGIN:VTODO
UID:uid-1
SUMMARY:old
STATUS:NEEDS-ACTION
END:VTODO
END:VCALENDAR`)
	seedOpenSubtask(t, database, "task-child-1", "task-1")

	key := bytes.Repeat([]byte{0x5a}, 32)
	if err := database.SaveCalDAVCredentials(context.Background(), key, db.CalDAVCredentials{URL: "https://dav.example", Username: "alice", Password: "secret"}); err != nil {
		t.Fatalf("save credentials: %v", err)
	}

	stub := &stubTaskUpdateTodoClient{updateETag: `"etag-2"`}
	h := TaskComplete(taskUpdateDependencies{database: database, encryptionKey: key, todos: stub})
	form := url.Values{"expected_version": {"9"}, "subtasks_action": {"complete_open"}}
	req := httptest.NewRequest(http.MethodPost, "/tasks/task-1/complete", bytes.NewBufferString(form.Encode()))
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
	if stub.updateCalls != 0 {
		t.Fatalf("expected no CalDAV writes for stale parent version, got %d", stub.updateCalls)
	}

	for _, taskID := range []string{"task-1", "task-child-1"} {
		var status string
		var version int
		if err := database.Conn.QueryRowContext(context.Background(), `SELECT status, server_version FROM tasks WHERE id = ?;`, taskID).Scan(&status, &version); err != nil {
			t.Fatalf("query task %s: %v", taskID, err)
		}
		if status != "needs-action" || version != 2 {
			t.Fatalf("unexpected task state for %s: status=%q version=%d", taskID, status, version)
		}
	}
}

func TestTaskCompleteWithOpenSubtasksCompleteOpenCompletesAll(t *testing.T) {
	t.Parallel()
	database := openSQLiteForTaskUpdateHandlerTest(t)
	seedTaskUpdateHandlerData(t, database)
	setTaskRawVTODO(t, database, `BEGIN:VCALENDAR
BEGIN:VTODO
UID:uid-1
SUMMARY:old
STATUS:NEEDS-ACTION
END:VTODO
END:VCALENDAR`)
	seedOpenSubtask(t, database, "task-child-1", "task-1")
	seedOpenSubtask(t, database, "task-child-2", "task-1")

	key := bytes.Repeat([]byte{0x59}, 32)
	if err := database.SaveCalDAVCredentials(context.Background(), key, db.CalDAVCredentials{URL: "https://dav.example", Username: "alice", Password: "secret"}); err != nil {
		t.Fatalf("save credentials: %v", err)
	}

	stub := &stubTaskUpdateTodoClient{updateETag: `"etag-2"`}
	h := TaskComplete(taskUpdateDependencies{database: database, encryptionKey: key, todos: stub})
	form := url.Values{"expected_version": {"2"}, "subtasks_action": {"complete_open"}}
	req := httptest.NewRequest(http.MethodPost, "/tasks/task-1/complete", bytes.NewBufferString(form.Encode()))
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
	if stub.updateCalls != 3 {
		t.Fatalf("expected three CalDAV writes, got %d", stub.updateCalls)
	}

	for _, taskID := range []string{"task-1", "task-child-1", "task-child-2"} {
		var status string
		var rawVTODO string
		if err := database.Conn.QueryRowContext(context.Background(), `SELECT status, raw_vtodo FROM tasks WHERE id = ?;`, taskID).Scan(&status, &rawVTODO); err != nil {
			t.Fatalf("query task %s: %v", taskID, err)
		}
		if status != "completed" {
			t.Fatalf("unexpected status for %s: %q", taskID, status)
		}
		if !strings.Contains(rawVTODO, "STATUS:COMPLETED") || !strings.Contains(rawVTODO, "COMPLETED:") {
			t.Fatalf("expected completed fields in vtodo for %s, got %q", taskID, rawVTODO)
		}
	}
}
