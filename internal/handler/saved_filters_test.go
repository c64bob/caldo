package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"caldo/internal/db"
	"github.com/go-chi/chi/v5"
)

func TestSavedFilterCreatePersistsLocalFilter(t *testing.T) {
	t.Parallel()

	database := openSavedFilterHandlerTestDB(t)
	h := SavedFilterCreate(savedFilterDependencies{database: database})

	form := url.Values{"name": {"Heute Fokus"}, "query": {"today AND @urgent"}, "favorite": {"1"}}
	request := httptest.NewRequest(http.MethodPost, "/filters", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	responseRecorder := httptest.NewRecorder()

	h(responseRecorder, request)

	if responseRecorder.Code != http.StatusCreated {
		t.Fatalf("unexpected status code: got %d want %d", responseRecorder.Code, http.StatusCreated)
	}
	body := responseRecorder.Body.String()
	if !strings.Contains(body, `data-saved-filter-list`) || !strings.Contains(body, "Heute Fokus") || !strings.Contains(body, "Favorit") {
		t.Fatalf("expected refreshed filters page with created filter, got %q", body)
	}

	var count int
	if err := database.Conn.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM saved_filters WHERE name = 'Heute Fokus' AND query = 'today AND @urgent' AND is_favorite = 1;`).Scan(&count); err != nil {
		t.Fatalf("count saved filters: %v", err)
	}
	if count != 1 {
		t.Fatalf("unexpected saved filter count: got %d want 1", count)
	}
}

func TestSavedFilterCreateRejectsMissingFields(t *testing.T) {
	t.Parallel()

	database := openSavedFilterHandlerTestDB(t)
	h := SavedFilterCreate(savedFilterDependencies{database: database})

	form := url.Values{"name": {"   "}, "query": {"today"}}
	request := httptest.NewRequest(http.MethodPost, "/filters", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	responseRecorder := httptest.NewRecorder()

	h(responseRecorder, request)

	if responseRecorder.Code != http.StatusOK {
		t.Fatalf("unexpected status code: got %d want %d", responseRecorder.Code, http.StatusOK)
	}
	if body := responseRecorder.Body.String(); !strings.Contains(body, `data-saved-filter-create-error`) || !strings.Contains(body, "filtername ist erforderlich") {
		t.Fatalf("expected validation error, got %q", body)
	}
}

func TestSavedFilterCreateRejectsInvalidQuery(t *testing.T) {
	t.Parallel()

	database := openSavedFilterHandlerTestDB(t)
	h := SavedFilterCreate(savedFilterDependencies{database: database})

	form := url.Values{"name": {"Broken"}, "query": {"today AND ("}, "favorite": {"1"}}
	request := httptest.NewRequest(http.MethodPost, "/filters", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	responseRecorder := httptest.NewRecorder()

	h(responseRecorder, request)

	if responseRecorder.Code != http.StatusOK {
		t.Fatalf("unexpected status code: got %d want %d", responseRecorder.Code, http.StatusOK)
	}
	body := responseRecorder.Body.String()
	for _, want := range []string{
		`data-saved-filter-create-error`,
		"filterquery ist ungültig",
		`value="Broken"`,
		`value="today AND ("`,
		`checked`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected invalid query create state %q in %q", want, body)
		}
	}
	var count int
	if err := database.Conn.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM saved_filters;`).Scan(&count); err != nil {
		t.Fatalf("count saved filters: %v", err)
	}
	if count != 0 {
		t.Fatalf("invalid filter should not be persisted, got %d rows", count)
	}
}

func TestSavedFilterUpdateUsesExpectedVersion(t *testing.T) {
	t.Parallel()

	database := openSavedFilterHandlerTestDB(t)
	filter := seedSavedFilter(t, database, "filter-1", "Heute", "today", false, 1)
	h := SavedFilterUpdate(savedFilterDependencies{database: database})

	form := url.Values{"expected_version": {"1"}, "name": {"Heute Fokus"}, "query": {"today AND @urgent"}, "favorite": {"1"}}
	request := httptest.NewRequest(http.MethodPatch, "/filters/"+filter.ID, strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	responseRecorder := httptest.NewRecorder()

	h(responseRecorder, request.WithContext(withFilterID(request.Context(), filter.ID)))

	if responseRecorder.Code != http.StatusOK {
		t.Fatalf("unexpected status code: got %d want %d", responseRecorder.Code, http.StatusOK)
	}
	var name, filterQuery string
	var favorite, version int
	if err := database.Conn.QueryRowContext(context.Background(), `SELECT name, query, is_favorite, server_version FROM saved_filters WHERE id = ?;`, filter.ID).Scan(&name, &filterQuery, &favorite, &version); err != nil {
		t.Fatalf("load saved filter: %v", err)
	}
	if name != "Heute Fokus" || filterQuery != "today AND @urgent" || favorite != 1 || version != 2 {
		t.Fatalf("unexpected saved filter row: name=%q query=%q favorite=%d version=%d", name, filterQuery, favorite, version)
	}
}

func TestSavedFilterUpdateRejectsInvalidQuery(t *testing.T) {
	t.Parallel()

	database := openSavedFilterHandlerTestDB(t)
	filter := seedSavedFilter(t, database, "filter-1", "Heute", "today", true, 1)
	h := SavedFilterUpdate(savedFilterDependencies{database: database})

	form := url.Values{"expected_version": {"1"}, "name": {"Broken"}, "query": {"today AND ("}}
	request := httptest.NewRequest(http.MethodPatch, "/filters/"+filter.ID, strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	responseRecorder := httptest.NewRecorder()

	h(responseRecorder, request.WithContext(withFilterID(request.Context(), filter.ID)))

	if responseRecorder.Code != http.StatusOK {
		t.Fatalf("unexpected status code: got %d want %d", responseRecorder.Code, http.StatusOK)
	}
	body := responseRecorder.Body.String()
	for _, want := range []string{
		`data-saved-filter-edit-error`,
		"filterquery ist ungültig",
		`value="Broken"`,
		`value="today AND ("`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected invalid query edit state %q in %q", want, body)
		}
	}
	var name, filterQuery string
	var favorite, version int
	if err := database.Conn.QueryRowContext(context.Background(), `SELECT name, query, is_favorite, server_version FROM saved_filters WHERE id = ?;`, filter.ID).Scan(&name, &filterQuery, &favorite, &version); err != nil {
		t.Fatalf("load saved filter: %v", err)
	}
	if name != "Heute" || filterQuery != "today" || favorite != 1 || version != 1 {
		t.Fatalf("invalid update should not persist: name=%q query=%q favorite=%d version=%d", name, filterQuery, favorite, version)
	}
}

func TestSavedFilterUpdateRendersVersionConflict(t *testing.T) {
	t.Parallel()

	database := openSavedFilterHandlerTestDB(t)
	filter := seedSavedFilter(t, database, "filter-1", "Heute", "today", true, 2)
	h := SavedFilterUpdate(savedFilterDependencies{database: database})

	form := url.Values{"expected_version": {"1"}, "name": {"Heute Fokus"}, "query": {"today AND @urgent"}}
	request := httptest.NewRequest(http.MethodPatch, "/filters/"+filter.ID, strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	responseRecorder := httptest.NewRecorder()

	h(responseRecorder, request.WithContext(withFilterID(request.Context(), filter.ID)))

	if responseRecorder.Code != http.StatusOK {
		t.Fatalf("unexpected status code: got %d want %d", responseRecorder.Code, http.StatusOK)
	}
	if body := responseRecorder.Body.String(); !strings.Contains(body, `data-saved-filter-edit-error`) || !strings.Contains(body, "filter wurde zwischenzeitlich geändert") || !strings.Contains(body, `value="Heute Fokus"`) {
		t.Fatalf("expected visible version conflict, got %q", body)
	}
}

func TestSavedFilterDeleteUsesExpectedVersion(t *testing.T) {
	t.Parallel()

	database := openSavedFilterHandlerTestDB(t)
	filter := seedSavedFilter(t, database, "filter-1", "Heute", "today", false, 1)
	h := SavedFilterDelete(savedFilterDependencies{database: database})

	form := url.Values{"expected_version": {"1"}}
	request := httptest.NewRequest(http.MethodDelete, "/filters/"+filter.ID, strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	responseRecorder := httptest.NewRecorder()

	h(responseRecorder, request.WithContext(withFilterID(request.Context(), filter.ID)))

	if responseRecorder.Code != http.StatusOK {
		t.Fatalf("unexpected status code: got %d want %d", responseRecorder.Code, http.StatusOK)
	}
	if strings.Contains(responseRecorder.Body.String(), `data-saved-filter-list`) {
		t.Fatalf("expected refreshed page without deleted filter, got %q", responseRecorder.Body.String())
	}
	var count int
	if err := database.Conn.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM saved_filters WHERE id = ?;`, filter.ID).Scan(&count); err != nil {
		t.Fatalf("count saved filters: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected filter to be deleted, got %d rows", count)
	}
}

func TestSavedFilterTasksRendersMatchingTasks(t *testing.T) {
	t.Parallel()

	database := openSavedFilterHandlerTestDB(t)
	seedSavedFilterTaskData(t, database)
	h := SavedFilterTasks(savedFilterDependencies{
		database: database,
		now:      func() time.Time { return time.Date(2026, 4, 28, 12, 0, 0, 0, time.UTC) },
	})

	request := httptest.NewRequest(http.MethodGet, "/filters/filter-today", nil)
	responseRecorder := httptest.NewRecorder()

	h(responseRecorder, request.WithContext(withFilterID(request.Context(), "filter-today")))

	if responseRecorder.Code != http.StatusOK {
		t.Fatalf("unexpected status code: got %d want %d", responseRecorder.Code, http.StatusOK)
	}
	body := responseRecorder.Body.String()
	if !strings.Contains(body, "Heute Büro") || !strings.Contains(body, "Heute Aufgabe") || !strings.Contains(body, "Heute Unteraufgabe") || strings.Contains(body, `data-task-id="task-tomorrow"`) {
		t.Fatalf("unexpected saved filter task page: %q", body)
	}
	for _, want := range []string{"Unteraufgabe von Morgen Aufgabe", `data-task-parent-open`, `data-parent-task-id="task-tomorrow"`} {
		if !strings.Contains(body, want) {
			t.Fatalf("saved filter orphaned child missing %q: %q", want, body)
		}
	}
	if strings.Contains(body, `caldo-task-row-subtask`) {
		t.Fatalf("saved filter child must not be indented without its parent: %q", body)
	}
}

func TestSavedFilterTasksInvalidQueryRendersEmptyResult(t *testing.T) {
	t.Parallel()

	database := openSavedFilterHandlerTestDB(t)
	seedSavedFilterTaskData(t, database)
	h := SavedFilterTasks(savedFilterDependencies{
		database: database,
		now:      func() time.Time { return time.Date(2026, 4, 28, 12, 0, 0, 0, time.UTC) },
	})

	request := httptest.NewRequest(http.MethodGet, "/filters/filter-invalid", nil)
	responseRecorder := httptest.NewRecorder()

	h(responseRecorder, request.WithContext(withFilterID(request.Context(), "filter-invalid")))

	if responseRecorder.Code != http.StatusOK {
		t.Fatalf("unexpected status code: got %d want %d", responseRecorder.Code, http.StatusOK)
	}
	body := responseRecorder.Body.String()
	if !strings.Contains(body, "Broken") || !strings.Contains(body, "Keine Aufgaben für diesen Filter.") || strings.Contains(body, "Heute Aufgabe") {
		t.Fatalf("expected empty saved filter task page, got %q", body)
	}
}

func openSavedFilterHandlerTestDB(t *testing.T) *db.Database {
	t.Helper()

	database, err := db.OpenSQLite(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	return database
}

func seedSavedFilter(t *testing.T, database *db.Database, id string, name string, filterQuery string, favorite bool, version int) db.SavedFilter {
	t.Helper()

	fav := 0
	if favorite {
		fav = 1
	}
	if _, err := database.Conn.ExecContext(context.Background(), `
INSERT INTO saved_filters (id, name, query, is_favorite, server_version, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
`, id, name, filterQuery, fav, version); err != nil {
		t.Fatalf("seed saved filter: %v", err)
	}
	return db.SavedFilter{ID: id, Name: name, Query: filterQuery, IsFavorite: favorite, ServerVersion: version}
}

func seedSavedFilterTaskData(t *testing.T, database *db.Database) {
	t.Helper()

	if _, err := database.Conn.ExecContext(context.Background(), `
INSERT INTO projects (id, calendar_href, display_name, sync_strategy, created_at, updated_at)
VALUES ('project-1', '/cal/work/', 'Work', 'fullscan', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
INSERT INTO tasks (
	id, project_id, uid, href, etag, server_version, title, description, status, raw_vtodo, base_vtodo,
	priority, label_names, project_name, sync_status, due_date, parent_id, created_at, updated_at
) VALUES
(
	'task-today', 'project-1', 'uid-today', '/cal/work/today.ics', '"etag-1"', 1,
	'Heute Aufgabe', '', 'needs-action', 'BEGIN:VTODO\nUID:uid-today\nEND:VTODO',
	'BEGIN:VTODO\nUID:uid-today\nEND:VTODO', NULL, 'Büro,urgent', 'Work', 'synced', '2026-04-28', NULL, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
),
(
	'task-tomorrow', 'project-1', 'uid-tomorrow', '/cal/work/tomorrow.ics', '"etag-2"', 1,
	'Morgen Aufgabe', '', 'needs-action', 'BEGIN:VTODO\nUID:uid-tomorrow\nEND:VTODO',
	'BEGIN:VTODO\nUID:uid-tomorrow\nEND:VTODO', NULL, 'Büro,urgent', 'Work', 'synced', '2026-04-29', NULL, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
),
(
	'task-today-child', 'project-1', 'uid-today-child', '/cal/work/today-child.ics', '"etag-3"', 1,
	'Heute Unteraufgabe', '', 'needs-action', 'BEGIN:VTODO\nUID:uid-today-child\nRELATED-TO;RELTYPE=PARENT:uid-tomorrow\nEND:VTODO',
	'BEGIN:VTODO\nUID:uid-today-child\nRELATED-TO;RELTYPE=PARENT:uid-tomorrow\nEND:VTODO', NULL, 'urgent', 'Work', 'synced', '2026-04-28', 'task-tomorrow', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
);
INSERT INTO saved_filters (id, name, query, is_favorite, created_at, updated_at)
VALUES
	('filter-today', 'Heute Büro', 'today AND @urgent', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
	('filter-invalid', 'Broken', 'today AND (', 0, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
`); err != nil {
		t.Fatalf("seed saved filter tasks: %v", err)
	}
}

func withFilterID(ctx context.Context, filterID string) context.Context {
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("filterID", filterID)
	return context.WithValue(ctx, chi.RouteCtxKey, routeCtx)
}
