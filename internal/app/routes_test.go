package app

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"caldo/internal/service"
)

func TestNewRouter_HealthEndpoint(t *testing.T) {
	r := NewRouter(Config{}, &service.SettingsService{}, &service.TaskService{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if rr.Body.String() != "ok" {
		t.Fatalf("expected body ok, got %q", rr.Body.String())
	}
}

func TestNewRouter_RootRedirectsToTasks(t *testing.T) {
	r := NewRouter(Config{}, &service.SettingsService{}, &service.TaskService{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d", rr.Code)
	}
	if got := rr.Header().Get("Location"); got != "/tasks" {
		t.Fatalf("expected redirect to /tasks, got %q", got)
	}
}

func TestNewRouter_SettingsRequiresProxyHeader(t *testing.T) {
	cfg := Config{}
	cfg.Server.AuthHeader = "X-Forwarded-User"
	r := NewRouter(cfg, &service.SettingsService{}, &service.TaskService{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/settings", nil)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}
