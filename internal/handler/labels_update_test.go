package handler

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"caldo/internal/db"
	"github.com/go-chi/chi/v5"
)

func TestLabelRenameUpdatesAffectedTasksViaCalDAV(t *testing.T) {
	t.Parallel()

	database := openLabelUpdateHandlerDB(t)
	seedLabelUpdateHandlerData(t, database)
	key := bytes.Repeat([]byte{0x31}, 32)
	if err := database.SaveCalDAVCredentials(context.Background(), key, db.CalDAVCredentials{URL: "https://dav.example", Username: "alice", Password: "secret"}); err != nil {
		t.Fatalf("save credentials: %v", err)
	}

	stub := &stubTaskUpdateTodoClient{updateETag: `"etag-renamed"`}
	h := LabelRename(labelUpdateDependencies{database: database, encryptionKey: key, todos: stub})
	request := labelMutationRequest(http.MethodPatch, "/labels/label-home", "label-home", url.Values{"name": {"Work"}})
	responseRecorder := httptest.NewRecorder()

	h.ServeHTTP(responseRecorder, request)

	if responseRecorder.Code != http.StatusOK {
		t.Fatalf("unexpected status: got %d body=%q", responseRecorder.Code, responseRecorder.Body.String())
	}
	body := responseRecorder.Body.String()
	if !strings.Contains(body, `data-label-page-success`) || !strings.Contains(body, "label wurde umbenannt") || !strings.Contains(body, "Work") {
		t.Fatalf("expected successful labels page, got %q", body)
	}
	if stub.updateCalls != 2 {
		t.Fatalf("expected two CalDAV task updates, got %d", stub.updateCalls)
	}
	assertSingleIntResult(t, database, `SELECT COUNT(*) FROM labels WHERE name = 'home';`, 0)
	assertSingleIntResult(t, database, `SELECT COUNT(*) FROM labels WHERE name = 'Work' COLLATE NOCASE;`, 1)
	assertSingleIntResult(t, database, `SELECT COUNT(*) FROM task_labels tl JOIN labels l ON l.id = tl.label_id WHERE l.name = 'Work' COLLATE NOCASE;`, 2)
	assertSingleTextResult(t, database, `SELECT label_names FROM tasks WHERE id = 'task-1';`, "Work,STARRED")
	assertSingleTextResult(t, database, `SELECT label_names FROM tasks WHERE id = 'task-2';`, "other,Work")

	var rawVTODO string
	if err := database.Conn.QueryRowContext(context.Background(), `SELECT raw_vtodo FROM tasks WHERE id = 'task-1';`).Scan(&rawVTODO); err != nil {
		t.Fatalf("query raw vtodo: %v", err)
	}
	for _, want := range []string{"CATEGORIES:Work,STARRED", "X-KEEP:1"} {
		if !strings.Contains(rawVTODO, want) {
			t.Fatalf("renamed raw vtodo missing %q in %q", want, rawVTODO)
		}
	}
}

func TestLabelDeleteRemovesLabelFromAffectedTasksViaCalDAV(t *testing.T) {
	t.Parallel()

	database := openLabelUpdateHandlerDB(t)
	seedLabelUpdateHandlerData(t, database)
	key := bytes.Repeat([]byte{0x32}, 32)
	if err := database.SaveCalDAVCredentials(context.Background(), key, db.CalDAVCredentials{URL: "https://dav.example", Username: "alice", Password: "secret"}); err != nil {
		t.Fatalf("save credentials: %v", err)
	}

	stub := &stubTaskUpdateTodoClient{updateETag: `"etag-deleted"`}
	h := LabelDelete(labelUpdateDependencies{database: database, encryptionKey: key, todos: stub})
	request := labelMutationRequest(http.MethodDelete, "/labels/label-home", "label-home", url.Values{"confirmed": {"true"}})
	responseRecorder := httptest.NewRecorder()

	h.ServeHTTP(responseRecorder, request)

	if responseRecorder.Code != http.StatusOK {
		t.Fatalf("unexpected status: got %d body=%q", responseRecorder.Code, responseRecorder.Body.String())
	}
	if !strings.Contains(responseRecorder.Body.String(), `data-label-page-success`) || !strings.Contains(responseRecorder.Body.String(), "label wurde gelöscht") {
		t.Fatalf("expected successful delete page, got %q", responseRecorder.Body.String())
	}
	if stub.updateCalls != 2 {
		t.Fatalf("expected two CalDAV task updates, got %d", stub.updateCalls)
	}
	assertSingleIntResult(t, database, `SELECT COUNT(*) FROM labels WHERE name = 'home';`, 0)
	assertSingleTextResult(t, database, `SELECT COALESCE(label_names, '') FROM tasks WHERE id = 'task-1';`, "STARRED")
	assertSingleTextResult(t, database, `SELECT COALESCE(label_names, '') FROM tasks WHERE id = 'task-2';`, "other")

	var rawVTODO string
	if err := database.Conn.QueryRowContext(context.Background(), `SELECT raw_vtodo FROM tasks WHERE id = 'task-2';`).Scan(&rawVTODO); err != nil {
		t.Fatalf("query raw vtodo: %v", err)
	}
	if strings.Contains(rawVTODO, "home") || !strings.Contains(rawVTODO, "CATEGORIES:other") {
		t.Fatalf("delete raw vtodo did not remove only target label: %q", rawVTODO)
	}
	if err := database.Conn.QueryRowContext(context.Background(), `SELECT raw_vtodo FROM tasks WHERE id = 'task-1';`).Scan(&rawVTODO); err != nil {
		t.Fatalf("query first raw vtodo: %v", err)
	}
	for _, want := range []string{"CATEGORIES:STARRED", "X-KEEP:1"} {
		if !strings.Contains(rawVTODO, want) {
			t.Fatalf("delete raw vtodo did not preserve %q in %q", want, rawVTODO)
		}
	}
}

func TestLabelRenameShowsVisibleErrorWhenCalDAVWriteFails(t *testing.T) {
	t.Parallel()

	database := openLabelUpdateHandlerDB(t)
	seedLabelUpdateHandlerData(t, database)
	key := bytes.Repeat([]byte{0x33}, 32)
	if err := database.SaveCalDAVCredentials(context.Background(), key, db.CalDAVCredentials{URL: "https://dav.example", Username: "alice", Password: "secret"}); err != nil {
		t.Fatalf("save credentials: %v", err)
	}

	stub := &stubTaskUpdateTodoClient{updateErr: errors.New("upstream rejected private task")}
	h := LabelRename(labelUpdateDependencies{database: database, encryptionKey: key, todos: stub})
	request := labelMutationRequest(http.MethodPatch, "/labels/label-home", "label-home", url.Values{"name": {"Work"}})
	responseRecorder := httptest.NewRecorder()

	h.ServeHTTP(responseRecorder, request)

	if responseRecorder.Code != http.StatusOK {
		t.Fatalf("unexpected status: got %d body=%q", responseRecorder.Code, responseRecorder.Body.String())
	}
	body := responseRecorder.Body.String()
	if !strings.Contains(body, `data-label-rename-error`) || !strings.Contains(body, "label konnte nicht vollständig umbenannt werden (0 von 2 Aufgaben aktualisiert)") {
		t.Fatalf("expected visible safe rename error, got %q", body)
	}
	for _, leaked := range []string{"upstream rejected", "private task", "https://dav.example", "secret"} {
		if strings.Contains(body, leaked) {
			t.Fatalf("label error leaked %q in body %q", leaked, body)
		}
	}
	assertSingleTextResult(t, database, `SELECT sync_status FROM tasks WHERE id = 'task-1';`, "error")
	assertSingleTextResult(t, database, `SELECT sync_status FROM tasks WHERE id = 'task-2';`, "synced")
	assertSingleIntResult(t, database, `SELECT COUNT(*) FROM labels WHERE name = 'home' COLLATE NOCASE;`, 1)
}

func TestLabelDeleteRequiresExplicitConfirmation(t *testing.T) {
	t.Parallel()

	database := openLabelUpdateHandlerDB(t)
	seedLabelUpdateHandlerData(t, database)
	key := bytes.Repeat([]byte{0x34}, 32)
	if err := database.SaveCalDAVCredentials(context.Background(), key, db.CalDAVCredentials{URL: "https://dav.example", Username: "alice", Password: "secret"}); err != nil {
		t.Fatalf("save credentials: %v", err)
	}

	stub := &stubTaskUpdateTodoClient{updateETag: `"etag-deleted"`}
	h := LabelDelete(labelUpdateDependencies{database: database, encryptionKey: key, todos: stub})
	request := labelMutationRequest(http.MethodDelete, "/labels/label-home", "label-home", url.Values{"confirmation_name": {"home"}})
	responseRecorder := httptest.NewRecorder()

	h.ServeHTTP(responseRecorder, request)

	if responseRecorder.Code != http.StatusOK {
		t.Fatalf("unexpected status: got %d body=%q", responseRecorder.Code, responseRecorder.Body.String())
	}
	if !strings.Contains(responseRecorder.Body.String(), `data-label-delete-error`) || !strings.Contains(responseRecorder.Body.String(), "löschung muss bestätigt werden") {
		t.Fatalf("expected confirmation error, got %q", responseRecorder.Body.String())
	}
	if !strings.Contains(responseRecorder.Body.String(), `data-label-delete-reopen`) {
		t.Fatalf("expected failed confirmation dialog to reopen, got %q", responseRecorder.Body.String())
	}
	if stub.updateCalls != 0 {
		t.Fatalf("expected no CalDAV writes, got %d", stub.updateCalls)
	}
	assertSingleIntResult(t, database, `SELECT COUNT(*) FROM labels WHERE name = 'home' COLLATE NOCASE;`, 1)
}

func TestLabelDeleteRemovesUnusedLabelWithoutCalDAVWrite(t *testing.T) {
	t.Parallel()

	database := openLabelUpdateHandlerDB(t)
	if _, err := database.Conn.ExecContext(context.Background(), `INSERT INTO labels (id, name, created_at) VALUES ('label-unused', 'unused', CURRENT_TIMESTAMP);`); err != nil {
		t.Fatalf("seed unused label: %v", err)
	}

	stub := &stubTaskUpdateTodoClient{updateETag: `"not-used"`}
	h := LabelDelete(labelUpdateDependencies{database: database, todos: stub})
	request := labelMutationRequest(http.MethodDelete, "/labels/label-unused", "label-unused", url.Values{"confirmed": {"true"}})
	responseRecorder := httptest.NewRecorder()

	h.ServeHTTP(responseRecorder, request)

	if responseRecorder.Code != http.StatusOK || !strings.Contains(responseRecorder.Body.String(), "label wurde gelöscht") {
		t.Fatalf("expected successful unused-label deletion, got status=%d body=%q", responseRecorder.Code, responseRecorder.Body.String())
	}
	if stub.updateCalls != 0 {
		t.Fatalf("expected no CalDAV writes, got %d", stub.updateCalls)
	}
	assertSingleIntResult(t, database, `SELECT COUNT(*) FROM labels WHERE id = 'label-unused';`, 0)
}

func TestLabelDeleteShowsVisibleSafeErrorWhenCalDAVWriteFails(t *testing.T) {
	t.Parallel()

	database := openLabelUpdateHandlerDB(t)
	seedLabelUpdateHandlerData(t, database)
	key := bytes.Repeat([]byte{0x35}, 32)
	if err := database.SaveCalDAVCredentials(context.Background(), key, db.CalDAVCredentials{URL: "https://dav.example", Username: "alice", Password: "secret"}); err != nil {
		t.Fatalf("save credentials: %v", err)
	}

	stub := &stubTaskUpdateTodoClient{updateErr: errors.New("upstream rejected private task")}
	h := LabelDelete(labelUpdateDependencies{database: database, encryptionKey: key, todos: stub})
	request := labelMutationRequest(http.MethodDelete, "/labels/label-home", "label-home", url.Values{"confirmed": {"true"}})
	responseRecorder := httptest.NewRecorder()

	h.ServeHTTP(responseRecorder, request)

	body := responseRecorder.Body.String()
	if responseRecorder.Code != http.StatusOK || !strings.Contains(body, `data-label-delete-error`) || !strings.Contains(body, "label konnte nicht vollständig gelöscht werden (0 von 2 Aufgaben aktualisiert)") {
		t.Fatalf("expected visible delete error, got status=%d body=%q", responseRecorder.Code, body)
	}
	if !strings.Contains(body, `data-label-delete-reopen`) {
		t.Fatalf("expected failed delete dialog to reopen, got %q", body)
	}
	for _, leaked := range []string{"upstream rejected", "private task", "https://dav.example", "secret"} {
		if strings.Contains(body, leaked) {
			t.Fatalf("label error leaked %q in body %q", leaked, body)
		}
	}
	if stub.updateCalls != 1 {
		t.Fatalf("expected one attempted CalDAV write, got %d", stub.updateCalls)
	}
	assertSingleIntResult(t, database, `SELECT COUNT(*) FROM labels WHERE name = 'home' COLLATE NOCASE;`, 1)
}

func openLabelUpdateHandlerDB(t *testing.T) *db.Database {
	t.Helper()

	database, err := db.OpenSQLite(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	return database
}

func labelMutationRequest(method string, target string, labelID string, form url.Values) *http.Request {
	request := httptest.NewRequest(method, target, strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("X-Tab-ID", "tab-label-management")
	routeContext := chi.NewRouteContext()
	routeContext.URLParams.Add("labelID", labelID)
	return request.WithContext(context.WithValue(request.Context(), chi.RouteCtxKey, routeContext))
}

func seedLabelUpdateHandlerData(t *testing.T, database *db.Database) {
	t.Helper()

	rawTaskOne := "BEGIN:VTODO\nUID:uid-1\nSUMMARY:Task 1\nCATEGORIES:home,STARRED\nX-KEEP:1\nEND:VTODO"
	rawTaskTwo := "BEGIN:VTODO\nUID:uid-2\nSUMMARY:Task 2\nCATEGORIES:home,other\nEND:VTODO"
	if _, err := database.Conn.ExecContext(context.Background(), `
INSERT INTO projects (id, calendar_href, display_name, sync_strategy, created_at, updated_at)
VALUES ('project-1', '/cal/work/', 'Work', 'fullscan', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
INSERT INTO labels (id, name, created_at) VALUES
	('label-home', 'home', CURRENT_TIMESTAMP),
	('label-other', 'other', CURRENT_TIMESTAMP);
`); err != nil {
		t.Fatalf("seed label update base data: %v", err)
	}

	if _, err := database.Conn.ExecContext(context.Background(), `
INSERT INTO tasks (
	id, project_id, uid, href, etag, server_version, title, status, raw_vtodo, base_vtodo,
	label_names, project_name, sync_status, created_at, updated_at
) VALUES
	('task-1', 'project-1', 'uid-1', '/cal/work/task-1.ics', '"etag-1"', 3, 'Task 1', 'needs-action',
	 ?, ?, 'home,STARRED', 'Work', 'synced', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
	('task-2', 'project-1', 'uid-2', '/cal/work/task-2.ics', '"etag-2"', 4, 'Task 2', 'completed',
	 ?, ?, 'home,other', 'Work', 'synced', CURRENT_TIMESTAMP, DATETIME(CURRENT_TIMESTAMP, '+1 second'));
`, rawTaskOne, rawTaskOne, rawTaskTwo, rawTaskTwo); err != nil {
		t.Fatalf("seed label update task data: %v", err)
	}

	if _, err := database.Conn.ExecContext(context.Background(), `
INSERT INTO task_labels (task_id, label_id) VALUES
	('task-1', 'label-home'),
	('task-2', 'label-home'),
	('task-2', 'label-other');
`); err != nil {
		t.Fatalf("seed label update data: %v", err)
	}
}
