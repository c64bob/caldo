package handler

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
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
	r := NewRouter(logger, "X-User", testManifest(t), true, []byte("12345678901234567890123456789012"), database, context.Background(), nil)
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

func TestConflictDetailPageShowsReadableComparison(t *testing.T) {
	t.Parallel()

	database, err := db.OpenSQLite(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	seedConflictData(t, database)
	if _, err := database.Conn.ExecContext(context.Background(), `
UPDATE conflicts
SET base_vtodo='BEGIN:VTODO
SUMMARY:Base title
DESCRIPTION:Base desc
STATUS:NEEDS-ACTION
DUE;VALUE=DATE:20260610
END:VTODO',
    local_vtodo='BEGIN:VTODO
SUMMARY:Local title
DESCRIPTION:Local desc
STATUS:NEEDS-ACTION
DUE;VALUE=DATE:20260611
END:VTODO',
    remote_vtodo='BEGIN:VTODO
SUMMARY:Remote title
DESCRIPTION:Remote desc
STATUS:COMPLETED
DUE;VALUE=DATE:20260612
END:VTODO'
WHERE id='open-1';
`); err != nil {
		t.Fatalf("update conflict data: %v", err)
	}

	logger := logging.New(bytes.NewBuffer(nil), "production", "info")
	r := NewRouter(logger, "X-User", testManifest(t), true, []byte("12345678901234567890123456789012"), database, context.Background(), nil)
	req := httptest.NewRequest(http.MethodGet, "/conflicts/open-1", nil)
	req.Header.Set("X-User", "u")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status: %d body=%s", rr.Code, rr.Body.String())
	}
	body := rr.Body.String()
	for _, want := range []string{
		`data-conflict-comparison`,
		`Base title`,
		`Local title`,
		`Remote title`,
		`Erledigt`,
		`data-conflict-value="local-title"`,
		`data-conflict-resolution`,
		`data-conflict-split-preview`,
		`Beide Versionen behalten`,
		`data-conflict-manual-form`,
		`data-conflict-field-source="title"`,
		`data-conflict-source-option="local"`,
		`type="radio" name="title_source" value="local"`,
		`caldo-conflict-manual-value`,
		`name="project_id"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("response body missing conflict comparison detail %q: %s", want, body)
		}
	}
}

func TestResolveConflictManualSelectsFieldSourcesAndParent(t *testing.T) {
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
	base := "BEGIN:VCALENDAR\r\nBEGIN:VTODO\r\nUID:uid-1\r\nSUMMARY:Base title\r\nDESCRIPTION:Base desc\r\nSTATUS:NEEDS-ACTION\r\nDUE;VALUE=DATE:20260610\r\nPRIORITY:5\r\nCATEGORIES:base\r\nRELATED-TO;RELTYPE=PARENT:parent-base\r\nEND:VTODO\r\nEND:VCALENDAR\r\n"
	local := "BEGIN:VCALENDAR\r\nBEGIN:VTODO\r\nUID:uid-1\r\nSUMMARY:Local title\r\nDESCRIPTION:Local desc\r\nSTATUS:NEEDS-ACTION\r\nDUE;VALUE=DATE:20260611\r\nPRIORITY:1\r\nCATEGORIES:local,shared\r\nRELATED-TO;RELTYPE=PARENT:parent-local\r\nEND:VTODO\r\nEND:VCALENDAR\r\n"
	remote := "BEGIN:VCALENDAR\r\nBEGIN:VTODO\r\nUID:uid-1\r\nSUMMARY:Remote title\r\nDESCRIPTION:Remote desc\r\nSTATUS:COMPLETED\r\nCOMPLETED:20260612T090000Z\r\nDUE;VALUE=DATE:20260612\r\nPRIORITY:9\r\nCATEGORIES:remote\r\nRELATED-TO;RELTYPE=PARENT:parent-remote\r\nEND:VTODO\r\nEND:VCALENDAR\r\n"
	if _, err := database.Conn.Exec(`UPDATE conflicts SET base_vtodo=?, local_vtodo=?, remote_vtodo=? WHERE id='open-1'`, base, local, remote); err != nil {
		t.Fatal(err)
	}
	todos := &stubTaskUpdateTodoClient{updateETag: `"etag-new"`}
	h := ResolveConflict(taskUpdateDependencies{database: database, encryptionKey: key, todos: todos})
	form := url.Values{
		"resolution":         {"manual"},
		"title_source":       {"local"},
		"description_source": {"remote"},
		"due_source":         {"local"},
		"priority_source":    {"remote"},
		"labels_source":      {"local"},
		"status_source":      {"remote"},
		"parent_source":      {"local"},
	}
	req := httptest.NewRequest(http.MethodPost, "/conflicts/open-1/resolve", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("conflictID", "open-1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if todos.updateCalls != 1 || todos.createCalls != 0 || todos.lastHref != "/cal/inbox/uid-1.ics" {
		t.Fatalf("unexpected caldav calls: update=%d create=%d href=%q", todos.updateCalls, todos.createCalls, todos.lastHref)
	}
	var resolved string
	if err := database.Conn.QueryRow(`SELECT resolved_vtodo FROM conflicts WHERE id='open-1'`).Scan(&resolved); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"SUMMARY:Local title",
		"DESCRIPTION:Remote desc",
		"STATUS:COMPLETED",
		"COMPLETED:20260612T090000Z",
		"DUE;VALUE=DATE:20260611",
		"PRIORITY:9",
		"CATEGORIES:local,shared",
		"RELATED-TO;RELTYPE=PARENT:parent-local",
	} {
		if !strings.Contains(resolved, want) {
			t.Fatalf("expected resolved payload to contain %q: %s", want, resolved)
		}
	}
	for _, unwanted := range []string{"SUMMARY:Remote title", "DUE;VALUE=DATE:20260612", "RELATED-TO;RELTYPE=PARENT:parent-remote"} {
		if strings.Contains(resolved, unwanted) {
			t.Fatalf("resolved payload kept unwanted field %q: %s", unwanted, resolved)
		}
	}
}

func TestResolveConflictManualMovesProject(t *testing.T) {
	t.Parallel()
	database, err := db.OpenSQLite(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	seedConflictData(t, database)
	if _, err := database.Conn.ExecContext(context.Background(), `
INSERT INTO projects (id, calendar_href, display_name, sync_strategy, created_at, updated_at)
VALUES ('project-2','/cal/side/','Side','ctag',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP);
`); err != nil {
		t.Fatal(err)
	}
	key := bytes.Repeat([]byte{0x56}, 32)
	if err := database.SaveCalDAVCredentials(context.Background(), key, db.CalDAVCredentials{URL: "https://dav.example", Username: "a", Password: "b"}); err != nil {
		t.Fatal(err)
	}

	todos := &stubTaskUpdateTodoClient{createETag: `"etag-moved"`}
	h := ResolveConflict(taskUpdateDependencies{database: database, encryptionKey: key, todos: todos})
	form := url.Values{
		"resolution": {"manual"},
		"project_id": {"project-2"},
	}
	req := httptest.NewRequest(http.MethodPost, "/conflicts/open-1/resolve", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("conflictID", "open-1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if todos.updateCalls != 0 || todos.createCalls != 1 || todos.deleteCalls != 1 {
		t.Fatalf("unexpected caldav calls: update=%d create=%d delete=%d", todos.updateCalls, todos.createCalls, todos.deleteCalls)
	}
	if todos.lastHref != "/cal/side/uid-1.ics" {
		t.Fatalf("create href=%q", todos.lastHref)
	}
	if todos.lastDeleteHref != "/cal/inbox/uid-1.ics" || todos.lastDeleteETag != `"etag-1"` {
		t.Fatalf("unexpected delete call: href=%q etag=%q", todos.lastDeleteHref, todos.lastDeleteETag)
	}

	var projectID, projectName, href, etag string
	if err := database.Conn.QueryRowContext(context.Background(), `
SELECT project_id, project_name, href, etag
FROM tasks
WHERE id='task-1';
`).Scan(&projectID, &projectName, &href, &etag); err != nil {
		t.Fatal(err)
	}
	if projectID != "project-2" || projectName != "Side" || href != "/cal/side/uid-1.ics" || etag != `"etag-moved"` {
		t.Fatalf("unexpected moved task row: project=%q name=%q href=%q etag=%q", projectID, projectName, href, etag)
	}
	var resolution string
	if err := database.Conn.QueryRow(`SELECT resolution FROM conflicts WHERE id='open-1'`).Scan(&resolution); err != nil {
		t.Fatal(err)
	}
	if resolution != "manual" {
		t.Fatalf("resolution=%q", resolution)
	}
}

func TestResolveConflictLocalRestoresRemoteDeletedTask(t *testing.T) {
	t.Parallel()
	database, err := db.OpenSQLite(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	seedConflictData(t, database)
	if _, err := database.Conn.ExecContext(context.Background(), `
UPDATE conflicts
SET conflict_type='edit_delete', remote_vtodo=NULL
WHERE id='open-1';
`); err != nil {
		t.Fatal(err)
	}
	key := bytes.Repeat([]byte{0x57}, 32)
	if err := database.SaveCalDAVCredentials(context.Background(), key, db.CalDAVCredentials{URL: "https://dav.example", Username: "a", Password: "b"}); err != nil {
		t.Fatal(err)
	}

	todos := &stubTaskUpdateTodoClient{createETag: `"etag-restored"`}
	h := ResolveConflict(taskUpdateDependencies{database: database, encryptionKey: key, todos: todos})
	req := httptest.NewRequest(http.MethodPost, "/conflicts/open-1/resolve", strings.NewReader("resolution=local"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("conflictID", "open-1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if todos.updateCalls != 0 || todos.createCalls != 1 || todos.deleteCalls != 0 {
		t.Fatalf("unexpected caldav calls: update=%d create=%d delete=%d", todos.updateCalls, todos.createCalls, todos.deleteCalls)
	}
	if todos.lastHref != "/cal/inbox/uid-1.ics" {
		t.Fatalf("create href=%q", todos.lastHref)
	}
	var resolution string
	if err := database.Conn.QueryRow(`SELECT resolution FROM conflicts WHERE id='open-1'`).Scan(&resolution); err != nil {
		t.Fatal(err)
	}
	if resolution != "local" {
		t.Fatalf("resolution=%q", resolution)
	}
}

func seedConflictData(t *testing.T, database *db.Database) {
	t.Helper()
	_, err := database.Conn.ExecContext(context.Background(), `
INSERT INTO projects (id, calendar_href, display_name, sync_strategy, created_at, updated_at)
VALUES ('project-1','/cal/inbox/','Inbox','ctag',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP);
INSERT INTO tasks (id, project_id, uid, href, etag, server_version, title, status, raw_vtodo, sync_status, created_at, updated_at)
VALUES ('task-1','project-1','uid-1','/cal/inbox/uid-1.ics','"etag-1"',2,'Task 1','needs-action','BEGIN:VCALENDAR
BEGIN:VTODO
UID:uid-1
SUMMARY:Task 1
STATUS:NEEDS-ACTION
END:VTODO
END:VCALENDAR','conflict',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP);
INSERT INTO conflicts (id, task_id, project_id, conflict_type, created_at, base_vtodo, local_vtodo, remote_vtodo)
VALUES ('open-1','task-1','project-1','field_conflict',CURRENT_TIMESTAMP,'BEGIN:VCALENDAR
BEGIN:VTODO
UID:uid-1
SUMMARY:Base
STATUS:NEEDS-ACTION
END:VTODO
END:VCALENDAR','BEGIN:VCALENDAR
BEGIN:VTODO
UID:uid-1
SUMMARY:Local
STATUS:NEEDS-ACTION
END:VTODO
END:VCALENDAR','BEGIN:VCALENDAR
BEGIN:VTODO
UID:uid-1
SUMMARY:Remote
STATUS:NEEDS-ACTION
END:VTODO
END:VCALENDAR');
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

func TestResolveConflictUsesRemoteETagStoredByFullScanConflict(t *testing.T) {
	t.Parallel()
	database, err := db.OpenSQLite(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	seedConflictData(t, database)
	key := bytes.Repeat([]byte{0x34}, 32)
	if err := database.SaveCalDAVCredentials(context.Background(), key, db.CalDAVCredentials{URL: "https://dav.example", Username: "a", Password: "b"}); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Conn.ExecContext(context.Background(), `
UPDATE tasks
SET etag='"etag-2"', raw_vtodo='BEGIN:VTODO\r\nUID:uid-1\r\nSUMMARY:Local\r\nEND:VTODO\r\n'
WHERE id='task-1';
UPDATE conflicts
SET base_vtodo='BEGIN:VTODO\r\nUID:uid-1\r\nSUMMARY:Base\r\nEND:VTODO\r\n',
    local_vtodo='BEGIN:VTODO\r\nUID:uid-1\r\nSUMMARY:Local\r\nEND:VTODO\r\n',
    remote_vtodo='BEGIN:VTODO\r\nUID:uid-1\r\nSUMMARY:Remote\r\nEND:VTODO\r\n'
WHERE id='open-1';
`); err != nil {
		t.Fatal(err)
	}

	todos := &stubTaskUpdateTodoClient{updateETag: `"etag-3"`}
	h := ResolveConflict(taskUpdateDependencies{database: database, encryptionKey: key, todos: todos})
	req := httptest.NewRequest(http.MethodPost, "/conflicts/open-1/resolve", strings.NewReader("resolution=local"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("conflictID", "open-1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if todos.lastUpdateETag != `"etag-2"` {
		t.Fatalf("expected conflict resolution to use remote etag, got %q", todos.lastUpdateETag)
	}
	var resolvedAt sql.NullString
	if err := database.Conn.QueryRow(`SELECT resolved_at FROM conflicts WHERE id='open-1'`).Scan(&resolvedAt); err != nil {
		t.Fatal(err)
	}
	if !resolvedAt.Valid {
		t.Fatal("expected conflict to be marked resolved")
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
	remote := "BEGIN:VCALENDAR\r\nBEGIN:VTODO\r\nUID:uid-remote\r\nSUMMARY:Remote version\r\nSTATUS:NEEDS-ACTION\r\nRELATED-TO;RELTYPE=PARENT:uid-parent\r\nRELATED-TO:uid-bare-parent\r\nRELATED-TO;RELTYPE=SIBLING:uid-sibling\r\nEND:VTODO\r\nEND:VCALENDAR\r\n"
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
	if strings.Contains(strings.ToUpper(todos.lastRawVTODO), "RELATED-TO;RELTYPE=PARENT") || strings.Contains(todos.lastRawVTODO, "RELATED-TO:uid-bare-parent") {
		t.Fatalf("split payload must not include parent link: %s", todos.lastRawVTODO)
	}
	if !strings.Contains(todos.lastRawVTODO, "RELATED-TO;RELTYPE=SIBLING:uid-sibling") {
		t.Fatalf("split payload should preserve non-parent related-to links: %s", todos.lastRawVTODO)
	}
	if !strings.Contains(todos.lastRawVTODO, "UID:"+splitConflictUID("open-1")) {
		t.Fatalf("split payload uid not rewritten deterministically: %s", todos.lastRawVTODO)
	}
	if todos.lastHref != "/cal/inbox/"+splitConflictUID("open-1")+".ics" {
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

func TestResolveConflictSplitKeepsUnresolvedOnCreateFailure(t *testing.T) {
	t.Parallel()
	database, err := db.OpenSQLite(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	seedConflictData(t, database)
	key := bytes.Repeat([]byte{0x46}, 32)
	if err := database.SaveCalDAVCredentials(context.Background(), key, db.CalDAVCredentials{URL: "https://dav.example", Username: "a", Password: "b"}); err != nil {
		t.Fatal(err)
	}

	todos := &stubTaskUpdateTodoClient{createErr: errors.New("boom")}
	h := ResolveConflict(taskUpdateDependencies{database: database, encryptionKey: key, todos: todos})
	req := httptest.NewRequest(http.MethodPost, "/conflicts/open-1/resolve", strings.NewReader("resolution=split"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("conflictID", "open-1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadGateway {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	assertSingleIntResult(t, database, `SELECT COUNT(*) FROM conflicts WHERE id='open-1' AND resolved_at IS NULL;`, 1)
	assertSingleIntResult(t, database, `SELECT COUNT(*) FROM tasks WHERE project_id='project-1';`, 1)
}

func TestResolveConflictSplitKeepsUnresolvedOnPersistFailure(t *testing.T) {
	t.Parallel()
	database, err := db.OpenSQLite(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	seedConflictData(t, database)
	key := bytes.Repeat([]byte{0x47}, 32)
	if err := database.SaveCalDAVCredentials(context.Background(), key, db.CalDAVCredentials{URL: "https://dav.example", Username: "a", Password: "b"}); err != nil {
		t.Fatal(err)
	}

	todos := &stubTaskUpdateTodoClient{
		createETag: `"etag-split"`,
		onCreate: func() {
			_, _ = database.Conn.ExecContext(context.Background(), `UPDATE tasks SET server_version=server_version+1 WHERE id='task-1';`)
		},
	}
	h := ResolveConflict(taskUpdateDependencies{database: database, encryptionKey: key, todos: todos})
	req := httptest.NewRequest(http.MethodPost, "/conflicts/open-1/resolve", strings.NewReader("resolution=split"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("conflictID", "open-1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if todos.createCalls != 1 {
		t.Fatalf("create calls=%d", todos.createCalls)
	}
	assertSingleIntResult(t, database, `SELECT COUNT(*) FROM conflicts WHERE id='open-1' AND resolved_at IS NULL;`, 1)
	assertSingleIntResult(t, database, `SELECT COUNT(*) FROM tasks WHERE project_id='project-1';`, 1)
}
