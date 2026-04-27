package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewRouterExposesHealthWithoutAuth(t *testing.T) {
	t.Parallel()

	responseRecorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/health", nil)

	NewRouter().ServeHTTP(responseRecorder, request)

	if responseRecorder.Code != http.StatusOK {
		t.Fatalf("unexpected status code: got %d want %d", responseRecorder.Code, http.StatusOK)
	}
}
