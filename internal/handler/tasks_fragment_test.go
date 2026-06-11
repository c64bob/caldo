package handler

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"caldo/internal/logging"
)

func TestTaskFragmentRouteRendersCurrentTaskRow(t *testing.T) {
	t.Parallel()

	database := openDateViewRouteDB(t)
	seedDateViewRouteTasks(t, database)

	logger := logging.New(bytes.NewBuffer(nil), "production", "info")
	req := httptest.NewRequest(http.MethodGet, "/tasks/task-today-active", nil)
	req.Header.Set("X-Forwarded-User", "alice")
	rr := httptest.NewRecorder()

	NewRouter(logger, "X-Forwarded-User", testManifest(t), true, []byte("12345678901234567890123456789012"), database, context.Background(), nil).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("unexpected status: got %d body=%q", rr.Code, rr.Body.String())
	}
	body := rr.Body.String()
	for _, want := range []string{
		`data-task-id="task-today-active"`,
		`data-server-version="3"`,
		`name="expected_version" value="3"`,
		`Heute Aufgabe`,
		`Heute Beschreibung`,
		`Work`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("fragment missing %q in %s", want, body)
		}
	}
}

func TestTaskFragmentRouteReturnsNotFoundForMissingTask(t *testing.T) {
	t.Parallel()

	database := openDateViewRouteDB(t)
	logger := logging.New(bytes.NewBuffer(nil), "production", "info")
	req := httptest.NewRequest(http.MethodGet, "/tasks/missing", nil)
	req.Header.Set("X-Forwarded-User", "alice")
	rr := httptest.NewRecorder()

	NewRouter(logger, "X-Forwarded-User", testManifest(t), true, []byte("12345678901234567890123456789012"), database, context.Background(), nil).ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("unexpected status: got %d body=%q", rr.Code, rr.Body.String())
	}
}
