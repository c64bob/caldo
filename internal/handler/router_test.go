package handler

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"caldo/internal/assets"
	"caldo/internal/logging"
	"caldo/internal/view"
)

func testManifest() assets.Manifest {
	return assets.Manifest{
		"app.css":       "app.8f3a1c2.css",
		"app.js":        "app.42ab19f.js",
		"htmx.min.js":   "htmx.5e741aa.min.js",
		"htmx-sse.js":   "htmx-sse.9d2f6c1.js",
		"alpine.min.js": "alpine.7cc80d0.min.js",
	}
}

func TestNewRouterExposesHealthWithoutAuth(t *testing.T) {
	t.Parallel()

	logger := logging.New(bytes.NewBuffer(nil), "production", "info")
	responseRecorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/health", nil)

	NewRouter(logger, "X-Forwarded-User", testManifest(), true, []byte("12345678901234567890123456789012"), nil, context.Background(), nil).ServeHTTP(responseRecorder, request)

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

	NewRouter(logger, "X-Forwarded-User", testManifest(), true, []byte("12345678901234567890123456789012"), nil, context.Background(), nil).ServeHTTP(responseRecorder, request)

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

	NewRouter(logger, "X-Forwarded-User", testManifest(), true, []byte("12345678901234567890123456789012"), nil, context.Background(), nil).ServeHTTP(responseRecorder, request)

	if responseRecorder.Code != http.StatusOK {
		t.Fatalf("unexpected status code: got %d want %d", responseRecorder.Code, http.StatusOK)
	}
	if got := responseRecorder.Header().Get("Cache-Control"); got != "public, max-age=31536000, immutable" {
		t.Fatalf("unexpected Cache-Control header: got %q", got)
	}
}

func TestNewRouterRendersBaseLayoutOnRoot(t *testing.T) {
	t.Parallel()

	logger := logging.New(bytes.NewBuffer(nil), "production", "info")
	responseRecorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("X-Forwarded-User", "alice")

	NewRouter(logger, "X-Forwarded-User", testManifest(), true, []byte("12345678901234567890123456789012"), nil, context.Background(), nil).ServeHTTP(responseRecorder, request)

	if responseRecorder.Code != http.StatusOK {
		t.Fatalf("unexpected status code: got %d want %d", responseRecorder.Code, http.StatusOK)
	}

	body := responseRecorder.Body.String()
	for _, want := range []string{
		"<!doctype html>",
		`<meta name="csrf-token" content="">`,
		`id="notifications"`,
		`data-theme-toggle`,
		`/static/htmx.5e741aa.min.js`,
		`/static/htmx-sse.9d2f6c1.js`,
		`/static/alpine.7cc80d0.min.js`,
		`/static/app.42ab19f.js`,
		`/static/app.8f3a1c2.css`,
		`href="/today"`,
		`href="/upcoming"`,
		`href="/projects"`,
		`href="/labels"`,
		`href="/filters"`,
		`href="/favorites"`,
		`href="/search"`,
		`href="/conflicts"`,
		`href="/settings"`,
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

func TestNewRouterRedirectsNormalRouteToSetupWhenIncomplete(t *testing.T) {
	t.Parallel()

	logger := logging.New(bytes.NewBuffer(nil), "production", "info")
	responseRecorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("X-Forwarded-User", "alice")

	NewRouter(logger, "X-Forwarded-User", testManifest(), false, []byte("12345678901234567890123456789012"), nil, context.Background(), nil).ServeHTTP(responseRecorder, request)

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

	NewRouter(logger, "X-Forwarded-User", testManifest(), false, []byte("12345678901234567890123456789012"), nil, context.Background(), nil).ServeHTTP(responseRecorder, request)

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

	NewRouter(logger, "X-Forwarded-User", testManifest(), false, []byte("12345678901234567890123456789012"), nil, context.Background(), nil).ServeHTTP(responseRecorder, request)

	if responseRecorder.Code != http.StatusForbidden {
		t.Fatalf("unexpected status code: got %d want %d", responseRecorder.Code, http.StatusForbidden)
	}
}

func TestAssetManifestMiddlewarePreservesExistingCSRFToken(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request = request.WithContext(view.WithCSRFToken(request.Context(), "token-123"))
	responseRecorder := httptest.NewRecorder()

	handler := AssetManifestMiddleware(testManifest())(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := view.CSRFToken(r.Context()); got != "token-123" {
			t.Fatalf("unexpected csrf token: got %q want %q", got, "token-123")
		}
		if got := view.AssetPath(r.Context(), "app.css"); got != "/static/app.8f3a1c2.css" {
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

	NewRouter(logger, "X-Forwarded-User", testManifest(), true, []byte("12345678901234567890123456789012"), nil, context.Background(), nil).ServeHTTP(responseRecorder, request)

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

	NewRouter(logger, "X-Forwarded-User", testManifest(), true, []byte("12345678901234567890123456789012"), nil, context.Background(), nil).ServeHTTP(responseRecorder, request)

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

	NewRouter(logger, "X-Forwarded-User", testManifest(), true, []byte("12345678901234567890123456789012"), nil, context.Background(), nil).ServeHTTP(responseRecorder, request)

	if responseRecorder.Code != http.StatusForbidden {
		t.Fatalf("unexpected status code: got %d want %d", responseRecorder.Code, http.StatusForbidden)
	}
}
