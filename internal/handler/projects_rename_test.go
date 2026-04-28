package handler

import (
	"context"
	"errors"
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

type fakeProjectRenameCalendarClient struct {
	renamed     caldav.Calendar
	renameErr   error
	renameCalls int
}

func (f *fakeProjectRenameCalendarClient) RenameCalendar(_ context.Context, _ caldav.Credentials, _ string, _ string) (caldav.Calendar, error) {
	f.renameCalls++
	if f.renameErr != nil {
		return caldav.Calendar{}, f.renameErr
	}
	return f.renamed, nil
}

func TestProjectRenamePersistsAfterRemoteSuccess(t *testing.T) {
	t.Parallel()

	database, err := db.OpenSQLite(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	if err := database.SaveCalDAVCredentials(context.Background(), []byte("12345678901234567890123456789012"), db.CalDAVCredentials{
		URL: "https://example.test/caldav", Username: "alice", Password: "secret",
	}); err != nil {
		t.Fatalf("save credentials: %v", err)
	}
	if _, err := database.Conn.ExecContext(context.Background(), `
INSERT INTO projects (id, calendar_href, display_name, sync_strategy, server_version, created_at, updated_at)
VALUES ('project-1', '/cal/work/', 'Work', 'fullscan', 2, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
`); err != nil {
		t.Fatalf("seed project: %v", err)
	}
	if _, err := database.Conn.ExecContext(context.Background(), `
INSERT INTO tasks (
    id, project_id, uid, href, server_version, title, status, raw_vtodo, project_name, sync_status, created_at, updated_at
) VALUES (
    'task-1', 'project-1', 'uid-1', '/cal/work/uid-1.ics', 1, 'Task', 'needs-action', 'BEGIN:VTODO\nUID:uid-1\nEND:VTODO', 'Work', 'synced', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
);
`); err != nil {
		t.Fatalf("seed task: %v", err)
	}

	calendar := &fakeProjectRenameCalendarClient{renamed: caldav.Calendar{Href: "/cal/work/", DisplayName: "Renamed Work"}}
	h := ProjectRename(projectRenameDependencies{database: database, encryptionKey: []byte("12345678901234567890123456789012"), calendar: calendar})

	form := url.Values{"expected_version": {"2"}, "display_name": {"Renamed Work"}}
	request := httptest.NewRequest(http.MethodPatch, "/projects/project-1", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	responseRecorder := httptest.NewRecorder()

	h(responseRecorder, request.WithContext(withProjectID(request.Context(), "project-1")))

	if responseRecorder.Code != http.StatusOK {
		t.Fatalf("unexpected status code: got %d want %d", responseRecorder.Code, http.StatusOK)
	}
	if calendar.renameCalls != 1 {
		t.Fatalf("unexpected rename calls: got %d want %d", calendar.renameCalls, 1)
	}

	var displayName string
	if err := database.Conn.QueryRowContext(context.Background(), `SELECT display_name FROM projects WHERE id = 'project-1';`).Scan(&displayName); err != nil {
		t.Fatalf("load project: %v", err)
	}
	if displayName != "Renamed Work" {
		t.Fatalf("unexpected project name: got %q want %q", displayName, "Renamed Work")
	}

	var taskProjectName string
	if err := database.Conn.QueryRowContext(context.Background(), `SELECT project_name FROM tasks WHERE id = 'task-1';`).Scan(&taskProjectName); err != nil {
		t.Fatalf("load task: %v", err)
	}
	if taskProjectName != "Renamed Work" {
		t.Fatalf("unexpected task project_name: got %q want %q", taskProjectName, "Renamed Work")
	}
}

func TestProjectRenameDoesNotPersistWhenRemoteRenameFails(t *testing.T) {
	t.Parallel()

	database, err := db.OpenSQLite(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	if err := database.SaveCalDAVCredentials(context.Background(), []byte("12345678901234567890123456789012"), db.CalDAVCredentials{
		URL: "https://example.test/caldav", Username: "alice", Password: "secret",
	}); err != nil {
		t.Fatalf("save credentials: %v", err)
	}
	if _, err := database.Conn.ExecContext(context.Background(), `
INSERT INTO projects (id, calendar_href, display_name, sync_strategy, server_version, created_at, updated_at)
VALUES ('project-1', '/cal/work/', 'Work', 'fullscan', 2, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
`); err != nil {
		t.Fatalf("seed project: %v", err)
	}

	calendar := &fakeProjectRenameCalendarClient{renameErr: errors.New("boom")}
	h := ProjectRename(projectRenameDependencies{database: database, encryptionKey: []byte("12345678901234567890123456789012"), calendar: calendar})

	form := url.Values{"expected_version": {"2"}, "display_name": {"Renamed Work"}}
	request := httptest.NewRequest(http.MethodPatch, "/projects/project-1", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	responseRecorder := httptest.NewRecorder()

	h(responseRecorder, request.WithContext(withProjectID(request.Context(), "project-1")))

	if responseRecorder.Code != http.StatusBadGateway {
		t.Fatalf("unexpected status code: got %d want %d", responseRecorder.Code, http.StatusBadGateway)
	}

	var displayName string
	if err := database.Conn.QueryRowContext(context.Background(), `SELECT display_name FROM projects WHERE id = 'project-1';`).Scan(&displayName); err != nil {
		t.Fatalf("load project: %v", err)
	}
	if displayName != "Work" {
		t.Fatalf("expected unchanged project name, got %q", displayName)
	}
}

func withProjectID(ctx context.Context, projectID string) context.Context {
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("projectID", projectID)
	return context.WithValue(ctx, chi.RouteCtxKey, routeCtx)
}
