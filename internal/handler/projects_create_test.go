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

type fakeProjectCreateCalendarClient struct {
	created     caldav.Calendar
	createErr   error
	createCalls int
}

func (f *fakeProjectCreateCalendarClient) CreateCalendar(_ context.Context, _ caldav.Credentials, _ string) (caldav.Calendar, error) {
	f.createCalls++
	if f.createErr != nil {
		return caldav.Calendar{}, f.createErr
	}
	return f.created, nil
}

func TestProjectCreatePersistsAfterRemoteSuccess(t *testing.T) {
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
	if err := database.SaveCalDAVServerCapabilities(context.Background(), db.CalDAVServerCapabilities{CTag: true, FullScan: true}); err != nil {
		t.Fatalf("save capabilities: %v", err)
	}

	calendar := &fakeProjectCreateCalendarClient{created: caldav.Calendar{Href: "/calendars/new-project/", DisplayName: "New Project"}}
	h := ProjectCreate(projectCreateDependencies{database: database, encryptionKey: []byte("12345678901234567890123456789012"), calendar: calendar})

	form := url.Values{"display_name": {"New Project"}}
	request := httptest.NewRequest(http.MethodPost, "/projects", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	responseRecorder := httptest.NewRecorder()

	h(responseRecorder, request)

	if responseRecorder.Code != http.StatusCreated {
		t.Fatalf("unexpected status code: got %d want %d", responseRecorder.Code, http.StatusCreated)
	}
	if calendar.createCalls != 1 {
		t.Fatalf("unexpected create calls: got %d want %d", calendar.createCalls, 1)
	}

	var count int
	if err := database.Conn.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM projects WHERE calendar_href = '/calendars/new-project/' AND display_name = 'New Project' AND sync_strategy = 'ctag';`).Scan(&count); err != nil {
		t.Fatalf("count projects: %v", err)
	}
	if count != 1 {
		t.Fatalf("unexpected stored projects count: got %d want %d", count, 1)
	}
}

func TestProjectCreateDoesNotPersistWhenRemoteCreateFails(t *testing.T) {
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

	calendar := &fakeProjectCreateCalendarClient{createErr: errors.New("boom")}
	h := ProjectCreate(projectCreateDependencies{database: database, encryptionKey: []byte("12345678901234567890123456789012"), calendar: calendar})

	form := url.Values{"display_name": {"New Project"}}
	request := httptest.NewRequest(http.MethodPost, "/projects", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	responseRecorder := httptest.NewRecorder()

	h(responseRecorder, request)

	if responseRecorder.Code != http.StatusBadGateway {
		t.Fatalf("unexpected status code: got %d want %d", responseRecorder.Code, http.StatusBadGateway)
	}

	var count int
	if err := database.Conn.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM projects;`).Scan(&count); err != nil {
		t.Fatalf("count projects: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected no local project persisted, got %d", count)
	}
}

func TestProjectCreateReturnsErrorBeforeRemoteCreateWhenCapabilitiesUnreadable(t *testing.T) {
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
	if _, err := database.Conn.ExecContext(context.Background(), `UPDATE settings SET caldav_server_capabilities = '{not-json}' WHERE id = 'default';`); err != nil {
		t.Fatalf("corrupt capabilities: %v", err)
	}

	calendar := &fakeProjectCreateCalendarClient{}
	h := ProjectCreate(projectCreateDependencies{database: database, encryptionKey: []byte("12345678901234567890123456789012"), calendar: calendar})

	form := url.Values{"display_name": {"New Project"}}
	request := httptest.NewRequest(http.MethodPost, "/projects", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	responseRecorder := httptest.NewRecorder()

	h(responseRecorder, request)

	if responseRecorder.Code != http.StatusInternalServerError {
		t.Fatalf("unexpected status code: got %d want %d", responseRecorder.Code, http.StatusInternalServerError)
	}
	if calendar.createCalls != 0 {
		t.Fatalf("expected no remote calendar creation attempt, got %d", calendar.createCalls)
	}

	var count int
	if err := database.Conn.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM projects;`).Scan(&count); err != nil {
		t.Fatalf("count projects: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected no local project persisted, got %d", count)
	}
}

func TestProjectCreateRejectsEmptyName(t *testing.T) {
	t.Parallel()

	h := ProjectCreate(projectCreateDependencies{})
	form := url.Values{"display_name": {"   "}}
	request := httptest.NewRequest(http.MethodPost, "/projects", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	responseRecorder := httptest.NewRecorder()

	h(responseRecorder, request)

	if responseRecorder.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status code: got %d want %d", responseRecorder.Code, http.StatusBadRequest)
	}
	if !strings.Contains(responseRecorder.Body.String(), "display_name is required") {
		t.Fatalf("expected validation error body, got %q", responseRecorder.Body.String())
	}
}
