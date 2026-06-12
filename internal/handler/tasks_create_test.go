package handler

import (
	"bytes"
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"caldo/internal/caldav"
	"caldo/internal/db"
	"github.com/go-chi/chi/v5"
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

	form := url.Values{"title": {"Buy milk"}, "labels": {"finance,home"}, "priority": {"high"}, "recurrence": {"FREQ=WEEKLY;BYDAY=MO"}, "due_date": {"2026-06-09"}}
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
	if !strings.Contains(stub.raw, "DUE;VALUE=DATE:20260609") {
		t.Fatalf("expected due date in raw payload: %q", stub.raw)
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
	var dueDate string
	var priority int
	var labelNames string
	if err := database.Conn.QueryRowContext(context.Background(), `SELECT sync_status, etag, server_version, date(due_date), priority, label_names FROM tasks LIMIT 1;`).Scan(&syncStatus, &etag, &serverVersion, &dueDate, &priority, &labelNames); err != nil {
		t.Fatalf("query task: %v", err)
	}
	if syncStatus != "synced" || etag != `"etag-1"` || serverVersion != 2 {
		t.Fatalf("unexpected task state: status=%q etag=%q version=%d", syncStatus, etag, serverVersion)
	}
	if dueDate != "2026-06-09" || priority != 1 || labelNames != "finance,home" {
		t.Fatalf("unexpected denormalized task fields: due=%q priority=%d labels=%q", dueDate, priority, labelNames)
	}
}

func TestTaskCreateQuickAddCanCreateProjectBeforeTask(t *testing.T) {
	t.Parallel()
	database := openSQLiteForTaskCreateHandlerTest(t)
	seedTaskCreateHandlerProject(t, database)

	key := bytes.Repeat([]byte{0x12}, 32)
	if err := database.SaveCalDAVCredentials(context.Background(), key, db.CalDAVCredentials{URL: "https://dav.example/caldav", Username: "alice", Password: "secret"}); err != nil {
		t.Fatalf("save credentials: %v", err)
	}

	todos := &stubTaskCreateTodoClient{etag: `"etag-quick"`}
	calendar := &fakeProjectCreateCalendarClient{created: caldav.Calendar{Href: "/cal/work/", DisplayName: "Work"}}
	h := TaskCreate(taskCreateDependencies{database: database, encryptionKey: key, todos: todos, calendar: calendar})

	form := url.Values{
		"title":            {"Plan release"},
		"create_project":   {"1"},
		"project_new_name": {"Work"},
		"labels":           {"urgent,backend"},
		"priority":         {"medium"},
	}
	req := httptest.NewRequest(http.MethodPost, "/tasks", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("unexpected status: got %d body=%q", rr.Code, rr.Body.String())
	}
	if calendar.createCalls != 1 || calendar.displayName != "Work" {
		t.Fatalf("unexpected calendar create: calls=%d name=%q", calendar.createCalls, calendar.displayName)
	}
	if !strings.HasPrefix(todos.href, "/cal/work/") || !strings.Contains(todos.raw, "SUMMARY:Plan release") || !strings.Contains(todos.raw, "CATEGORIES:urgent,backend") || !strings.Contains(todos.raw, "PRIORITY:5") {
		t.Fatalf("unexpected task create payload: href=%q raw=%q", todos.href, todos.raw)
	}

	var projectName, href, labelNames string
	var priority int
	if err := database.Conn.QueryRowContext(context.Background(), `
SELECT project_name, href, label_names, priority
FROM tasks
WHERE title = 'Plan release';
`).Scan(&projectName, &href, &labelNames, &priority); err != nil {
		t.Fatalf("query quick add task: %v", err)
	}
	if projectName != "Work" || !strings.HasPrefix(href, "/cal/work/") || labelNames != "urgent,backend" || priority != 5 {
		t.Fatalf("unexpected quick add task row: project=%q href=%q labels=%q priority=%d", projectName, href, labelNames, priority)
	}
	assertSingleIntResult(t, database, `SELECT COUNT(*) FROM projects WHERE display_name = 'Work' AND calendar_href = '/cal/work/';`, 1)
	assertSingleIntResult(t, database, `SELECT COUNT(*) FROM labels WHERE name IN ('urgent', 'backend');`, 2)
}

func TestTaskCreateQuickAddUsesExistingProjectInsteadOfDuplicateCreate(t *testing.T) {
	t.Parallel()
	database := openSQLiteForTaskCreateHandlerTest(t)
	seedTaskCreateHandlerProject(t, database)

	key := bytes.Repeat([]byte{0x13}, 32)
	if err := database.SaveCalDAVCredentials(context.Background(), key, db.CalDAVCredentials{URL: "https://dav.example/caldav", Username: "alice", Password: "secret"}); err != nil {
		t.Fatalf("save credentials: %v", err)
	}
	if _, err := database.Conn.ExecContext(context.Background(), `
INSERT INTO projects (id, calendar_href, display_name, sync_strategy, created_at, updated_at)
VALUES ('project-work', '/cal/work/', 'Work', 'fullscan', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
`); err != nil {
		t.Fatalf("seed existing project: %v", err)
	}

	todos := &stubTaskCreateTodoClient{etag: `"etag-quick"`}
	calendar := &fakeProjectCreateCalendarClient{created: caldav.Calendar{Href: "/cal/duplicate/", DisplayName: "Work"}}
	h := TaskCreate(taskCreateDependencies{database: database, encryptionKey: key, todos: todos, calendar: calendar})

	form := url.Values{
		"title":            {"Plan release"},
		"create_project":   {"1"},
		"project_new_name": {"Work"},
	}
	req := httptest.NewRequest(http.MethodPost, "/tasks", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("unexpected status: got %d body=%q", rr.Code, rr.Body.String())
	}
	if calendar.createCalls != 0 {
		t.Fatalf("expected existing project to skip remote create, got %d calls", calendar.createCalls)
	}
	if !strings.HasPrefix(todos.href, "/cal/work/") {
		t.Fatalf("expected existing work project href, got %q", todos.href)
	}
	assertSingleIntResult(t, database, `SELECT COUNT(*) FROM projects WHERE display_name = 'Work';`, 1)
}

func TestTaskCreateQuickAddUsesSelectedProjectSuggestion(t *testing.T) {
	t.Parallel()
	database := openSQLiteForTaskCreateHandlerTest(t)
	seedTaskCreateHandlerProject(t, database)

	key := bytes.Repeat([]byte{0x14}, 32)
	if err := database.SaveCalDAVCredentials(context.Background(), key, db.CalDAVCredentials{URL: "https://dav.example/caldav", Username: "alice", Password: "secret"}); err != nil {
		t.Fatalf("save credentials: %v", err)
	}
	if _, err := database.Conn.ExecContext(context.Background(), `
INSERT INTO projects (id, calendar_href, display_name, sync_strategy, created_at, updated_at)
VALUES ('project-work', '/cal/work/', 'Work', 'fullscan', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
`); err != nil {
		t.Fatalf("seed suggested project: %v", err)
	}

	todos := &stubTaskCreateTodoClient{etag: `"etag-suggestion"`}
	calendar := &fakeProjectCreateCalendarClient{created: caldav.Calendar{Href: "/cal/new-work/", DisplayName: "Wrok"}}
	h := TaskCreate(taskCreateDependencies{database: database, encryptionKey: key, todos: todos, calendar: calendar})

	form := url.Values{
		"title":             {"Plan release"},
		"project_selection": {"existing:project-work"},
		"project_new_name":  {"Wrok"},
	}
	req := httptest.NewRequest(http.MethodPost, "/tasks", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("unexpected status: got %d body=%q", rr.Code, rr.Body.String())
	}
	if calendar.createCalls != 0 {
		t.Fatalf("expected selected suggestion to skip remote project create, got %d calls", calendar.createCalls)
	}
	if !strings.HasPrefix(todos.href, "/cal/work/") {
		t.Fatalf("expected task in selected work project, got href %q", todos.href)
	}
	var projectName string
	if err := database.Conn.QueryRowContext(context.Background(), `SELECT project_name FROM tasks WHERE title = 'Plan release';`).Scan(&projectName); err != nil {
		t.Fatalf("query selected suggestion task: %v", err)
	}
	if projectName != "Work" {
		t.Fatalf("unexpected project name: got %q want Work", projectName)
	}
}

func TestTaskCreateQuickAddRequiresExplicitUnknownProjectSelection(t *testing.T) {
	t.Parallel()
	database := openSQLiteForTaskCreateHandlerTest(t)
	seedTaskCreateHandlerProject(t, database)

	h := TaskCreate(taskCreateDependencies{database: database})

	form := url.Values{
		"title":            {"Plan release"},
		"project_new_name": {"Wrok"},
	}
	req := httptest.NewRequest(http.MethodPost, "/tasks", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status: got %d body=%q", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "project selection is required") {
		t.Fatalf("expected project selection error, got %q", rr.Body.String())
	}
	assertSingleIntResult(t, database, `SELECT COUNT(*) FROM tasks WHERE title = 'Plan release';`, 0)
}

func TestTaskCreateRejectsRecurrenceWithCRLF(t *testing.T) {
	t.Parallel()
	database := openSQLiteForTaskCreateHandlerTest(t)
	seedTaskCreateHandlerProject(t, database)

	key := bytes.Repeat([]byte{0x55}, 32)
	if err := database.SaveCalDAVCredentials(context.Background(), key, db.CalDAVCredentials{URL: "https://dav.example", Username: "alice", Password: "secret"}); err != nil {
		t.Fatalf("save credentials: %v", err)
	}

	stub := &stubTaskCreateTodoClient{etag: `"etag-1"`}
	h := TaskCreate(taskCreateDependencies{database: database, encryptionKey: key, todos: stub})

	form := url.Values{"title": {"Buy milk"}, "recurrence": {"FREQ=DAILY\r\nATTENDEE:mailto:evil@example.com"}}
	req := httptest.NewRequest(http.MethodPost, "/tasks", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status: got %d body=%q", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "recurrence is invalid") {
		t.Fatalf("expected recurrence validation error, got %q", rr.Body.String())
	}
	if stub.raw != "" || stub.href != "" {
		t.Fatalf("invalid recurrence must not call caldav: href=%q raw=%q", stub.href, stub.raw)
	}
	assertSingleIntResult(t, database, `SELECT COUNT(*) FROM tasks;`, 0)
}

func TestTaskCreateRejectsMalformedRecurrence(t *testing.T) {
	t.Parallel()
	database := openSQLiteForTaskCreateHandlerTest(t)
	seedTaskCreateHandlerProject(t, database)

	key := bytes.Repeat([]byte{0x56}, 32)
	if err := database.SaveCalDAVCredentials(context.Background(), key, db.CalDAVCredentials{URL: "https://dav.example", Username: "alice", Password: "secret"}); err != nil {
		t.Fatalf("save credentials: %v", err)
	}

	stub := &stubTaskCreateTodoClient{etag: `"etag-1"`}
	h := TaskCreate(taskCreateDependencies{database: database, encryptionKey: key, todos: stub})

	form := url.Values{"title": {"Buy milk"}, "recurrence": {"COUNT=2"}}
	req := httptest.NewRequest(http.MethodPost, "/tasks", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status: got %d body=%q", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "recurrence is invalid") {
		t.Fatalf("expected recurrence validation error, got %q", rr.Body.String())
	}
	if stub.raw != "" || stub.href != "" {
		t.Fatalf("invalid recurrence must not call caldav: href=%q raw=%q", stub.href, stub.raw)
	}
	assertSingleIntResult(t, database, `SELECT COUNT(*) FROM tasks;`, 0)
}

func TestTaskCreatePreservesComplexQuickAddRecurrence(t *testing.T) {
	t.Parallel()
	database := openSQLiteForTaskCreateHandlerTest(t)
	seedTaskCreateHandlerProject(t, database)

	key := bytes.Repeat([]byte{0x57}, 32)
	if err := database.SaveCalDAVCredentials(context.Background(), key, db.CalDAVCredentials{URL: "https://dav.example", Username: "alice", Password: "secret"}); err != nil {
		t.Fatalf("save credentials: %v", err)
	}

	stub := &stubTaskCreateTodoClient{etag: `"etag-1"`}
	h := TaskCreate(taskCreateDependencies{database: database, encryptionKey: key, todos: stub})

	complexRRule := "FREQ=MONTHLY;BYDAY=MO,TU,WE,TH,FR;BYSETPOS=1"
	form := url.Values{"title": {"Review reports"}, "recurrence": {complexRRule}}
	req := httptest.NewRequest(http.MethodPost, "/tasks", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("unexpected status: got %d body=%q", rr.Code, rr.Body.String())
	}
	if !strings.Contains(stub.raw, "RRULE:"+complexRRule) {
		t.Fatalf("expected complex recurrence preserved in raw payload: %q", stub.raw)
	}
	var rrule string
	if err := database.Conn.QueryRowContext(context.Background(), `SELECT rrule FROM tasks WHERE title = 'Review reports';`).Scan(&rrule); err != nil {
		t.Fatalf("query recurrence: %v", err)
	}
	if rrule != complexRRule {
		t.Fatalf("unexpected stored recurrence: got %q want %q", rrule, complexRRule)
	}
}

func TestTaskCreateRejectsParentFieldOnRootCreate(t *testing.T) {
	t.Parallel()
	database := openSQLiteForTaskCreateHandlerTest(t)
	seedTaskCreateHandlerProject(t, database)
	key := bytes.Repeat([]byte{0x11}, 32)
	_ = database.SaveCalDAVCredentials(context.Background(), key, db.CalDAVCredentials{URL: "https://dav.example", Username: "alice", Password: "secret"})
	stub := &stubTaskCreateTodoClient{etag: `"etag-1"`}
	h := TaskCreate(taskCreateDependencies{database: database, encryptionKey: key, todos: stub})
	form := url.Values{"title": {"Root"}, "parent_task_id": {"task-1"}}
	req := httptest.NewRequest(http.MethodPost, "/tasks", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status: %d", rr.Code)
	}
	if stub.raw != "" || stub.href != "" {
		t.Fatalf("root create with parent field must not call caldav: href=%q raw=%q", stub.href, stub.raw)
	}
	var count int
	if err := database.Conn.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM tasks;`).Scan(&count); err != nil {
		t.Fatalf("count tasks: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected no local task rows, got %d", count)
	}
}

func TestTaskCreateSubtaskSetsParentLinkAndParentID(t *testing.T) {
	t.Parallel()
	database := openSQLiteForTaskCreateHandlerTest(t)
	seedTaskCreateHandlerProject(t, database)
	seedTaskCreateParentTask(t, database, "task-parent", "", "uid-parent")
	key := bytes.Repeat([]byte{0x61}, 32)
	_ = database.SaveCalDAVCredentials(context.Background(), key, db.CalDAVCredentials{URL: "https://dav.example", Username: "alice", Password: "secret"})
	stub := &stubTaskCreateTodoClient{etag: `"etag-sub"`}
	h := TaskCreateSubtask(taskCreateDependencies{database: database, encryptionKey: key, todos: stub})
	form := url.Values{"title": {"Child task"}}
	req := httptest.NewRequest(http.MethodPost, "/tasks/task-parent/subtasks", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("taskID", "task-parent")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("unexpected status: got %d body=%q", rr.Code, rr.Body.String())
	}
	if !strings.Contains(stub.raw, "RELATED-TO;RELTYPE=PARENT:uid-parent") {
		t.Fatalf("expected parent uid link in payload: %q", stub.raw)
	}
	var parentID sql.NullString
	var href string
	var etag string
	var syncStatus string
	var serverVersion int
	var rawVTODO string
	if err := database.Conn.QueryRowContext(context.Background(), `
SELECT parent_id, href, etag, sync_status, server_version, raw_vtodo
FROM tasks
WHERE title = 'Child task';
`).Scan(&parentID, &href, &etag, &syncStatus, &serverVersion, &rawVTODO); err != nil {
		t.Fatalf("query subtask parent: %v", err)
	}
	if !parentID.Valid || parentID.String != "task-parent" {
		t.Fatalf("unexpected parent id: %#v", parentID)
	}
	if href != stub.href || !strings.HasPrefix(href, "/cal/inbox/") || !strings.HasSuffix(href, ".ics") {
		t.Fatalf("unexpected subtask href: db=%q caldav=%q", href, stub.href)
	}
	if etag != `"etag-sub"` || syncStatus != "synced" || serverVersion != 2 {
		t.Fatalf("unexpected subtask sync state: etag=%q status=%q version=%d", etag, syncStatus, serverVersion)
	}
	if rawVTODO != stub.raw {
		t.Fatalf("local subtask raw must match caldav payload:\nlocal=%q\nremote=%q", rawVTODO, stub.raw)
	}
}

func TestTaskCreateSubtaskRejectsNestedSubtask(t *testing.T) {
	t.Parallel()
	database := openSQLiteForTaskCreateHandlerTest(t)
	seedTaskCreateHandlerProject(t, database)
	seedTaskCreateParentTask(t, database, "task-parent", "", "uid-parent")
	seedTaskCreateParentTask(t, database, "task-child", "task-parent", "uid-child")
	key := bytes.Repeat([]byte{0x62}, 32)
	_ = database.SaveCalDAVCredentials(context.Background(), key, db.CalDAVCredentials{URL: "https://dav.example", Username: "alice", Password: "secret"})
	stub := &stubTaskCreateTodoClient{etag: `"etag-grandchild"`}
	h := TaskCreateSubtask(taskCreateDependencies{database: database, encryptionKey: key, todos: stub})
	form := url.Values{"title": {"Grandchild task"}}
	req := httptest.NewRequest(http.MethodPost, "/tasks/task-child/subtasks", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("taskID", "task-child")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusConflict {
		t.Fatalf("unexpected status: got %d body=%q", rr.Code, rr.Body.String())
	}
	if stub.raw != "" || stub.href != "" {
		t.Fatalf("nested subtask create must not call caldav: href=%q raw=%q", stub.href, stub.raw)
	}
	var count int
	if err := database.Conn.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM tasks WHERE title = 'Grandchild task';`).Scan(&count); err != nil {
		t.Fatalf("count nested subtasks: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected no nested subtask row, got %d", count)
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

func seedTaskCreateParentTask(t *testing.T, database *db.Database, id string, parentID string, uid string) {
	t.Helper()
	raw := "BEGIN:VCALENDAR\nBEGIN:VTODO\nUID:" + uid + "\nSUMMARY:Parent\nSTATUS:NEEDS-ACTION\nEND:VTODO\nEND:VCALENDAR"
	_, err := database.Conn.ExecContext(context.Background(), `
INSERT INTO tasks (id, project_id, uid, href, etag, server_version, title, status, parent_id, raw_vtodo, base_vtodo, project_name, sync_status, created_at, updated_at)
VALUES (?, 'project-default', ?, ?, '"etag"', 2, 'Parent', 'needs-action', ?, ?, ?, 'Inbox', 'synced', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
`, id, uid, "/cal/inbox/"+uid+".ics", nullable(parentID), raw, raw)
	if err != nil {
		t.Fatalf("seed parent task: %v", err)
	}
}

func nullable(value string) interface{} {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
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
