package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthReturnsLivenessOnly(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	responseRecorder := httptest.NewRecorder()

	Health(responseRecorder, request)

	response := responseRecorder.Result()
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status code: got %d want %d", response.StatusCode, http.StatusOK)
	}

	if got := response.Header.Get("Content-Type"); got != "application/json" {
		t.Fatalf("unexpected content type: got %q want %q", got, "application/json")
	}

	var payload healthResponse
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response body: %v", err)
	}

	if payload.Status != "ok" {
		t.Fatalf("unexpected status payload: got %q want %q", payload.Status, "ok")
	}
}
