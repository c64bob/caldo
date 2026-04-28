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
)

type fakeProjectDeleteCalendarClient struct {
	deleteErr   error
	deleteCalls int
}

func (f *fakeProjectDeleteCalendarClient) DeleteCalendar(_ context.Context, _ caldav.Credentials, _ string) error {
	f.deleteCalls++
	return f.deleteErr
}

func TestProjectDeletePersistsAfterRemoteSuccess(t *testing.T) {
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
INSERT INTO tasks (
    id, project_id, uid, href, server_version, title, status, raw_vtodo, project_name, sync_status, created_at, updated_at
) VALUES (
    'task-1', 'project-1', 'uid-1', '/cal/work/uid-1.ics', 1, 'Task', 'needs-action', 'BEGIN:VTODO\nUID:uid-1\nEND:VTODO', 'Work', 'synced', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
);
UPDATE settings SET default_project_id = 'project-1' WHERE id = 'default';
`); err != nil {
		t.Fatalf("seed data: %v", err)
	}

	calendar := &fakeProjectDeleteCalendarClient{}
	h := ProjectDelete(projectDeleteDependencies{database: database, encryptionKey: []byte("12345678901234567890123456789012"), calendar: calendar})

	form := url.Values{"expected_version": {"2"}, "confirmation_name": {"Work"}}
	request := httptest.NewRequest(http.MethodDelete, "/projects/project-1", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	responseRecorder := httptest.NewRecorder()

	h(responseRecorder, request.WithContext(withProjectID(request.Context(), "project-1")))

	if responseRecorder.Code != http.StatusOK {
		t.Fatalf("unexpected status code: got %d want %d", responseRecorder.Code, http.StatusOK)
	}
	if calendar.deleteCalls != 1 {
		t.Fatalf("unexpected delete calls: got %d want %d", calendar.deleteCalls, 1)
	}

	var projectCount int
	if err := database.Conn.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM projects WHERE id = 'project-1';`).Scan(&projectCount); err != nil {
		t.Fatalf("count project: %v", err)
	}
	if projectCount != 0 {
		t.Fatalf("expected project to be deleted, got %d rows", projectCount)
	}

	var taskCount int
	if err := database.Conn.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM tasks WHERE project_id = 'project-1';`).Scan(&taskCount); err != nil {
		t.Fatalf("count tasks: %v", err)
	}
	if taskCount != 0 {
		t.Fatalf("expected tasks to be deleted, got %d rows", taskCount)
	}
}

func TestProjectDeleteDoesNotPersistWhenRemoteDeleteFails(t *testing.T) {
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

	calendar := &fakeProjectDeleteCalendarClient{deleteErr: errors.New("boom")}
	h := ProjectDelete(projectDeleteDependencies{database: database, encryptionKey: []byte("12345678901234567890123456789012"), calendar: calendar})

	form := url.Values{"expected_version": {"2"}, "confirmation_name": {"Work"}}
	request := httptest.NewRequest(http.MethodDelete, "/projects/project-1", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	responseRecorder := httptest.NewRecorder()

	h(responseRecorder, request.WithContext(withProjectID(request.Context(), "project-1")))

	if responseRecorder.Code != http.StatusBadGateway {
		t.Fatalf("unexpected status code: got %d want %d", responseRecorder.Code, http.StatusBadGateway)
	}

	var projectCount int
	if err := database.Conn.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM projects WHERE id = 'project-1';`).Scan(&projectCount); err != nil {
		t.Fatalf("count project: %v", err)
	}
	if projectCount != 1 {
		t.Fatalf("project should remain after remote failure, got %d", projectCount)
	}
}

func TestProjectDeleteCancelsReservationWhenCredentialsUnavailable(t *testing.T) {
	t.Parallel()

	database, err := db.OpenSQLite(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	if _, err := database.Conn.ExecContext(context.Background(), `
INSERT INTO projects (id, calendar_href, display_name, sync_strategy, server_version, created_at, updated_at)
VALUES ('project-1', '/cal/work/', 'Work', 'fullscan', 2, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
`); err != nil {
		t.Fatalf("seed project: %v", err)
	}

	calendar := &fakeProjectDeleteCalendarClient{}
	h := ProjectDelete(projectDeleteDependencies{database: database, encryptionKey: []byte("12345678901234567890123456789012"), calendar: calendar})

	form := url.Values{"expected_version": {"2"}, "confirmation_name": {"Work"}}
	request := httptest.NewRequest(http.MethodDelete, "/projects/project-1", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	responseRecorder := httptest.NewRecorder()

	h(responseRecorder, request.WithContext(withProjectID(request.Context(), "project-1")))

	if responseRecorder.Code != http.StatusFailedDependency {
		t.Fatalf("unexpected status code: got %d want %d", responseRecorder.Code, http.StatusFailedDependency)
	}
	if calendar.deleteCalls != 0 {
		t.Fatalf("remote delete should not be called when credentials are unavailable, got %d calls", calendar.deleteCalls)
	}

	var version int
	if err := database.Conn.QueryRowContext(context.Background(), `SELECT server_version FROM projects WHERE id = 'project-1';`).Scan(&version); err != nil {
		t.Fatalf("load project version: %v", err)
	}
	if version != 2 {
		t.Fatalf("expected reservation rollback to restore version 2, got %d", version)
	}
}
