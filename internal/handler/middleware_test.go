package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"caldo/internal/logging"
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
