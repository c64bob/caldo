package app

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"caldo/internal/http/render"
	"caldo/internal/service"
)

func TestNewRouter_HealthEndpoint(t *testing.T) {
	r := NewRouter(Config{}, &service.SettingsService{}, &service.PreferencesService{}, &service.SavedFiltersService{}, &service.TaskService{}, &service.SyncService{}, (*render.Templates)(nil))
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
	r := NewRouter(Config{}, &service.SettingsService{}, &service.PreferencesService{}, &service.SavedFiltersService{}, &service.TaskService{}, &service.SyncService{}, (*render.Templates)(nil))
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
	r := NewRouter(cfg, &service.SettingsService{}, &service.PreferencesService{}, &service.SavedFiltersService{}, &service.TaskService{}, &service.SyncService{}, (*render.Templates)(nil))
	req := httptest.NewRequest(http.MethodGet, "/settings", nil)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestResolveStaticRoot_PrefersWorkingDirectory(t *testing.T) {
	base := t.TempDir()
	staticRoot := filepath.Join(base, "web", "static", "css")
	if err := os.MkdirAll(staticRoot, 0o755); err != nil {
		t.Fatalf("mkdir static dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(staticRoot, "app.css"), []byte(":root{}"), 0o600); err != nil {
		t.Fatalf("write app.css: %v", err)
	}

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(base); err != nil {
		t.Fatalf("chdir to temp dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(originalWD)
	})

	resolved := resolveStaticRoot()
	expected := filepath.Join(base, "web", "static")
	if resolved != expected {
		t.Fatalf("expected %q, got %q", expected, resolved)
	}
}

func TestNewRouter_ServesTaskStaticAlias(t *testing.T) {
	base := t.TempDir()
	staticRoot := filepath.Join(base, "web", "static", "css")
	if err := os.MkdirAll(staticRoot, 0o755); err != nil {
		t.Fatalf("mkdir static dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(staticRoot, "app.css"), []byte("body{}"), 0o600); err != nil {
		t.Fatalf("write app.css: %v", err)
	}

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(base); err != nil {
		t.Fatalf("chdir to temp dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(originalWD)
	})

	r := NewRouter(Config{}, &service.SettingsService{}, &service.PreferencesService{}, &service.SavedFiltersService{}, &service.TaskService{}, &service.SyncService{}, (*render.Templates)(nil))
	req := httptest.NewRequest(http.MethodGet, "/tasks/static/css/app.css", nil)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}
