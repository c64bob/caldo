package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"caldo/internal/assets"
	"caldo/internal/logging"
	"caldo/internal/view"
	"github.com/google/uuid"
)

func TestRequestIDMiddlewareSetsHeaderAndContext(t *testing.T) {
	t.Parallel()

	h := RequestIDMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID, ok := RequestIDFromContext(r.Context())
		if !ok || requestID == "" {
			t.Fatalf("expected request id in context")
		}
	}))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	h.ServeHTTP(rr, req)

	requestID := rr.Header().Get("X-Request-ID")
	if requestID == "" {
		t.Fatal("expected X-Request-ID header")
	}
	if _, err := uuid.Parse(requestID); err != nil {
		t.Fatalf("expected UUID request id, got %q: %v", requestID, err)
	}
}

func TestRecoveryMiddlewareReturnsInternalServerErrorWithoutPanicDetails(t *testing.T) {
	t.Parallel()

	buf := bytes.NewBuffer(nil)
	logger := logging.New(buf, "production", "info")

	h := RequestIDMiddleware()(RecoveryMiddleware(logger)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("sensitive panic content")
	})))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("unexpected status code: got %d want %d", rr.Code, http.StatusInternalServerError)
	}

	body := rr.Body.String()
	if strings.Contains(body, "sensitive panic content") {
		t.Fatalf("panic details leaked in response body: %q", body)
	}
	if strings.TrimSpace(body) != http.StatusText(http.StatusInternalServerError) {
		t.Fatalf("unexpected response body: got %q want %q", strings.TrimSpace(body), http.StatusText(http.StatusInternalServerError))
	}

	output := buf.String()
	if !strings.Contains(output, "http_panic_recovered") {
		t.Fatalf("expected panic recovery log entry: %s", output)
	}
	if strings.Contains(output, "sensitive panic content") {
		t.Fatalf("panic content must not be logged: %s", output)
	}
}

func TestRecoveryMiddlewareDoesNotWriteFallback500AfterCommit(t *testing.T) {
	t.Parallel()

	buf := bytes.NewBuffer(nil)
	logger := logging.New(buf, "production", "info")

	h := RequestIDMiddleware()(RecoveryMiddleware(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("partial-response"))
		panic("panic after commit")
	})))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("unexpected status code after committed panic: got %d want %d", rr.Code, http.StatusOK)
	}
	if got := rr.Body.String(); got != "partial-response" {
		t.Fatalf("unexpected body after committed panic: got %q", got)
	}
}

func TestSecurityHeadersMiddlewareSetsHeaders(t *testing.T) {
	t.Parallel()

	h := SecurityHeadersMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	h.ServeHTTP(rr, req)

	assertHeader(t, rr, "X-Frame-Options", "DENY")
	assertHeader(t, rr, "X-Content-Type-Options", "nosniff")
	assertHeader(t, rr, "Referrer-Policy", "strict-origin-when-cross-origin")
	assertHeader(t, rr, "Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data:; connect-src 'self'; frame-ancestors 'none';")
}

func assertHeader(t *testing.T, rr *httptest.ResponseRecorder, key string, want string) {
	t.Helper()
	if got := rr.Header().Get(key); got != want {
		t.Fatalf("unexpected %s header: got %q want %q", key, got, want)
	}
}

func TestSafeLoggingMiddlewareDoesNotLogHeadersOrBody(t *testing.T) {
	t.Parallel()

	buf := bytes.NewBuffer(nil)
	logger := logging.New(buf, "production", "info")

	h := RequestIDMiddleware()(SafeLoggingMiddleware(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("response-body-secret"))
	})))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("Authorization", "Bearer top-secret-token")
	h.ServeHTTP(rr, req)

	output := buf.String()
	if strings.Contains(output, "top-secret-token") || strings.Contains(output, "Authorization") {
		t.Fatalf("headers were leaked in logs: %s", output)
	}
	if strings.Contains(output, "response-body-secret") {
		t.Fatalf("response body was leaked in logs: %s", output)
	}

	var event map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &event); err != nil {
		t.Fatalf("expected json log output: %v", err)
	}
	if _, ok := event["method"]; !ok {
		t.Fatalf("expected method field in log event: %v", event)
	}
}

func TestSafeLoggingMiddlewareLogsPathWithoutQuery(t *testing.T) {
	t.Parallel()

	buf := bytes.NewBuffer(nil)
	logger := logging.New(buf, "production", "info")

	h := RequestIDMiddleware()(SafeLoggingMiddleware(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health?token=super-secret", nil)
	h.ServeHTTP(rr, req)

	output := buf.String()
	if !strings.Contains(output, `"path":"/health"`) {
		t.Fatalf("expected path without query in logs: %s", output)
	}
	if strings.Contains(output, "super-secret") {
		t.Fatalf("unexpected query value in logs: %s", output)
	}
	if !strings.Contains(output, `"request_id":"`) {
		t.Fatalf("expected request_id in logs: %s", output)
	}
}

func TestSafeLoggingMiddlewareLogsRequestOnPanic(t *testing.T) {
	t.Parallel()

	buf := bytes.NewBuffer(nil)
	logger := logging.New(buf, "production", "info")

	h := RequestIDMiddleware()(RecoveryMiddleware(logger)(SafeLoggingMiddleware(logger)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("fail")
	}))))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	h.ServeHTTP(rr, req)

	output := buf.String()
	if !strings.Contains(output, "http_panic_recovered") {
		t.Fatalf("expected panic recovery log entry: %s", output)
	}
	if !strings.Contains(output, `"msg":"http_request"`) {
		t.Fatalf("expected request log entry for panic request: %s", output)
	}
	if !strings.Contains(output, `"status":500`) {
		t.Fatalf("expected 500 status in request log for panic request: %s", output)
	}
}

func TestReverseProxyAuthMiddlewareRejectsMissingHeader(t *testing.T) {
	t.Parallel()

	h := ReverseProxyAuthMiddleware("X-Forwarded-User")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/today", nil)
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("unexpected status code: got %d want %d", rr.Code, http.StatusForbidden)
	}
}

func TestReverseProxyAuthMiddlewareAcceptsNonEmptyHeader(t *testing.T) {
	t.Parallel()

	h := ReverseProxyAuthMiddleware("X-Forwarded-User")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/today", nil)
	req.Header.Set("X-Forwarded-User", "alice")
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("unexpected status code: got %d want %d", rr.Code, http.StatusNoContent)
	}
}

func TestReverseProxyAuthMiddlewareAllowsHealthWithoutAuth(t *testing.T) {
	t.Parallel()

	h := ReverseProxyAuthMiddleware("X-Forwarded-User")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("unexpected status code: got %d want %d", rr.Code, http.StatusNoContent)
	}
}

func TestSetupGateMiddlewareRedirectsDisallowedRoutesWhenSetupIncomplete(t *testing.T) {
	t.Parallel()

	h := SetupGateMiddleware(NewSetupState(false), assets.Manifest{"app.css": "app.8f3a1c2.css", "htmx.min.js": "htmx.5e741aa.min.js"})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/today", nil)
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusFound {
		t.Fatalf("unexpected status code: got %d want %d", rr.Code, http.StatusFound)
	}
	if got := rr.Header().Get("Location"); got != "/setup" {
		t.Fatalf("unexpected location header: got %q want %q", got, "/setup")
	}
}

func TestSetupGateMiddlewareAllowsStaticAssetsWhenSetupIncomplete(t *testing.T) {
	t.Parallel()

	h := SetupGateMiddleware(NewSetupState(false), assets.Manifest{"app.css": "app.8f3a1c2.css", "htmx.min.js": "htmx.5e741aa.min.js"})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/static/htmx.5e741aa.min.js", nil)
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("unexpected status code: got %d want %d", rr.Code, http.StatusNoContent)
	}
}
func TestSetupGateMiddlewareRedirectsUnknownStaticAssetWhenSetupIncomplete(t *testing.T) {
	t.Parallel()

	h := SetupGateMiddleware(NewSetupState(false), assets.Manifest{"app.css": "app.8f3a1c2.css"})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/static/unknown.js", nil)
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusFound {
		t.Fatalf("unexpected status code: got %d want %d", rr.Code, http.StatusFound)
	}
}

func TestSetupGateMiddlewareAllowsSetupRoutesWhenSetupIncomplete(t *testing.T) {
	t.Parallel()

	h := SetupGateMiddleware(NewSetupState(false), assets.Manifest{"app.css": "app.8f3a1c2.css", "htmx.min.js": "htmx.5e741aa.min.js"})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/setup/import/events", nil)
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("unexpected status code: got %d want %d", rr.Code, http.StatusNoContent)
	}
}

func TestSetupGateMiddlewareAllowsAllRoutesWhenSetupComplete(t *testing.T) {
	t.Parallel()

	h := SetupGateMiddleware(NewSetupState(true), assets.Manifest{"app.css": "app.8f3a1c2.css"})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/today", nil)
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("unexpected status code: got %d want %d", rr.Code, http.StatusNoContent)
	}
}

func TestSetupGateMiddlewareReflectsRuntimeCompletionState(t *testing.T) {
	t.Parallel()

	state := NewSetupState(false)
	h := SetupGateMiddleware(state, assets.Manifest{"app.css": "app.8f3a1c2.css"})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/today", nil)
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusFound {
		t.Fatalf("unexpected status code before completion: got %d want %d", rr.Code, http.StatusFound)
	}

	state.MarkComplete()

	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/today", nil)
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("unexpected status code after completion: got %d want %d", rr.Code, http.StatusNoContent)
	}
}

func TestSetupCSRFMiddlewareRejectsMutatingRequestWithoutToken(t *testing.T) {
	t.Parallel()

	h := SetupCSRFMiddleware([]byte("12345678901234567890123456789012"))(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/setup/caldav", nil)
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("unexpected status code: got %d want %d", rr.Code, http.StatusForbidden)
	}
}

func TestSetupCSRFMiddlewareExposesTokenOnSetupPageResponses(t *testing.T) {
	t.Parallel()

	var tokenFromContext string
	h := SetupCSRFMiddleware([]byte("12345678901234567890123456789012"))(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenFromContext = view.CSRFToken(r.Context())
		w.WriteHeader(http.StatusNoContent)
	}))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/setup/", nil)
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("unexpected status code: got %d want %d", rr.Code, http.StatusNoContent)
	}

	headerToken := rr.Header().Get(csrfHeaderName)
	if headerToken == "" {
		t.Fatal("expected csrf token in response header")
	}
	if tokenFromContext == "" {
		t.Fatal("expected csrf token in request context")
	}
	if tokenFromContext != headerToken {
		t.Fatalf("csrf token mismatch between context and header: got context %q header %q", tokenFromContext, headerToken)
	}

	cookies := rr.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("unexpected cookies count: got %d want %d", len(cookies), 1)
	}

	cookie := cookies[0]
	if cookie.Name != csrfCookieName {
		t.Fatalf("unexpected cookie name: got %q want %q", cookie.Name, csrfCookieName)
	}
	if cookie.Value != headerToken {
		t.Fatalf("csrf token mismatch between cookie and header: got cookie %q header %q", cookie.Value, headerToken)
	}
	if !cookie.HttpOnly {
		t.Fatal("expected csrf cookie to be httpOnly")
	}
}
