package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"caldo/internal/caldav"
	"caldo/internal/db"
)

type fakeSettingsConnectionTester struct {
	capabilities caldav.ServerCapabilities
	err          error
	credentials  caldav.Credentials
	calls        int
}

func (f *fakeSettingsConnectionTester) TestConnection(_ context.Context, credentials caldav.Credentials) (caldav.ServerCapabilities, error) {
	f.calls++
	f.credentials = credentials
	if f.err != nil {
		return caldav.ServerCapabilities{}, f.err
	}
	return f.capabilities, nil
}

func TestSettingsCalDAVUpdateTestsThenStoresCredentials(t *testing.T) {
	t.Parallel()

	database, err := db.OpenSQLite(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	tester := &fakeSettingsConnectionTester{capabilities: caldav.ServerCapabilities{
		WebDAVSync: true,
		CTag:       true,
		ETag:       true,
		FullScan:   true,
	}}
	handler := SettingsCalDAVUpdate(settingsDependencies{
		database:      database,
		encryptionKey: []byte("12345678901234567890123456789012"),
		tester:        tester,
	})

	form := url.Values{}
	form.Set("caldav_url", "https://nextcloud.example/remote.php/dav")
	form.Set("caldav_username", "alice")
	form.Set("caldav_password", "secret")
	form.Set("caldav_action", "save")
	request := httptest.NewRequest(http.MethodPost, "/settings/caldav", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	responseRecorder := httptest.NewRecorder()

	handler(responseRecorder, request)

	if responseRecorder.Code != http.StatusSeeOther {
		t.Fatalf("unexpected status: got %d want %d", responseRecorder.Code, http.StatusSeeOther)
	}
	if got := responseRecorder.Header().Get("Location"); got != "/settings" {
		t.Fatalf("unexpected redirect: got %q", got)
	}
	if tester.credentials.URL != "https://nextcloud.example/remote.php/dav" || tester.credentials.Username != "alice" || tester.credentials.Password != "secret" {
		t.Fatalf("unexpected tested credentials: %#v", tester.credentials)
	}
	credentials, err := database.LoadCalDAVCredentials(context.Background(), []byte("12345678901234567890123456789012"))
	if err != nil {
		t.Fatalf("load credentials: %v", err)
	}
	if credentials.URL != "https://nextcloud.example/remote.php/dav" || credentials.Username != "alice" || credentials.Password != "secret" {
		t.Fatalf("unexpected stored credentials: %#v", credentials)
	}
	capabilities, err := database.LoadCalDAVServerCapabilities(context.Background())
	if err != nil {
		t.Fatalf("load capabilities: %v", err)
	}
	if !capabilities.WebDAVSync || !capabilities.CTag || !capabilities.ETag || !capabilities.FullScan {
		t.Fatalf("unexpected capabilities: %#v", capabilities)
	}
}

func TestSettingsCalDAVUpdateCanTestConnectionWithoutStoringCredentials(t *testing.T) {
	t.Parallel()

	database, err := db.OpenSQLite(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	tester := &fakeSettingsConnectionTester{capabilities: caldav.ServerCapabilities{FullScan: true}}
	handler := SettingsCalDAVUpdate(settingsDependencies{
		database:      database,
		encryptionKey: []byte("12345678901234567890123456789012"),
		tester:        tester,
	})

	form := url.Values{}
	form.Set("caldav_url", "https://nextcloud.example/remote.php/dav")
	form.Set("caldav_username", "alice")
	form.Set("caldav_password", "secret")
	form.Set("caldav_action", "test")
	request := httptest.NewRequest(http.MethodPost, "/settings/caldav", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	responseRecorder := httptest.NewRecorder()

	handler(responseRecorder, request)

	if responseRecorder.Code != http.StatusOK {
		t.Fatalf("unexpected status: got %d want %d body %s", responseRecorder.Code, http.StatusOK, responseRecorder.Body.String())
	}
	if tester.calls != 1 {
		t.Fatalf("expected one connection test, got %d", tester.calls)
	}
	if tester.credentials.URL != "https://nextcloud.example/remote.php/dav" || tester.credentials.Username != "alice" || tester.credentials.Password != "secret" {
		t.Fatalf("unexpected tested credentials: %#v", tester.credentials)
	}
	if !strings.Contains(responseRecorder.Body.String(), "verbindungstest erfolgreich") {
		t.Fatalf("expected success message in settings page: %s", responseRecorder.Body.String())
	}
	assertSingleIntResult(t, database, `SELECT COUNT(*) FROM settings WHERE id = 'default' AND caldav_password_enc IS NULL;`, 1)
}

func TestSettingsCalDAVUpdateReusesStoredPasswordWhenPasswordFieldIsEmpty(t *testing.T) {
	t.Parallel()

	database, err := db.OpenSQLite(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	key := []byte("12345678901234567890123456789012")
	if err := database.SaveCalDAVCredentials(context.Background(), key, db.CalDAVCredentials{
		URL:      "https://old.example/remote.php/dav",
		Username: "alice",
		Password: "stored-secret",
	}); err != nil {
		t.Fatalf("save credentials: %v", err)
	}

	tester := &fakeSettingsConnectionTester{capabilities: caldav.ServerCapabilities{FullScan: true}}
	handler := SettingsCalDAVUpdate(settingsDependencies{
		database:      database,
		encryptionKey: key,
		tester:        tester,
	})

	form := url.Values{}
	form.Set("caldav_url", "https://nextcloud.example/remote.php/dav")
	form.Set("caldav_username", "alice")
	form.Set("caldav_password", "")
	form.Set("caldav_action", "test")
	request := httptest.NewRequest(http.MethodPost, "/settings/caldav", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	responseRecorder := httptest.NewRecorder()

	handler(responseRecorder, request)

	if responseRecorder.Code != http.StatusOK {
		t.Fatalf("unexpected status: got %d want %d body %s", responseRecorder.Code, http.StatusOK, responseRecorder.Body.String())
	}
	if tester.credentials.Password != "stored-secret" {
		t.Fatalf("expected stored password to be reused for test, got %#v", tester.credentials)
	}
}

func TestSettingsCalendarsUpdateStoresSelectedCalendarsAndDefault(t *testing.T) {
	t.Parallel()

	database, err := db.OpenSQLite(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	key := []byte("12345678901234567890123456789012")
	if err := database.SaveCalDAVCredentials(context.Background(), key, db.CalDAVCredentials{
		URL:      "https://nextcloud.example/remote.php/dav",
		Username: "alice",
		Password: "secret",
	}); err != nil {
		t.Fatalf("save credentials: %v", err)
	}
	if err := database.SaveCalDAVServerCapabilities(context.Background(), db.CalDAVServerCapabilities{CTag: true, FullScan: true}); err != nil {
		t.Fatalf("save capabilities: %v", err)
	}

	handler := SettingsCalendarsUpdate(settingsDependencies{
		database:      database,
		encryptionKey: key,
		calendar: fakeCalendarClient{calendars: []caldav.Calendar{
			{Href: "/cal/work/", DisplayName: "Work"},
			{Href: "/cal/home/", DisplayName: "Home"},
		}},
	})

	form := url.Values{}
	form.Add("calendar_href", "/cal/work/")
	form.Add("calendar_href", "/cal/home/")
	form.Set("default_calendar_href", "/cal/home/")
	request := httptest.NewRequest(http.MethodPost, "/settings/calendars", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	responseRecorder := httptest.NewRecorder()

	handler(responseRecorder, request)

	if responseRecorder.Code != http.StatusSeeOther {
		t.Fatalf("unexpected status: got %d want %d body %s", responseRecorder.Code, http.StatusSeeOther, responseRecorder.Body.String())
	}
	assertSingleIntResult(t, database, `SELECT COUNT(*) FROM projects;`, 2)
	assertSingleTextResult(t, database, `SELECT sync_strategy FROM projects WHERE calendar_href = '/cal/work/';`, "ctag")
	assertSingleIntResult(t, database, `SELECT COUNT(*) FROM projects WHERE is_default = TRUE AND calendar_href = '/cal/home/';`, 1)
}

func TestSettingsCalendarsUpdateRejectsDefaultCalendarThatIsNotSelected(t *testing.T) {
	t.Parallel()

	database, err := db.OpenSQLite(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	key := []byte("12345678901234567890123456789012")
	if err := database.SaveCalDAVCredentials(context.Background(), key, db.CalDAVCredentials{
		URL:      "https://nextcloud.example/remote.php/dav",
		Username: "alice",
		Password: "secret",
	}); err != nil {
		t.Fatalf("save credentials: %v", err)
	}

	handler := SettingsCalendarsUpdate(settingsDependencies{
		database:      database,
		encryptionKey: key,
		calendar: fakeCalendarClient{calendars: []caldav.Calendar{
			{Href: "/cal/work/", DisplayName: "Work"},
			{Href: "/cal/home/", DisplayName: "Home"},
		}},
	})

	form := url.Values{}
	form.Add("calendar_href", "/cal/work/")
	form.Set("default_calendar_href", "/cal/home/")
	request := httptest.NewRequest(http.MethodPost, "/settings/calendars", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	responseRecorder := httptest.NewRecorder()

	handler(responseRecorder, request)

	if responseRecorder.Code != http.StatusOK {
		t.Fatalf("unexpected status: got %d want %d body %s", responseRecorder.Code, http.StatusOK, responseRecorder.Body.String())
	}
	if !strings.Contains(responseRecorder.Body.String(), "default-projekt muss ein ausgewählter kalender sein") {
		t.Fatalf("expected default selection error, got body %s", responseRecorder.Body.String())
	}
	assertSingleIntResult(t, database, `SELECT COUNT(*) FROM projects;`, 0)
}

func TestIsSupportedUILanguage(t *testing.T) {
	if !isSupportedUILanguage("de") || !isSupportedUILanguage("en") {
		t.Fatal("expected de and en to be supported")
	}
	if isSupportedUILanguage("fr") {
		t.Fatal("fr must not be supported")
	}
}

func TestIsSupportedDarkMode(t *testing.T) {
	for _, mode := range []string{"light", "dark", "system"} {
		if !isSupportedDarkMode(mode) {
			t.Fatalf("expected supported mode %q", mode)
		}
	}
	if isSupportedDarkMode("amoled") {
		t.Fatal("unexpected custom mode support")
	}
}
