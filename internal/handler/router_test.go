package handler

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"caldo/internal/assets"
	"caldo/internal/db"
	"caldo/internal/logging"
	"caldo/internal/view"
)

func testManifest(t *testing.T) assets.Manifest {
	t.Helper()

	manifest, err := assets.LoadManifest(filepath.Join("..", "..", "web", "static", "manifest.json"))
	if err != nil {
		t.Fatalf("load test manifest: %v", err)
	}

	return manifest
}

func staticAssetPath(manifest assets.Manifest, logicalName string) string {
	return "/static/" + manifest[logicalName]
}

func TestNewRouterExposesHealthWithoutAuth(t *testing.T) {
	t.Parallel()

	logger := logging.New(bytes.NewBuffer(nil), "production", "info")
	responseRecorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/health", nil)

	NewRouter(logger, "X-Forwarded-User", testManifest(t), true, []byte("12345678901234567890123456789012"), nil, context.Background(), nil).ServeHTTP(responseRecorder, request)

	if responseRecorder.Code != http.StatusOK {
		t.Fatalf("unexpected status code: got %d want %d", responseRecorder.Code, http.StatusOK)
	}
	if responseRecorder.Header().Get("X-Request-ID") == "" {
		t.Fatal("expected request id header")
	}
	if got := responseRecorder.Header().Get("X-Frame-Options"); got != "DENY" {
		t.Fatalf("unexpected X-Frame-Options: got %q", got)
	}
	if got := responseRecorder.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("unexpected X-Content-Type-Options: got %q", got)
	}
}

func TestNewRouterRejectsNonHealthRequestWithoutProxyAuthHeader(t *testing.T) {
	t.Parallel()

	logger := logging.New(bytes.NewBuffer(nil), "production", "info")
	responseRecorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/settings", nil)

	NewRouter(logger, "X-Forwarded-User", testManifest(t), true, []byte("12345678901234567890123456789012"), nil, context.Background(), nil).ServeHTTP(responseRecorder, request)

	if responseRecorder.Code != http.StatusForbidden {
		t.Fatalf("unexpected status code: got %d want %d", responseRecorder.Code, http.StatusForbidden)
	}
}

func TestNewRouterServesStaticAssetsWithLongTermCacheHeaders(t *testing.T) {
	t.Parallel()

	logger := logging.New(bytes.NewBuffer(nil), "production", "info")
	responseRecorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/static/manifest.json", nil)
	request.Header.Set("X-Forwarded-User", "alice")

	NewRouter(logger, "X-Forwarded-User", testManifest(t), true, []byte("12345678901234567890123456789012"), nil, context.Background(), nil).ServeHTTP(responseRecorder, request)

	if responseRecorder.Code != http.StatusOK {
		t.Fatalf("unexpected status code: got %d want %d", responseRecorder.Code, http.StatusOK)
	}
	if got := responseRecorder.Header().Get("Cache-Control"); got != "public, max-age=31536000, immutable" {
		t.Fatalf("unexpected Cache-Control header: got %q", got)
	}
}

func TestNewRouterRendersBaseLayoutOnRoot(t *testing.T) {
	t.Parallel()

	manifest := testManifest(t)
	logger := logging.New(bytes.NewBuffer(nil), "production", "info")
	responseRecorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("X-Forwarded-User", "alice")

	NewRouter(logger, "X-Forwarded-User", manifest, true, []byte("12345678901234567890123456789012"), nil, context.Background(), nil).ServeHTTP(responseRecorder, request)

	if responseRecorder.Code != http.StatusOK {
		t.Fatalf("unexpected status code: got %d want %d", responseRecorder.Code, http.StatusOK)
	}

	body := responseRecorder.Body.String()
	csrfToken := responseRecorder.Header().Get(csrfHeaderName)
	if csrfToken == "" {
		t.Fatal("expected csrf token response header")
	}
	sessionCookie := findResponseCookie(responseRecorder, sessionCookieName)
	if sessionCookie == nil {
		t.Fatal("expected session_id cookie")
	}
	if !validSessionID(sessionCookie.Value) {
		t.Fatalf("expected valid session_id cookie, got %q", sessionCookie.Value)
	}
	if !sessionCookie.HttpOnly || !sessionCookie.Secure || sessionCookie.SameSite != http.SameSiteStrictMode {
		t.Fatalf("unexpected session cookie attributes: %+v", sessionCookie)
	}
	if !strings.Contains(body, `<meta name="csrf-token" content="`+csrfToken+`">`) {
		t.Fatal("expected csrf token in base layout meta tag")
	}
	for _, want := range []string{
		"<!doctype html>",
		`id="notifications"`,
		`<button type="button" data-theme-toggle`,
		`data-theme-mode="system"`,
		`data-theme-system-label="System"`,
		staticAssetPath(manifest, "htmx.min.js"),
		staticAssetPath(manifest, "htmx-sse.js"),
		staticAssetPath(manifest, "alpine.min.js"),
		staticAssetPath(manifest, "app.js"),
		staticAssetPath(manifest, "app.css"),
		`href="/today"`,
		`href="/upcoming"`,
		`href="/projects"`,
		`href="/labels"`,
		`href="/filters"`,
		`href="/favorites"`,
		`href="/overdue"`,
		`href="/no-date"`,
		`href="/completed"`,
		`href="/search"`,
		`href="/conflicts"`,
		`href="/settings"`,
		`data-shortcut-help-dialog`,
		`data-shortcut-help-close`,
		`Neue Aufgabe`,
		`Hilfe öffnen`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("response body missing %q", want)
		}
	}

	for _, notWant := range []string{`x-data=`, `x-init=`, `:class=`, `@click=`} {
		if strings.Contains(body, notWant) {
			t.Fatalf("response body unexpectedly contains %q", notWant)
		}
	}
}

func findResponseCookie(rr *httptest.ResponseRecorder, name string) *http.Cookie {
	for _, cookie := range rr.Result().Cookies() {
		if cookie.Name == name {
			return cookie
		}
	}
	return nil
}

func TestNewRouterAppliesPersistedUIPreferences(t *testing.T) {
	t.Parallel()

	database, err := db.OpenSQLite(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	if _, err := database.Conn.Exec(`UPDATE settings SET ui_language = 'en', dark_mode = 'dark' WHERE id = 'default';`); err != nil {
		t.Fatalf("set ui preferences: %v", err)
	}

	logger := logging.New(bytes.NewBuffer(nil), "production", "info")
	responseRecorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/quick-add", nil)
	request.Header.Set("X-Forwarded-User", "alice")

	NewRouter(logger, "X-Forwarded-User", testManifest(t), true, []byte("12345678901234567890123456789012"), database, context.Background(), nil).ServeHTTP(responseRecorder, request)

	if responseRecorder.Code != http.StatusOK {
		t.Fatalf("unexpected status code: got %d want %d", responseRecorder.Code, http.StatusOK)
	}

	body := responseRecorder.Body.String()
	for _, want := range []string{
		`<html lang="en" data-theme-root data-theme-mode="dark" class="dark">`,
		`CalDAV tasks`,
		`System filters`,
		`New task`,
		`Task`,
		`Appearance: Dark`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("response body missing %q in %s", want, body)
		}
	}
}

func TestNavigationPagesRenderPersistedEntries(t *testing.T) {
	t.Parallel()

	database, err := db.OpenSQLite(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	if _, err := database.Conn.Exec(`
INSERT INTO projects (id, calendar_href, display_name, sync_strategy, is_default, created_at, updated_at)
VALUES ('project-1', '/cal/inbox/', 'Inbox', 'fullscan', TRUE, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
INSERT INTO labels (id, name, created_at) VALUES ('label-1', 'Büro', CURRENT_TIMESTAMP);
INSERT INTO tasks (id, project_id, uid, href, etag, title, status, raw_vtodo, sync_status, created_at, updated_at)
VALUES ('task-1', 'project-1', 'uid-1', '/cal/inbox/task-1.ics', '"etag-1"', 'Task 1', 'needs-action', 'BEGIN:VTODO\nUID:uid-1\nEND:VTODO', 'synced', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
INSERT INTO task_labels (task_id, label_id) VALUES ('task-1', 'label-1');
INSERT INTO saved_filters (id, name, query) VALUES ('filter-1', 'Heute Fokus', 'today');
`); err != nil {
		t.Fatalf("seed navigation pages: %v", err)
	}

	router := NewRouter(logging.New(bytes.NewBuffer(nil), "production", "info"), "X-Forwarded-User", testManifest(t), true, []byte("12345678901234567890123456789012"), database, context.Background(), nil)
	for _, tc := range []struct {
		path       string
		want       string
		wantMarker string
	}{
		{path: "/projects", want: "Inbox", wantMarker: `data-navigation-overview`},
		{path: "/labels", want: "Büro", wantMarker: `data-navigation-overview`},
		{path: "/filters", want: "Heute Fokus", wantMarker: `data-saved-filter-list`},
	} {
		responseRecorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, tc.path, nil)
		request.Header.Set("X-Forwarded-User", "alice")
		router.ServeHTTP(responseRecorder, request)
		if responseRecorder.Code != http.StatusOK {
			t.Fatalf("%s status: got %d want %d", tc.path, responseRecorder.Code, http.StatusOK)
		}
		body := responseRecorder.Body.String()
		if !strings.Contains(body, tc.want) || !strings.Contains(body, tc.wantMarker) || !strings.Contains(body, `caldo-nav-count`) {
			t.Fatalf("%s body missing persisted navigation content: %s", tc.path, body)
		}
	}
}

func TestNewRouterRedirectsNormalRouteToSetupWhenIncomplete(t *testing.T) {
	t.Parallel()

	logger := logging.New(bytes.NewBuffer(nil), "production", "info")
	responseRecorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("X-Forwarded-User", "alice")

	NewRouter(logger, "X-Forwarded-User", testManifest(t), false, []byte("12345678901234567890123456789012"), nil, context.Background(), nil).ServeHTTP(responseRecorder, request)

	if responseRecorder.Code != http.StatusFound {
		t.Fatalf("unexpected status code: got %d want %d", responseRecorder.Code, http.StatusFound)
	}
	if got := responseRecorder.Header().Get("Location"); got != "/setup" {
		t.Fatalf("unexpected redirect location: got %q want %q", got, "/setup")
	}
}

func TestNewRouterAllowsSetupRouteWhenIncomplete(t *testing.T) {
	t.Parallel()

	logger := logging.New(bytes.NewBuffer(nil), "production", "info")
	responseRecorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/setup/", nil)
	request.Header.Set("X-Forwarded-User", "alice")

	NewRouter(logger, "X-Forwarded-User", testManifest(t), false, []byte("12345678901234567890123456789012"), nil, context.Background(), nil).ServeHTTP(responseRecorder, request)

	if responseRecorder.Code != http.StatusOK {
		t.Fatalf("unexpected status code: got %d want %d", responseRecorder.Code, http.StatusOK)
	}
	if got := responseRecorder.Header().Get(csrfHeaderName); got == "" {
		t.Fatal("expected csrf token response header on setup route")
	}
}

func TestNewRouterSetupMutatingRouteRequiresCSRFToken(t *testing.T) {
	t.Parallel()

	logger := logging.New(bytes.NewBuffer(nil), "production", "info")
	responseRecorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/setup/caldav", nil)
	request.Header.Set("X-Forwarded-User", "alice")

	NewRouter(logger, "X-Forwarded-User", testManifest(t), false, []byte("12345678901234567890123456789012"), nil, context.Background(), nil).ServeHTTP(responseRecorder, request)

	if responseRecorder.Code != http.StatusForbidden {
		t.Fatalf("unexpected status code: got %d want %d", responseRecorder.Code, http.StatusForbidden)
	}
}

type routerManualSyncRunner struct {
	called chan struct{}
}

func (r routerManualSyncRunner) Run(context.Context) error {
	close(r.called)
	return nil
}

func TestNewRouterManualSyncUsesProvidedRunner(t *testing.T) {
	database, err := db.OpenSQLite(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	secret := []byte("12345678901234567890123456789012")
	token, err := generateSignedCSRF(secret)
	if err != nil {
		t.Fatalf("generate csrf token: %v", err)
	}
	runner := routerManualSyncRunner{called: make(chan struct{})}
	logger := logging.New(bytes.NewBuffer(nil), "production", "info")
	responseRecorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/sync/manual", nil)
	request.Header.Set("X-Forwarded-User", "alice")
	request.Header.Set(csrfHeaderName, token)
	request.AddCookie(&http.Cookie{Name: csrfCookieName, Value: token})

	NewRouter(logger, "X-Forwarded-User", testManifest(t), true, secret, database, context.Background(), nil, runner).ServeHTTP(responseRecorder, request)

	if responseRecorder.Code != http.StatusOK {
		t.Fatalf("unexpected status code: got %d want %d body=%q", responseRecorder.Code, http.StatusOK, responseRecorder.Body.String())
	}
	select {
	case <-runner.called:
	case <-time.After(time.Second):
		t.Fatal("expected runner to be called")
	}
	waitForSyncStatus(t, database, func(status db.SyncStatus) bool {
		return status.State == "idle" && status.LastSuccessAt.Valid
	})
}

func TestAssetManifestMiddlewarePreservesExistingCSRFToken(t *testing.T) {
	t.Parallel()

	manifest := testManifest(t)
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request = request.WithContext(view.WithCSRFToken(request.Context(), "token-123"))
	responseRecorder := httptest.NewRecorder()

	handler := AssetManifestMiddleware(manifest)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := view.CSRFToken(r.Context()); got != "token-123" {
			t.Fatalf("unexpected csrf token: got %q want %q", got, "token-123")
		}
		if got := view.AssetPath(r.Context(), "app.css"); got != staticAssetPath(manifest, "app.css") {
			t.Fatalf("unexpected asset path: got %q", got)
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	handler.ServeHTTP(responseRecorder, request)

	if responseRecorder.Code != http.StatusNoContent {
		t.Fatalf("unexpected status code: got %d want %d", responseRecorder.Code, http.StatusNoContent)
	}
}

func TestNewRouterProjectMutatingRouteRequiresCSRFToken(t *testing.T) {
	t.Parallel()

	logger := logging.New(bytes.NewBuffer(nil), "production", "info")
	responseRecorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/projects", nil)
	request.Header.Set("X-Forwarded-User", "alice")

	NewRouter(logger, "X-Forwarded-User", testManifest(t), true, []byte("12345678901234567890123456789012"), nil, context.Background(), nil).ServeHTTP(responseRecorder, request)

	if responseRecorder.Code != http.StatusForbidden {
		t.Fatalf("unexpected status code: got %d want %d", responseRecorder.Code, http.StatusForbidden)
	}
}

func TestNewRouterProjectRenameRouteRequiresCSRFToken(t *testing.T) {
	t.Parallel()

	logger := logging.New(bytes.NewBuffer(nil), "production", "info")
	responseRecorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPatch, "/projects/project-1", nil)
	request.Header.Set("X-Forwarded-User", "alice")

	NewRouter(logger, "X-Forwarded-User", testManifest(t), true, []byte("12345678901234567890123456789012"), nil, context.Background(), nil).ServeHTTP(responseRecorder, request)

	if responseRecorder.Code != http.StatusForbidden {
		t.Fatalf("unexpected status code: got %d want %d", responseRecorder.Code, http.StatusForbidden)
	}
}

func TestNewRouterProjectDeleteRouteRequiresCSRFToken(t *testing.T) {
	t.Parallel()

	logger := logging.New(bytes.NewBuffer(nil), "production", "info")
	responseRecorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodDelete, "/projects/project-1", nil)
	request.Header.Set("X-Forwarded-User", "alice")

	NewRouter(logger, "X-Forwarded-User", testManifest(t), true, []byte("12345678901234567890123456789012"), nil, context.Background(), nil).ServeHTTP(responseRecorder, request)

	if responseRecorder.Code != http.StatusForbidden {
		t.Fatalf("unexpected status code: got %d want %d", responseRecorder.Code, http.StatusForbidden)
	}
}

func TestNewRouterFilterMutatingRoutesRequireCSRFToken(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		method string
		path   string
	}{
		{method: http.MethodPost, path: "/filters"},
		{method: http.MethodPatch, path: "/filters/filter-1"},
		{method: http.MethodDelete, path: "/filters/filter-1"},
	} {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			t.Parallel()

			logger := logging.New(bytes.NewBuffer(nil), "production", "info")
			responseRecorder := httptest.NewRecorder()
			request := httptest.NewRequest(tc.method, tc.path, nil)
			request.Header.Set("X-Forwarded-User", "alice")

			NewRouter(logger, "X-Forwarded-User", testManifest(t), true, []byte("12345678901234567890123456789012"), nil, context.Background(), nil).ServeHTTP(responseRecorder, request)

			if responseRecorder.Code != http.StatusForbidden {
				t.Fatalf("unexpected status code: got %d want %d", responseRecorder.Code, http.StatusForbidden)
			}
		})
	}
}

func TestNewRouterTaskLabelsRouteRequiresCSRFToken(t *testing.T) {
	t.Parallel()

	logger := logging.New(bytes.NewBuffer(nil), "production", "info")
	responseRecorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/tasks/task-1/labels", nil)
	request.Header.Set("X-Forwarded-User", "alice")

	NewRouter(logger, "X-Forwarded-User", testManifest(t), true, []byte("12345678901234567890123456789012"), nil, context.Background(), nil).ServeHTTP(responseRecorder, request)

	if responseRecorder.Code != http.StatusForbidden {
		t.Fatalf("unexpected status code: got %d want %d", responseRecorder.Code, http.StatusForbidden)
	}
}
