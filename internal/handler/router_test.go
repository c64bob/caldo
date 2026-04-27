package handler

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"caldo/internal/logging"
)

func TestNewRouterExposesHealthWithoutAuth(t *testing.T) {
	t.Parallel()

	logger := logging.New(bytes.NewBuffer(nil), "production", "info")
	responseRecorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/health", nil)

	NewRouter(logger).ServeHTTP(responseRecorder, request)

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
