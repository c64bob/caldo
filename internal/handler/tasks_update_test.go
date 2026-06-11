package handler

import (
	"bytes"
	"context"
	"database/sql"
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
	updateETag     string
	updateErr      error
	createETag     string
	createErr      error
	deleteErr      error
	updateCalls    int
	deleteCalls    int
	createCalls    int
	lastHref       string
	lastDeleteHref string
	lastDeleteETag string
	lastRawVTODO   string
	lastUpdateETag string
	onCreate       func()
}

func (s *stubTaskUpdateTodoClient) PutVTODOUpdate(_ context.Context, _ caldav.Credentials, href string, rawVTODO string, etag string) (string, error) {
	s.updateCalls++
	s.lastHref = href
	s.lastRawVTODO = rawVTODO
	s.lastUpdateETag = etag
	if s.updateErr != nil {
		return "", s.updateErr
	}
	return s.updateETag, nil
}

func (s *stubTaskUpdateTodoClient) PutVTODOCreate(_ context.Context, _ caldav.Credentials, href string, rawVTODO string) (string, error) {
	s.createCalls++
	s.lastHref = href
	s.lastRawVTODO = rawVTODO
	if s.onCreate != nil {
		s.onCreate()
	}
	if s.createErr != nil {
		return "", s.createErr
	}
	return s.createETag, nil
}

func (s *stubTaskUpdateTodoClient) DeleteVTODO(_ context.Context, _ caldav.Credentials, href string, etag string) error {
	s.deleteCalls++
	s.lastDeleteHref = href
	s.lastDeleteETag = etag
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

	stub := &stubTaskUpdateTodoClient{updateETag: `"etag-2"`}
	h := TaskUpdate(taskUpdateDependencies{database: database, encryptionKey: key, todos: stub})
	form := url.Values{
		"expected_version": {"2"},
		"title":            {"new title"},
		"description":      {"updated"},
		"status":           {"completed"},
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

	var title, description, status, dueDate, labelNames, rawVTODO, syncStatus, etag string
	var priority int
	var version int
	if err := database.Conn.QueryRowContext(context.Background(), `
SELECT title, description, status, date(due_date), priority, label_names, raw_vtodo, sync_status, etag, server_version
FROM tasks
WHERE id = 'task-1';
`).Scan(&title, &description, &status, &dueDate, &priority, &labelNames, &rawVTODO, &syncStatus, &etag, &version); err != nil {
		t.Fatalf("query task: %v", err)
	}
	if title != "new title" || description != "updated" || status != "completed" || dueDate != "2026-07-10" || priority != 4 || labelNames != "home,urgent" {
		t.Fatalf("unexpected edited fields: title=%q description=%q status=%q due=%q priority=%d labels=%q", title, description, status, dueDate, priority, labelNames)
	}
	if syncStatus != "synced" || etag != `"etag-2"` || version != 4 {
		t.Fatalf("unexpected sync row: status=%q etag=%q version=%d", syncStatus, etag, version)
	}
	for _, want := range []string{"SUMMARY:new title", "DESCRIPTION:updated", "STATUS:COMPLETED", "DUE;VALUE=DATE:20260710", "PRIORITY:4", "CATEGORIES:home,urgent", "COMPLETED:"} {
		if !bytes.Contains([]byte(rawVTODO), []byte(want)) {
			t.Fatalf("expected raw vtodo to contain %q in %q", want, rawVTODO)
		}
	}
	if stub.updateCalls != 1 || stub.lastHref != "/cal/inbox/uid-1.ics" || stub.lastUpdateETag != `"etag-1"` || stub.lastRawVTODO != rawVTODO {
		t.Fatalf("unexpected caldav update call: calls=%d href=%q etag=%q raw=%q", stub.updateCalls, stub.lastHref, stub.lastUpdateETag, stub.lastRawVTODO)
	}
	assertSingleIntResult(t, database, `SELECT COUNT(*) FROM undo_snapshots WHERE session_id = 'single-user-session' AND tab_id = 'tab-1' AND task_id = 'task-1' AND action_type = 'task_updated';`, 1)
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

func TestTaskUpdateClearsExplicitFieldsAndPreservesFavoriteCategory(t *testing.T) {
	t.Parallel()
	database := openSQLiteForTaskUpdateHandlerTest(t)
	seedTaskUpdateHandlerData(t, database)

	if _, err := database.Conn.ExecContext(context.Background(), `
UPDATE tasks
SET raw_vtodo = 'BEGIN:VCALENDAR
BEGIN:VTODO
UID:uid-1
SUMMARY:old
DESCRIPTION:old-desc
STATUS:NEEDS-ACTION
DUE;VALUE=DATE:20260710
PRIORITY:4
CATEGORIES:home,STARRED
END:VTODO
END:VCALENDAR',
    due_date = '2026-07-10',
    priority = 4,
    label_names = 'home,STARRED'
WHERE id = 'task-1';
`); err != nil {
		t.Fatalf("seed editable fields: %v", err)
	}

	key := bytes.Repeat([]byte{0x72}, 32)
	if err := database.SaveCalDAVCredentials(context.Background(), key, db.CalDAVCredentials{URL: "https://dav.example", Username: "alice", Password: "secret"}); err != nil {
		t.Fatalf("save credentials: %v", err)
	}

	h := TaskUpdate(taskUpdateDependencies{database: database, encryptionKey: key, todos: &stubTaskUpdateTodoClient{updateETag: `"etag-clear"`}})
	form := url.Values{
		"expected_version": {"2"},
		"title":            {"old"},
		"description":      {""},
		"status":           {"needs-action"},
		"due_date":         {""},
		"priority":         {""},
		"labels":           {""},
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
	var description, dueDate, labelNames sql.NullString
	var priority sql.NullInt64
	if err := database.Conn.QueryRowContext(context.Background(), `
SELECT raw_vtodo, description, due_date, priority, label_names
FROM tasks
WHERE id = 'task-1';
`).Scan(&rawVTODO, &description, &dueDate, &priority, &labelNames); err != nil {
		t.Fatalf("query task: %v", err)
	}
	for _, notWant := range []string{"DESCRIPTION:", "DUE", "PRIORITY:", "home"} {
		if bytes.Contains([]byte(rawVTODO), []byte(notWant)) {
			t.Fatalf("raw vtodo unexpectedly contains %q: %q", notWant, rawVTODO)
		}
	}
	if !bytes.Contains([]byte(rawVTODO), []byte("CATEGORIES:STARRED")) {
		t.Fatalf("expected favorite category to be preserved, raw=%q", rawVTODO)
	}
	if description.Valid || dueDate.Valid || priority.Valid {
		t.Fatalf("expected cleared nullable fields, description=%#v dueDate=%#v priority=%#v", description, dueDate, priority)
	}
	if !labelNames.Valid || labelNames.String != "STARRED" {
		t.Fatalf("expected only favorite label name to remain, got %#v", labelNames)
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

func TestTaskUpdateCredentialsUnavailableMarksError(t *testing.T) {
	t.Parallel()
	database := openSQLiteForTaskUpdateHandlerTest(t)
	seedTaskUpdateHandlerData(t, database)

	key := bytes.Repeat([]byte{0x73}, 32)
	stub := &stubTaskUpdateTodoClient{updateETag: `"etag-2"`}
	h := TaskUpdate(taskUpdateDependencies{database: database, encryptionKey: key, todos: stub})
	form := url.Values{"expected_version": {"2"}, "title": {"new title"}}
	req := httptest.NewRequest(http.MethodPatch, "/tasks/task-1", bytes.NewBufferString(form.Encode()))
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
	if stub.updateCalls != 0 || stub.createCalls != 0 || stub.deleteCalls != 0 {
		t.Fatalf("expected no caldav writes, got update=%d create=%d delete=%d", stub.updateCalls, stub.createCalls, stub.deleteCalls)
	}

	var syncStatus, title string
	var version int
	if err := database.Conn.QueryRowContext(context.Background(), `SELECT sync_status, title, server_version FROM tasks WHERE id = 'task-1';`).Scan(&syncStatus, &title, &version); err != nil {
		t.Fatalf("query task: %v", err)
	}
	if syncStatus != "error" || title != "new title" || version != 3 {
		t.Fatalf("unexpected row: status=%q title=%q version=%d", syncStatus, title, version)
	}
}

func TestTaskUpdateMoveSuccessCreatesTargetResourceAndDeletesPrevious(t *testing.T) {
	t.Parallel()
	database := openSQLiteForTaskUpdateHandlerTest(t)
	seedTaskUpdateHandlerData(t, database)

	if _, err := database.Conn.ExecContext(context.Background(), `
INSERT INTO projects (id, calendar_href, display_name, sync_strategy, is_default, created_at, updated_at)
VALUES ('project-2', '/cal/work/', 'Work', 'fullscan', FALSE, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
`); err != nil {
		t.Fatalf("seed second project: %v", err)
	}

	key := bytes.Repeat([]byte{0x74}, 32)
	if err := database.SaveCalDAVCredentials(context.Background(), key, db.CalDAVCredentials{URL: "https://dav.example", Username: "alice", Password: "secret"}); err != nil {
		t.Fatalf("save credentials: %v", err)
	}

	stub := &stubTaskUpdateTodoClient{createETag: `"etag-moved"`}
	h := TaskUpdate(taskUpdateDependencies{database: database, encryptionKey: key, todos: stub})
	form := url.Values{"expected_version": {"2"}, "project_id": {"project-2"}, "title": {"moved title"}}
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
	if stub.createCalls != 1 || stub.deleteCalls != 1 || stub.updateCalls != 0 {
		t.Fatalf("unexpected caldav calls: update=%d create=%d delete=%d", stub.updateCalls, stub.createCalls, stub.deleteCalls)
	}
	if stub.lastHref != "/cal/work/uid-1.ics" {
		t.Fatalf("unexpected created href: %q", stub.lastHref)
	}
	if !bytes.Contains([]byte(stub.lastRawVTODO), []byte("SUMMARY:moved title")) {
		t.Fatalf("expected created raw vtodo to contain moved title, raw=%q", stub.lastRawVTODO)
	}

	var projectID, projectName, href, syncStatus, etag string
	var version int
	if err := database.Conn.QueryRowContext(context.Background(), `
SELECT project_id, project_name, href, sync_status, etag, server_version
FROM tasks
WHERE id = 'task-1';
`).Scan(&projectID, &projectName, &href, &syncStatus, &etag, &version); err != nil {
		t.Fatalf("query task: %v", err)
	}
	if projectID != "project-2" || projectName != "Work" || href != "/cal/work/uid-1.ics" || syncStatus != "synced" || etag != `"etag-moved"` || version != 4 {
		t.Fatalf("unexpected moved row: project=%q name=%q href=%q status=%q etag=%q version=%d", projectID, projectName, href, syncStatus, etag, version)
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

func TestBuildExplicitRRuleUpdateUntilUsesEndOfDay(t *testing.T) {
	t.Parallel()
	form := map[string][]string{
		"repeat_update": {"1"},
		"repeat_freq":   {"WEEKLY"},
		"repeat_end":    {"until"},
		"repeat_until":  {"2026-03-07"},
	}

	rule := buildExplicitRRuleUpdate(form)
	if rule == nil {
		t.Fatal("expected rrule")
	}
	if *rule != "FREQ=WEEKLY;UNTIL=20260307T235959Z" {
		t.Fatalf("unexpected rule: %q", *rule)
	}
}

func TestTaskUpdateExplicitRecurrenceReplace(t *testing.T) {
	t.Parallel()
	database := openSQLiteForTaskUpdateHandlerTest(t)
	seedTaskUpdateHandlerData(t, database)

	if _, err := database.Conn.ExecContext(context.Background(), `UPDATE tasks SET raw_vtodo = 'BEGIN:VCALENDAR
BEGIN:VTODO
UID:uid-1
SUMMARY:old
DESCRIPTION:old-desc
STATUS:NEEDS-ACTION
RRULE:FREQ=DAILY
END:VTODO
END:VCALENDAR' WHERE id='task-1';`); err != nil {
		t.Fatalf("seed recurrence: %v", err)
	}

	key := bytes.Repeat([]byte{0x64}, 32)
	if err := database.SaveCalDAVCredentials(context.Background(), key, db.CalDAVCredentials{URL: "https://dav.example", Username: "alice", Password: "secret"}); err != nil {
		t.Fatalf("save credentials: %v", err)
	}

	h := TaskUpdate(taskUpdateDependencies{database: database, encryptionKey: key, todos: &stubTaskUpdateTodoClient{updateETag: `"etag-4"`}})
	form := url.Values{
		"expected_version": {"2"},
		"title":            {"updated title"},
		"repeat_update":    {"1"},
		"repeat_freq":      {"WEEKLY"},
		"repeat_interval":  {"2"},
		"repeat_end":       {"count"},
		"repeat_count":     {"5"},
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
	if !bytes.Contains([]byte(rawVTODO), []byte("RRULE:FREQ=WEEKLY;INTERVAL=2;COUNT=5")) {
		t.Fatalf("unexpected raw vtodo: %q", rawVTODO)
	}
}

func TestTaskUpdateDoesNotChangeRecurrenceWithoutExplicitFlag(t *testing.T) {
	t.Parallel()
	database := openSQLiteForTaskUpdateHandlerTest(t)
	seedTaskUpdateHandlerData(t, database)

	if _, err := database.Conn.ExecContext(context.Background(), `UPDATE tasks SET raw_vtodo = 'BEGIN:VCALENDAR
BEGIN:VTODO
UID:uid-1
SUMMARY:old
DESCRIPTION:old-desc
STATUS:NEEDS-ACTION
RRULE:FREQ=MONTHLY;BYDAY=MO
END:VTODO
END:VCALENDAR' WHERE id='task-1';`); err != nil {
		t.Fatalf("seed recurrence: %v", err)
	}

	key := bytes.Repeat([]byte{0x65}, 32)
	if err := database.SaveCalDAVCredentials(context.Background(), key, db.CalDAVCredentials{URL: "https://dav.example", Username: "alice", Password: "secret"}); err != nil {
		t.Fatalf("save credentials: %v", err)
	}

	h := TaskUpdate(taskUpdateDependencies{database: database, encryptionKey: key, todos: &stubTaskUpdateTodoClient{updateETag: `"etag-5"`}})
	form := url.Values{"expected_version": {"2"}, "title": {"new title"}, "repeat_freq": {"YEARLY"}}
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
	if !bytes.Contains([]byte(rawVTODO), []byte("RRULE:FREQ=MONTHLY;BYDAY=MO")) {
		t.Fatalf("expected recurrence unchanged, raw=%q", rawVTODO)
	}
}

func TestTaskUpdateDoesNotChangeComplexRecurrenceWithExplicitFlag(t *testing.T) {
	t.Parallel()
	database := openSQLiteForTaskUpdateHandlerTest(t)
	seedTaskUpdateHandlerData(t, database)

	if _, err := database.Conn.ExecContext(context.Background(), `UPDATE tasks SET raw_vtodo = 'BEGIN:VCALENDAR
BEGIN:VTODO
UID:uid-1
SUMMARY:old
DESCRIPTION:old-desc
STATUS:NEEDS-ACTION
RRULE:FREQ=MONTHLY;BYDAY=MO,TU,WE,TH,FR;BYSETPOS=1
END:VTODO
END:VCALENDAR' WHERE id='task-1';`); err != nil {
		t.Fatalf("seed recurrence: %v", err)
	}

	key := bytes.Repeat([]byte{0x66}, 32)
	if err := database.SaveCalDAVCredentials(context.Background(), key, db.CalDAVCredentials{URL: "https://dav.example", Username: "alice", Password: "secret"}); err != nil {
		t.Fatalf("save credentials: %v", err)
	}

	h := TaskUpdate(taskUpdateDependencies{database: database, encryptionKey: key, todos: &stubTaskUpdateTodoClient{updateETag: `"etag-6"`}})
	form := url.Values{
		"expected_version": {"2"},
		"title":            {"new title"},
		"repeat_update":    {"1"},
		"repeat_freq":      {"YEARLY"},
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
	if !bytes.Contains([]byte(rawVTODO), []byte("RRULE:FREQ=MONTHLY;BYDAY=MO,TU,WE,TH,FR;BYSETPOS=1")) {
		t.Fatalf("expected complex recurrence unchanged, raw=%q", rawVTODO)
	}
	if bytes.Contains([]byte(rawVTODO), []byte("RRULE:FREQ=YEARLY")) {
		t.Fatalf("expected explicit recurrence update to be ignored for complex recurrence, raw=%q", rawVTODO)
	}
}

func TestTaskUpdateDoesNotChangeNonMVPDailyByDayRecurrenceWithExplicitFlag(t *testing.T) {
	t.Parallel()
	database := openSQLiteForTaskUpdateHandlerTest(t)
	seedTaskUpdateHandlerData(t, database)

	if _, err := database.Conn.ExecContext(context.Background(), `UPDATE tasks SET raw_vtodo = 'BEGIN:VCALENDAR
BEGIN:VTODO
UID:uid-1
SUMMARY:old
DESCRIPTION:old-desc
STATUS:NEEDS-ACTION
RRULE:FREQ=DAILY;BYDAY=MO,TU
END:VTODO
END:VCALENDAR' WHERE id='task-1';`); err != nil {
		t.Fatalf("seed recurrence: %v", err)
	}

	key := bytes.Repeat([]byte{0x71}, 32)
	if err := database.SaveCalDAVCredentials(context.Background(), key, db.CalDAVCredentials{URL: "https://dav.example", Username: "alice", Password: "secret"}); err != nil {
		t.Fatalf("save credentials: %v", err)
	}

	h := TaskUpdate(taskUpdateDependencies{database: database, encryptionKey: key, todos: &stubTaskUpdateTodoClient{updateETag: `"etag-8"`}})
	form := url.Values{
		"expected_version": {"2"},
		"title":            {"new title"},
		"repeat_update":    {"1"},
		"repeat_freq":      {"YEARLY"},
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
	if !bytes.Contains([]byte(rawVTODO), []byte("RRULE:FREQ=DAILY;BYDAY=MO,TU")) {
		t.Fatalf("expected non-mvp recurrence unchanged, raw=%q", rawVTODO)
	}
	if bytes.Contains([]byte(rawVTODO), []byte("RRULE:FREQ=YEARLY")) {
		t.Fatalf("expected explicit recurrence update to be ignored for non-mvp recurrence, raw=%q", rawVTODO)
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
    'BEGIN:VCALENDAR
BEGIN:VTODO
UID:uid-1
SUMMARY:old
DESCRIPTION:old-desc
STATUS:NEEDS-ACTION
END:VTODO
END:VCALENDAR',
    'BEGIN:VCALENDAR
BEGIN:VTODO
UID:uid-1
SUMMARY:old
DESCRIPTION:old-desc
STATUS:NEEDS-ACTION
END:VTODO
END:VCALENDAR',
    'home', 'Inbox', 'synced', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
);
`); err != nil {
		t.Fatalf("seed handler update data: %v", err)
	}
}
