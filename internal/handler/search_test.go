package handler

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"caldo/internal/db"
	"caldo/internal/logging"
)

func TestSearchRouteReturnsActiveTasksOnly(t *testing.T) {
	t.Parallel()

	database, err := db.OpenSQLite(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() {
		if err := database.Close(); err != nil {
			t.Fatalf("close sqlite: %v", err)
		}
	})

	seedSearchRouteProjectAndTasks(t, database)

	logger := logging.New(bytes.NewBuffer(nil), "production", "info")
	request := httptest.NewRequest(http.MethodGet, "/search?q=rechnung", nil)
	request.Header.Set("X-Forwarded-User", "alice")
	responseRecorder := httptest.NewRecorder()

	NewRouter(logger, "X-Forwarded-User", testManifest(t), true, []byte("12345678901234567890123456789012"), database, context.Background(), nil).ServeHTTP(responseRecorder, request)

	if responseRecorder.Code != http.StatusOK {
		t.Fatalf("unexpected status code: got %d want %d", responseRecorder.Code, http.StatusOK)
	}

	body := responseRecorder.Body.String()
	if !strings.Contains(body, "Überweisung Rechnung") {
		t.Fatalf("response body missing active task title: %q", body)
	}
	if strings.Contains(body, "Überfällige Rechnung") {
		t.Fatalf("response body unexpectedly contains completed task title: %q", body)
	}
	if !strings.Contains(body, "Globale Suche") {
		t.Fatalf("response body missing search heading")
	}
	for _, want := range []string{
		`hx-post="/tasks/task-active/complete"`,
		`name="expected_version" value="5"`,
		"Prüfen",
		"Heute",
		"Finanzen",
		"Büro",
		"dringend",
		"Rechnung Unteraufgabe",
		"Unteraufgabe von Überweisung Rechnung",
		"1 Unteraufgabe",
		`caldo-task-row-subtask`,
		`hx-post="/tasks/task-child/complete"`,
		"P1 Hoch",
		`data-task-favorite-form`,
		`aria-label="Favorit setzen"`,
		"Fehler",
		`data-task-detail-open`,
		`data-task-detail-dialog`,
		`Letzter Schreibversuch ist fehlgeschlagen`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("response body missing task row detail %q: %q", want, body)
		}
	}
	if strings.Contains(body, `data-search-save-filter-form`) {
		t.Fatalf("freetext search must not offer saved filter creation: %q", body)
	}
	if !strings.Contains(body, `rel="noopener noreferrer"`) {
		t.Fatalf("response body missing secure external attachment rel attribute: %q", body)
	}
	if !strings.Contains(body, "Anhang vorhanden (inline/binary)") {
		t.Fatalf("response body missing inline attachment marker: %q", body)
	}
}

func TestSearchResultsRouteRendersLivePartial(t *testing.T) {
	t.Parallel()

	database, err := db.OpenSQLite(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() {
		if err := database.Close(); err != nil {
			t.Fatalf("close sqlite: %v", err)
		}
	})

	seedSearchRouteProjectAndTasks(t, database)

	logger := logging.New(bytes.NewBuffer(nil), "production", "info")
	request := httptest.NewRequest(http.MethodGet, "/search/results?q=rechnung", nil)
	request.Header.Set("X-Forwarded-User", "alice")
	responseRecorder := httptest.NewRecorder()

	NewRouter(logger, "X-Forwarded-User", testManifest(t), true, []byte("12345678901234567890123456789012"), database, context.Background(), nil).ServeHTTP(responseRecorder, request)

	if responseRecorder.Code != http.StatusOK {
		t.Fatalf("unexpected status code: got %d want %d", responseRecorder.Code, http.StatusOK)
	}

	body := responseRecorder.Body.String()
	for _, want := range []string{
		`id="search-live-results"`,
		`data-search-live-results`,
		"Überweisung Rechnung",
		`hx-post="/tasks/task-active/complete"`,
		`name="expected_version" value="5"`,
		"Finanzen",
		"Büro",
		"Rechnung Unteraufgabe",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("response body missing live search detail %q: %q", want, body)
		}
	}
	for _, unwanted := range []string{
		`class="caldo-app-shell"`,
		`<main class="caldo-main"`,
		`class="caldo-topbar-heading"`,
		`id="global-search"`,
	} {
		if strings.Contains(body, unwanted) {
			t.Fatalf("live search response unexpectedly contains full page detail %q: %q", unwanted, body)
		}
	}
}

func TestSearchRouteOmitsBottomCreateForProjectContext(t *testing.T) {
	t.Parallel()

	database, err := db.OpenSQLite(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() {
		if err := database.Close(); err != nil {
			t.Fatalf("close sqlite: %v", err)
		}
	})

	seedSearchRouteProjectAndTasks(t, database)

	logger := logging.New(bytes.NewBuffer(nil), "production", "info")
	request := httptest.NewRequest(http.MethodGet, "/search?q=%23Finanzen", nil)
	request.Header.Set("X-Forwarded-User", "alice")
	responseRecorder := httptest.NewRecorder()

	NewRouter(logger, "X-Forwarded-User", testManifest(t), true, []byte("12345678901234567890123456789012"), database, context.Background(), nil).ServeHTTP(responseRecorder, request)

	if responseRecorder.Code != http.StatusOK {
		t.Fatalf("unexpected status code: got %d want %d", responseRecorder.Code, http.StatusOK)
	}

	body := responseRecorder.Body.String()
	for _, want := range []string{
		`data-search-save-filter-form`,
		`hx-post="/filters"`,
		`name="query" value="#Finanzen"`,
		`data-search-save-filter-name`,
		`data-search-save-filter-query`,
		`Favorisieren`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("response body missing project inline create detail %q: %q", want, body)
		}
	}
	if strings.Contains(body, `class="caldo-inline-create"`) {
		t.Fatalf("search page must not render the bottom task creator: %q", body)
	}
}

func TestSaveFilterForSearchQueryOnlyAllowsValidFilterSyntax(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		query string
		want  bool
	}{
		{name: "empty", query: " ", want: false},
		{name: "freetext", query: "rechnung", want: false},
		{name: "implicit search combination", query: "rechnung #Finanzen", want: false},
		{name: "project", query: "#Finanzen", want: true},
		{name: "label", query: "@Büro", want: true},
		{name: "text", query: "text:rechnung", want: true},
		{name: "boolean", query: "today AND @Büro", want: true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := saveFilterForSearchQuery(tt.query)
			if got.Enabled != tt.want {
				t.Fatalf("Enabled=%v want %v for query %q", got.Enabled, tt.want, tt.query)
			}
			if got.Enabled {
				if got.Query != strings.TrimSpace(tt.query) || !got.IsFavorite {
					t.Fatalf("unexpected save filter view: %#v", got)
				}
			}
		})
	}
}

func TestSearchRouteRendersConflictTaskDirectLink(t *testing.T) {
	t.Parallel()

	database, err := db.OpenSQLite(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() {
		if err := database.Close(); err != nil {
			t.Fatalf("close sqlite: %v", err)
		}
	})

	seedSearchRouteProjectAndTasks(t, database)
	if _, err := database.Conn.ExecContext(context.Background(), `
UPDATE tasks SET sync_status='conflict' WHERE id='task-active';
INSERT INTO conflicts (id, task_id, project_id, conflict_type, created_at, base_vtodo, local_vtodo, remote_vtodo)
VALUES ('conflict-task-active','task-active','project-1','field_conflict',CURRENT_TIMESTAMP,'base','local','remote');
`); err != nil {
		t.Fatalf("seed conflict route data: %v", err)
	}

	logger := logging.New(bytes.NewBuffer(nil), "production", "info")
	request := httptest.NewRequest(http.MethodGet, "/search?q=%C3%BCberweisung", nil)
	request.Header.Set("X-Forwarded-User", "alice")
	responseRecorder := httptest.NewRecorder()

	NewRouter(logger, "X-Forwarded-User", testManifest(t), true, []byte("12345678901234567890123456789012"), database, context.Background(), nil).ServeHTTP(responseRecorder, request)

	if responseRecorder.Code != http.StatusOK {
		t.Fatalf("unexpected status code: got %d want %d", responseRecorder.Code, http.StatusOK)
	}

	body := responseRecorder.Body.String()
	for _, want := range []string{
		`data-task-conflict-link`,
		`href="/conflicts/conflict-task-active"`,
		`Konfliktlösung`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("response body missing conflict link %q: %q", want, body)
		}
	}
	if strings.Contains(body, `data-task-detail-open`) {
		t.Fatalf("conflict task should link directly instead of opening detail: %q", body)
	}
}

func seedSearchRouteProjectAndTasks(t *testing.T, database *db.Database) {
	t.Helper()

	if _, err := database.Conn.Exec(`
INSERT INTO projects (
    id, calendar_href, display_name, sync_strategy, server_version, created_at, updated_at
) VALUES (
    'project-1', '/calendars/work', 'Finanzen', 'fullscan', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
);

INSERT INTO tasks (
    id, project_id, uid, href, etag, server_version, title, description, status, raw_vtodo, base_vtodo,
    label_names, project_name, sync_status, due_date, priority, parent_id, created_at, updated_at
) VALUES
(
    'task-active', 'project-1', 'uid-active', '/calendars/work/task-active.ics', '"etag-active"', 5,
    'Überweisung Rechnung', 'Prüfen', 'needs-action', 'BEGIN:VTODO
UID:uid-active
ATTACH:https://example.com/rechnung.pdf
ATTACH;ENCODING=BASE64;VALUE=BINARY:AAAA
END:VTODO',
    'BEGIN:VTODO
UID:uid-active
ATTACH:https://example.com/rechnung.pdf
ATTACH;ENCODING=BASE64;VALUE=BINARY:AAAA
END:VTODO', 'Büro,dringend', 'Finanzen', 'error', date('now'), 1, NULL, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
),
(
    'task-child', 'project-1', 'uid-child', '/calendars/work/task-child.ics', '"etag-child"', 1,
    'Rechnung Unteraufgabe', 'Kind', 'needs-action', 'BEGIN:VTODO
UID:uid-child
RELATED-TO;RELTYPE=PARENT:uid-active
END:VTODO',
    'BEGIN:VTODO
UID:uid-child
RELATED-TO;RELTYPE=PARENT:uid-active
END:VTODO', 'Büro', 'Finanzen', 'synced', date('now'), NULL, 'task-active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
),
(
    'task-completed', 'project-1', 'uid-completed', '/calendars/work/task-completed.ics', '"etag-completed"', 1,
    'Überfällige Rechnung', 'Archiv', 'completed', 'BEGIN:VTODO\nUID:uid-completed\nEND:VTODO',
    'BEGIN:VTODO\nUID:uid-completed\nEND:VTODO', 'Büro,erledigt', 'Finanzen', 'synced', date('now', '-1 day'), NULL, NULL, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
);
`); err != nil {
		t.Fatalf("seed search route data: %v", err)
	}
}
