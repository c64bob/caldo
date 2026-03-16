package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"caldo/internal/http/middleware"
)

func TestMutateTask_NonHTMXRedirectsWithLocation(t *testing.T) {
	h := &TasksHandler{}
	handler := middleware.ProxyAuth("X-Forwarded-User")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.mutateTask(w, r, func(_ string) error { return nil })
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/tasks/update", strings.NewReader("list_id=tasks"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Forwarded-User", "alice@example.com")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("expected status 303, got %d", rr.Code)
	}
	if got := rr.Header().Get("Location"); got != "/tasks?list=tasks" {
		t.Fatalf("expected redirect location /tasks?list=tasks, got %q", got)
	}
}

func TestMutateTask_HTMXSetsHXRedirect(t *testing.T) {
	h := &TasksHandler{}
	handler := middleware.ProxyAuth("X-Forwarded-User")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.mutateTask(w, r, func(_ string) error { return nil })
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/tasks/update", strings.NewReader("list_id=tasks"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Forwarded-User", "alice@example.com")
	req.Header.Set("HX-Request", "true")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
	if got := rr.Header().Get("HX-Redirect"); got != "/tasks?list=tasks" {
		t.Fatalf("expected HX-Redirect /tasks?list=tasks, got %q", got)
	}
}
