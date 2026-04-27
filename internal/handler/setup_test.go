package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"caldo/internal/caldav"
	"caldo/internal/db"
	"caldo/internal/view"
)

type fakeCalendarClient struct {
	calendars []caldav.Calendar
	created   caldav.Calendar
	createErr error
}

func (f fakeCalendarClient) ListCalendars(_ context.Context, _ caldav.Credentials) ([]caldav.Calendar, error) {
	return f.calendars, nil
}

func (f fakeCalendarClient) CreateCalendar(_ context.Context, _ caldav.Credentials, _ string) (caldav.Calendar, error) {
	if f.createErr != nil {
		return caldav.Calendar{}, f.createErr
	}
	return f.created, nil
}

type fakeTodoClient struct {
	objectsByCalendar map[string][]caldav.CalendarObject
	err               error
}

type fakeSetupScheduler struct {
	startErr   error
	startCalls int
}

func (f *fakeSetupScheduler) Start(_ context.Context) error {
	f.startCalls++
	return f.startErr
}

func (f fakeTodoClient) ListVTODOs(_ context.Context, _ caldav.Credentials, calendarHref string) ([]caldav.CalendarObject, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.objectsByCalendar[calendarHref], nil
}
func TestSetupPageRendersCalDAVForm(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest(http.MethodGet, "/setup", nil)
	request = request.WithContext(view.WithAssetManifest(request.Context(), map[string]string{"app.css": "app.css", "app.js": "app.js", "htmx.min.js": "htmx.min.js", "htmx-sse.js": "htmx-sse.js", "alpine.min.js": "alpine.min.js"}))
	responseRecorder := httptest.NewRecorder()

	SetupPage(responseRecorder, request)

	if responseRecorder.Code != http.StatusOK {
		t.Fatalf("unexpected status code: got %d want %d", responseRecorder.Code, http.StatusOK)
	}
	body := responseRecorder.Body.String()
	for _, want := range []string{"name=\"caldav_url\"", "name=\"caldav_username\"", "name=\"caldav_password\"", "action=\"/setup/caldav\""} {
		if !strings.Contains(body, want) {
			t.Fatalf("setup page missing %q", want)
		}
	}
}

func TestSetupCalDAVSuccessStoresCredentialsCapabilitiesAndAdvancesStep(t *testing.T) {
	t.Parallel()

	database, err := db.OpenSQLite(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	caldavServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("DAV", "1, calendar-access, sync-collection")
		w.Header().Set("Content-Type", "application/xml; charset=utf-8")
		w.WriteHeader(http.StatusMultiStatus)
		_, _ = w.Write([]byte(`<d:multistatus xmlns:d="DAV:" xmlns:cs="http://calendarserver.org/ns/"><d:getetag>\"etag\"</d:getetag><cs:getctag>ctag</cs:getctag></d:multistatus>`))
	}))
	t.Cleanup(caldavServer.Close)

	h := SetupCalDAV(setupDependencies{
		database:      database,
		encryptionKey: []byte("12345678901234567890123456789012"),
		tester:        caldav.NewConnectionTester(caldavServer.Client()),
	})

	form := url.Values{}
	form.Set("caldav_url", caldavServer.URL)
	form.Set("caldav_username", "alice")
	form.Set("caldav_password", "secret")
	request := httptest.NewRequest(http.MethodPost, "/setup/caldav", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request = request.WithContext(view.WithAssetManifest(request.Context(), map[string]string{"app.css": "app.css", "app.js": "app.js", "htmx.min.js": "htmx.min.js", "htmx-sse.js": "htmx-sse.js", "alpine.min.js": "alpine.min.js"}))
	responseRecorder := httptest.NewRecorder()

	h(responseRecorder, request)

	if responseRecorder.Code != http.StatusFound {
		t.Fatalf("unexpected status code: got %d want %d", responseRecorder.Code, http.StatusFound)
	}
	if got := responseRecorder.Header().Get("Location"); got != "/setup/calendars" {
		t.Fatalf("unexpected location: got %q", got)
	}

	status, err := database.LoadSetupStatus(context.Background())
	if err != nil {
		t.Fatalf("load setup status: %v", err)
	}
	if status.Step != "calendars" {
		t.Fatalf("unexpected setup step: got %q want %q", status.Step, "calendars")
	}

	credentials, err := database.LoadCalDAVCredentials(context.Background(), []byte("12345678901234567890123456789012"))
	if err != nil {
		t.Fatalf("load credentials: %v", err)
	}
	if credentials.URL != caldavServer.URL || credentials.Username != "alice" || credentials.Password != "secret" {
		t.Fatalf("unexpected credentials: %#v", credentials)
	}

	var rawCapabilities string
	if err := database.Conn.QueryRowContext(context.Background(), `SELECT caldav_server_capabilities FROM settings WHERE id='default';`).Scan(&rawCapabilities); err != nil {
		t.Fatalf("query capabilities: %v", err)
	}
	var capabilities db.CalDAVServerCapabilities
	if err := json.Unmarshal([]byte(rawCapabilities), &capabilities); err != nil {
		t.Fatalf("unmarshal capabilities: %v", err)
	}
	if !capabilities.WebDAVSync || !capabilities.CTag || !capabilities.ETag || !capabilities.FullScan {
		t.Fatalf("unexpected capabilities: %#v", capabilities)
	}
}

func TestSetupCalDAVFailureKeepsStepOnCaldav(t *testing.T) {
	t.Parallel()

	database, err := db.OpenSQLite(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	caldavServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	t.Cleanup(caldavServer.Close)

	h := SetupCalDAV(setupDependencies{
		database:      database,
		encryptionKey: []byte("12345678901234567890123456789012"),
		tester:        caldav.NewConnectionTester(caldavServer.Client()),
	})

	form := url.Values{}
	form.Set("caldav_url", caldavServer.URL)
	form.Set("caldav_username", "alice")
	form.Set("caldav_password", "secret")
	request := httptest.NewRequest(http.MethodPost, "/setup/caldav", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request = request.WithContext(view.WithAssetManifest(request.Context(), map[string]string{"app.css": "app.css", "app.js": "app.js", "htmx.min.js": "htmx.min.js", "htmx-sse.js": "htmx-sse.js", "alpine.min.js": "alpine.min.js"}))
	responseRecorder := httptest.NewRecorder()

	h(responseRecorder, request)

	if responseRecorder.Code != http.StatusOK {
		t.Fatalf("unexpected status code: got %d want %d", responseRecorder.Code, http.StatusOK)
	}
	if !strings.Contains(responseRecorder.Body.String(), "verbindungstest fehlgeschlagen") {
		t.Fatalf("expected connection test error message, got %q", responseRecorder.Body.String())
	}

	status, err := database.LoadSetupStatus(context.Background())
	if err != nil {
		t.Fatalf("load setup status: %v", err)
	}
	if status.Step != "caldav" {
		t.Fatalf("unexpected setup step: got %q want %q", status.Step, "caldav")
	}

	if _, err := database.LoadCalDAVCredentials(context.Background(), []byte("12345678901234567890123456789012")); err != nil {
		t.Fatalf("expected credentials to be stored even on failed connection test: %v", err)
	}
}

func TestSetupCalendarsPageRendersCalendars(t *testing.T) {
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

	h := SetupCalendarsPage(setupDependencies{
		database:      database,
		encryptionKey: []byte("12345678901234567890123456789012"),
		calendar: fakeCalendarClient{
			calendars: []caldav.Calendar{{Href: "/cal/work/", DisplayName: "Work"}},
		},
	})

	request := httptest.NewRequest(http.MethodGet, "/setup/calendars", nil)
	request = request.WithContext(view.WithAssetManifest(request.Context(), map[string]string{"app.css": "app.css", "app.js": "app.js", "htmx.min.js": "htmx.min.js", "htmx-sse.js": "htmx-sse.js", "alpine.min.js": "alpine.min.js"}))
	responseRecorder := httptest.NewRecorder()
	h(responseRecorder, request)

	if responseRecorder.Code != http.StatusOK {
		t.Fatalf("unexpected status: got %d want %d", responseRecorder.Code, http.StatusOK)
	}
	if !strings.Contains(responseRecorder.Body.String(), "Work") {
		t.Fatalf("expected calendar name in response body, got %q", responseRecorder.Body.String())
	}
}

func TestSetupCalendarsPageRedirectsToSetupWhenCredentialsMissing(t *testing.T) {
	t.Parallel()

	database, err := db.OpenSQLite(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	h := SetupCalendarsPage(setupDependencies{
		database:      database,
		encryptionKey: []byte("12345678901234567890123456789012"),
		calendar: fakeCalendarClient{
			calendars: []caldav.Calendar{{Href: "/cal/work/", DisplayName: "Work"}},
		},
	})

	request := httptest.NewRequest(http.MethodGet, "/setup/calendars", nil)
	request = request.WithContext(view.WithAssetManifest(request.Context(), map[string]string{"app.css": "app.css", "app.js": "app.js", "htmx.min.js": "htmx.min.js", "htmx-sse.js": "htmx-sse.js", "alpine.min.js": "alpine.min.js"}))
	responseRecorder := httptest.NewRecorder()
	h(responseRecorder, request)

	if responseRecorder.Code != http.StatusFound {
		t.Fatalf("unexpected status: got %d want %d", responseRecorder.Code, http.StatusFound)
	}
	if got := responseRecorder.Header().Get("Location"); got != "/setup" {
		t.Fatalf("unexpected location: got %q want %q", got, "/setup")
	}
}

func TestSetupCalendarsSuccessStoresProjectsAndAdvancesToImport(t *testing.T) {
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
	if err := database.SaveCalDAVServerCapabilities(context.Background(), db.CalDAVServerCapabilities{WebDAVSync: true, FullScan: true}); err != nil {
		t.Fatalf("save capabilities: %v", err)
	}

	h := SetupCalendars(setupDependencies{
		database:      database,
		encryptionKey: []byte("12345678901234567890123456789012"),
		calendar: fakeCalendarClient{
			calendars: []caldav.Calendar{
				{Href: "/cal/work/", DisplayName: "Work"},
				{Href: "/cal/home/", DisplayName: "Home"},
			},
		},
	})

	form := url.Values{}
	form.Add("calendar_href", "/cal/work/")
	form.Add("calendar_href", "/cal/home/")
	form.Set("default_calendar_href", "/cal/home/")

	request := httptest.NewRequest(http.MethodPost, "/setup/calendars", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request = request.WithContext(view.WithAssetManifest(request.Context(), map[string]string{"app.css": "app.css", "app.js": "app.js", "htmx.min.js": "htmx.min.js", "htmx-sse.js": "htmx-sse.js", "alpine.min.js": "alpine.min.js"}))
	responseRecorder := httptest.NewRecorder()
	h(responseRecorder, request)

	if responseRecorder.Code != http.StatusFound {
		t.Fatalf("unexpected status: got %d want %d", responseRecorder.Code, http.StatusFound)
	}
	if got := responseRecorder.Header().Get("Location"); got != "/setup/import" {
		t.Fatalf("unexpected location: got %q want %q", got, "/setup/import")
	}

	status, err := database.LoadSetupStatus(context.Background())
	if err != nil {
		t.Fatalf("load setup status: %v", err)
	}
	if status.Step != "import" {
		t.Fatalf("unexpected setup step: got %q want %q", status.Step, "import")
	}
}

func TestSetupCalendarsRequiresDefaultProject(t *testing.T) {
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

	h := SetupCalendars(setupDependencies{
		database:      database,
		encryptionKey: []byte("12345678901234567890123456789012"),
		calendar: fakeCalendarClient{
			calendars: []caldav.Calendar{{Href: "/cal/work/", DisplayName: "Work"}},
		},
	})

	form := url.Values{}
	form.Add("calendar_href", "/cal/work/")
	request := httptest.NewRequest(http.MethodPost, "/setup/calendars", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request = request.WithContext(view.WithAssetManifest(request.Context(), map[string]string{"app.css": "app.css", "app.js": "app.js", "htmx.min.js": "htmx.min.js", "htmx-sse.js": "htmx-sse.js", "alpine.min.js": "alpine.min.js"}))
	responseRecorder := httptest.NewRecorder()
	h(responseRecorder, request)

	if responseRecorder.Code != http.StatusOK {
		t.Fatalf("unexpected status: got %d want %d", responseRecorder.Code, http.StatusOK)
	}
	if !strings.Contains(responseRecorder.Body.String(), "default-projekt ist erforderlich") {
		t.Fatalf("expected default project validation message, got %q", responseRecorder.Body.String())
	}
}

func TestSetupCalendarsCanCreateNewDefaultProject(t *testing.T) {
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

	h := SetupCalendars(setupDependencies{
		database:      database,
		encryptionKey: []byte("12345678901234567890123456789012"),
		calendar: fakeCalendarClient{
			calendars: []caldav.Calendar{{Href: "/cal/work/", DisplayName: "Work"}},
			created:   caldav.Calendar{Href: "/cal/inbox/", DisplayName: "Inbox"},
		},
	})

	form := url.Values{}
	form.Add("calendar_href", "/cal/work/")
	form.Set("new_default_project_name", "Inbox")

	request := httptest.NewRequest(http.MethodPost, "/setup/calendars", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request = request.WithContext(view.WithAssetManifest(request.Context(), map[string]string{"app.css": "app.css", "app.js": "app.js", "htmx.min.js": "htmx.min.js", "htmx-sse.js": "htmx-sse.js", "alpine.min.js": "alpine.min.js"}))
	responseRecorder := httptest.NewRecorder()
	h(responseRecorder, request)

	if responseRecorder.Code != http.StatusFound {
		t.Fatalf("unexpected status: got %d want %d", responseRecorder.Code, http.StatusFound)
	}
}

func TestSetupCalendarsCreateNewDefaultProjectFailureShowsError(t *testing.T) {
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

	h := SetupCalendars(setupDependencies{
		database:      database,
		encryptionKey: []byte("12345678901234567890123456789012"),
		calendar: fakeCalendarClient{
			calendars: []caldav.Calendar{
				{Href: "/cal/work/", DisplayName: "Work"},
				{Href: "/cal/home/", DisplayName: "Home"},
			},
			createErr: errors.New("boom"),
		},
	})

	form := url.Values{}
	form.Add("calendar_href", "/cal/work/")
	form.Set("new_default_project_name", "Inbox")
	request := httptest.NewRequest(http.MethodPost, "/setup/calendars", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request = request.WithContext(view.WithAssetManifest(request.Context(), map[string]string{"app.css": "app.css", "app.js": "app.js", "htmx.min.js": "htmx.min.js", "htmx-sse.js": "htmx-sse.js", "alpine.min.js": "alpine.min.js"}))
	responseRecorder := httptest.NewRecorder()
	h(responseRecorder, request)

	if responseRecorder.Code != http.StatusOK {
		t.Fatalf("unexpected status: got %d want %d", responseRecorder.Code, http.StatusOK)
	}
	if !strings.Contains(responseRecorder.Body.String(), "konnte nicht angelegt werden") {
		t.Fatalf("expected create default project error message, got %q", responseRecorder.Body.String())
	}
	if !strings.Contains(responseRecorder.Body.String(), `value="/cal/work/" checked`) {
		t.Fatalf("expected selected work calendar to remain checked, got %q", responseRecorder.Body.String())
	}
	if strings.Contains(responseRecorder.Body.String(), `value="/cal/home/" checked`) {
		t.Fatalf("expected unselected home calendar to remain unchecked, got %q", responseRecorder.Body.String())
	}
}

func TestSetupImportRunsInitialImportAndPersistsSyncedTasks(t *testing.T) {
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
	if err := database.SaveSetupCalendars(context.Background(), []db.SelectedCalendar{{Href: "/cal/work/", DisplayName: "Work"}}, "/cal/work/", "fullscan"); err != nil {
		t.Fatalf("save setup calendars: %v", err)
	}

	broker := newSetupImportEventBroker()
	h := SetupImport(setupDependencies{
		database:      database,
		encryptionKey: []byte("12345678901234567890123456789012"),
		todos: fakeTodoClient{objectsByCalendar: map[string][]caldav.CalendarObject{
			"/cal/work/": {{
				Href:     "/cal/work/uid-1.ics",
				ETag:     "\"etag-1\"",
				RawVTODO: "BEGIN:VCALENDAR\nBEGIN:VTODO\nUID:uid-1\nSUMMARY:Task One\nSTATUS:NEEDS-ACTION\nCATEGORIES:home,STARRED\nEND:VTODO\nEND:VCALENDAR",
			}},
		}},
		importBroker: broker,
	})

	request := httptest.NewRequest(http.MethodPost, "/setup/import", nil)
	responseRecorder := httptest.NewRecorder()
	h(responseRecorder, request)

	if responseRecorder.Code != http.StatusAccepted {
		t.Fatalf("unexpected status: got %d want %d", responseRecorder.Code, http.StatusAccepted)
	}

	waitForImportDone(t, broker)

	var syncStatus, baseVTODO, rawVTODO, projectName, labelNames string
	if err := database.Conn.QueryRow(`
SELECT sync_status, base_vtodo, raw_vtodo, project_name, label_names
FROM tasks
WHERE uid = 'uid-1'
`).Scan(&syncStatus, &baseVTODO, &rawVTODO, &projectName, &labelNames); err != nil {
		t.Fatalf("query imported task: %v", err)
	}
	if syncStatus != "synced" {
		t.Fatalf("unexpected sync_status: got %q", syncStatus)
	}
	if baseVTODO != rawVTODO {
		t.Fatalf("expected base_vtodo to equal raw_vtodo")
	}
	if projectName != "Work" || labelNames == "" {
		t.Fatalf("expected denormalized fields to be populated")
	}
}

func TestSetupImportEventsStreamsProgress(t *testing.T) {
	t.Parallel()

	broker := newSetupImportEventBroker()

	h := SetupImportEvents(setupDependencies{importBroker: broker})
	ctx, cancel := context.WithCancel(context.Background())
	request := httptest.NewRequest(http.MethodGet, "/setup/import/events", nil).WithContext(ctx)
	responseRecorder := httptest.NewRecorder()

	done := make(chan struct{})
	go func() {
		h(responseRecorder, request)
		close(done)
	}()

	time.Sleep(10 * time.Millisecond)
	broker.Publish(setupImportEvent{Event: "progress", Data: `{"phase":"calendar_done"}`})
	broker.Publish(setupImportEvent{Event: "done", Data: `{"phase":"done"}`})
	time.Sleep(10 * time.Millisecond)
	cancel()
	<-done

	if got := responseRecorder.Header().Get("Content-Type"); !strings.Contains(got, "text/event-stream") {
		t.Fatalf("unexpected content type: %q", got)
	}
	body := responseRecorder.Body.String()
	if !strings.Contains(body, "event: progress") {
		t.Fatalf("expected progress event, got %q", body)
	}
}

func waitForImportDone(t *testing.T, broker *setupImportEventBroker) {
	t.Helper()
	subscriber := broker.Subscribe()
	defer broker.Unsubscribe(subscriber.id)
	for i := 0; i < 200; i++ {
		select {
		case event := <-subscriber.ch:
			if event.Event == "done" || event.Event == "error" {
				return
			}
		default:
			time.Sleep(5 * time.Millisecond)
		}
	}
	t.Fatal("timed out waiting for import completion")
}

func TestSetupCompleteMarksSetupCompleteStartsSchedulerAndRedirects(t *testing.T) {
	t.Parallel()

	database, err := db.OpenSQLite(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	if err := database.SaveSetupCalendars(context.Background(), []db.SelectedCalendar{{Href: "/cal/work/", DisplayName: "Work"}}, "/cal/work/", "fullscan"); err != nil {
		t.Fatalf("save setup calendars: %v", err)
	}

	scheduler := &fakeSetupScheduler{}
	setupState := NewSetupState(false)
	h := SetupComplete(setupDependencies{
		database:   database,
		scheduler:  scheduler,
		setupState: setupState,
		logger:     slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil)),
	})

	request := httptest.NewRequest(http.MethodPost, "/setup/complete", nil)
	responseRecorder := httptest.NewRecorder()
	h(responseRecorder, request)

	if responseRecorder.Code != http.StatusFound {
		t.Fatalf("unexpected status: got %d want %d", responseRecorder.Code, http.StatusFound)
	}
	if got := responseRecorder.Header().Get("Location"); got != "/" {
		t.Fatalf("unexpected location: got %q want %q", got, "/")
	}
	if scheduler.startCalls != 1 {
		t.Fatalf("unexpected scheduler start calls: got %d want %d", scheduler.startCalls, 1)
	}
	if !setupState.IsComplete() {
		t.Fatal("expected setup state to be complete")
	}

	status, err := database.LoadSetupStatus(context.Background())
	if err != nil {
		t.Fatalf("load setup status: %v", err)
	}
	if !status.Complete || status.Step != "complete" {
		t.Fatalf("unexpected setup status: %#v", status)
	}
}

func TestSetupCompleteSchedulerStartFailureDoesNotRollbackSetup(t *testing.T) {
	t.Parallel()

	database, err := db.OpenSQLite(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	if err := database.SaveSetupCalendars(context.Background(), []db.SelectedCalendar{{Href: "/cal/work/", DisplayName: "Work"}}, "/cal/work/", "fullscan"); err != nil {
		t.Fatalf("save setup calendars: %v", err)
	}

	scheduler := &fakeSetupScheduler{startErr: errors.New("boom")}
	h := SetupComplete(setupDependencies{
		database:   database,
		scheduler:  scheduler,
		setupState: NewSetupState(false),
		logger:     slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil)),
	})

	request := httptest.NewRequest(http.MethodPost, "/setup/complete", nil)
	responseRecorder := httptest.NewRecorder()
	h(responseRecorder, request)

	if responseRecorder.Code != http.StatusFound {
		t.Fatalf("unexpected status: got %d want %d", responseRecorder.Code, http.StatusFound)
	}
	status, err := database.LoadSetupStatus(context.Background())
	if err != nil {
		t.Fatalf("load setup status: %v", err)
	}
	if !status.Complete || status.Step != "complete" {
		t.Fatalf("expected setup completion persisted despite scheduler error, got %#v", status)
	}
}
