package handler

import (
	"bytes"
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
