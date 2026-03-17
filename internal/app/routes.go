package app

import (
	"net/http"
	"os"
	"path/filepath"

	"caldo/internal/http/handlers"
	"caldo/internal/http/middleware"
	"caldo/internal/http/render"
	"caldo/internal/service"
)

func NewRouter(cfg Config, settingsSvc *service.SettingsService, prefsSvc *service.PreferencesService, taskSvc *service.TaskService, syncSvc *service.SyncService, templates *render.Templates) http.Handler {
	mux := http.NewServeMux()
	settingsHandler := &handlers.SettingsHandler{
		Service:            settingsSvc,
		PreferencesService: prefsSvc,
		TaskService:        taskSvc,
		DefaultServerURL:   cfg.CalDAV.ServerURL,
	}
	tasksHandler := &handlers.TasksHandler{Service: taskSvc, PreferencesService: prefsSvc, SyncService: syncSvc, Templates: templates}

	mux.HandleFunc("GET /health", handlers.Health)
	staticHandler := http.FileServer(http.Dir(resolveStaticRoot()))
	mux.Handle("GET /static/", http.StripPrefix("/static/", staticHandler))
	mux.Handle("GET /tasks/static/", http.StripPrefix("/tasks/static/", staticHandler))
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/tasks", http.StatusFound)
	})
	mux.Handle("GET /settings", middleware.ProxyAuth(cfg.Server.AuthHeader)(http.HandlerFunc(settingsHandler.Page)))
	mux.Handle("POST /settings/dav-account", middleware.ProxyAuth(cfg.Server.AuthHeader)(http.HandlerFunc(settingsHandler.SaveDAVAccount)))
	mux.Handle("POST /settings/preferences", middleware.ProxyAuth(cfg.Server.AuthHeader)(http.HandlerFunc(settingsHandler.SavePreferences)))
	mux.Handle("POST /settings/collections", middleware.ProxyAuth(cfg.Server.AuthHeader)(http.HandlerFunc(settingsHandler.CreateCollection)))
	mux.Handle("GET /tasks", middleware.ProxyAuth(cfg.Server.AuthHeader)(http.HandlerFunc(tasksHandler.Page)))
	mux.Handle("GET /htmx/tasks/list", middleware.ProxyAuth(cfg.Server.AuthHeader)(http.HandlerFunc(tasksHandler.HTMXTasksList)))
	mux.Handle("GET /htmx/sidebar/lists", middleware.ProxyAuth(cfg.Server.AuthHeader)(http.HandlerFunc(tasksHandler.HTMXSidebarLists)))
	mux.Handle("POST /api/tasks", middleware.ProxyAuth(cfg.Server.AuthHeader)(http.HandlerFunc(tasksHandler.APITaskCreate)))
	mux.Handle("POST /api/tasks/quick-add", middleware.ProxyAuth(cfg.Server.AuthHeader)(http.HandlerFunc(tasksHandler.APITaskQuickAdd)))
	mux.Handle("POST /api/tasks/update", middleware.ProxyAuth(cfg.Server.AuthHeader)(http.HandlerFunc(tasksHandler.APITaskUpdate)))
	mux.Handle("POST /api/tasks/delete", middleware.ProxyAuth(cfg.Server.AuthHeader)(http.HandlerFunc(tasksHandler.APITaskDelete)))
	mux.Handle("POST /api/sync/now", middleware.ProxyAuth(cfg.Server.AuthHeader)(http.HandlerFunc(tasksHandler.APISyncNow)))

	return mux
}

func resolveStaticRoot() string {
	const standardStaticRoot = "/app/web/static"

	candidates := make([]string, 0, 6)
	if wd, err := os.Getwd(); err == nil {
		candidates = append(candidates, filepath.Join(wd, "web", "static"))
	}
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		candidates = append(candidates,
			filepath.Join(exeDir, "web", "static"),
			filepath.Join(exeDir, "..", "web", "static"),
			filepath.Join(exeDir, "..", "..", "web", "static"),
		)
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(filepath.Join(candidate, filepath.FromSlash("css/app.css"))); err == nil {
			return candidate
		}
	}

	if _, err := os.Stat(filepath.Join(standardStaticRoot, filepath.FromSlash("css/app.css"))); err == nil {
		return standardStaticRoot
	}

	return standardStaticRoot
}
