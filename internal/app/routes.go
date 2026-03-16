package app

import (
	"net/http"

	"caldo/internal/http/handlers"
	"caldo/internal/http/middleware"
	"caldo/internal/http/render"
	"caldo/internal/service"
)

func NewRouter(cfg Config, settingsSvc *service.SettingsService, taskSvc *service.TaskService, syncSvc *service.SyncService, templates *render.Templates) http.Handler {
	mux := http.NewServeMux()
	settingsHandler := &handlers.SettingsHandler{
		Service:          settingsSvc,
		DefaultServerURL: cfg.CalDAV.ServerURL,
	}
	tasksHandler := &handlers.TasksHandler{Service: taskSvc, SyncService: syncSvc, Templates: templates}

	mux.HandleFunc("GET /health", handlers.Health)
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/tasks", http.StatusFound)
	})
	mux.Handle("GET /settings", middleware.ProxyAuth(cfg.Server.AuthHeader)(http.HandlerFunc(settingsHandler.Page)))
	mux.Handle("POST /settings/dav-account", middleware.ProxyAuth(cfg.Server.AuthHeader)(http.HandlerFunc(settingsHandler.SaveDAVAccount)))
	mux.Handle("GET /tasks", middleware.ProxyAuth(cfg.Server.AuthHeader)(http.HandlerFunc(tasksHandler.Page)))
	mux.Handle("GET /htmx/tasks/list", middleware.ProxyAuth(cfg.Server.AuthHeader)(http.HandlerFunc(tasksHandler.HTMXTasksList)))
	mux.Handle("GET /htmx/sidebar/lists", middleware.ProxyAuth(cfg.Server.AuthHeader)(http.HandlerFunc(tasksHandler.HTMXSidebarLists)))
	mux.Handle("POST /api/tasks", middleware.ProxyAuth(cfg.Server.AuthHeader)(http.HandlerFunc(tasksHandler.APITaskCreate)))
	mux.Handle("POST /api/tasks/update", middleware.ProxyAuth(cfg.Server.AuthHeader)(http.HandlerFunc(tasksHandler.APITaskUpdate)))
	mux.Handle("POST /api/tasks/delete", middleware.ProxyAuth(cfg.Server.AuthHeader)(http.HandlerFunc(tasksHandler.APITaskDelete)))
	mux.Handle("POST /api/sync/now", middleware.ProxyAuth(cfg.Server.AuthHeader)(http.HandlerFunc(tasksHandler.APISyncNow)))

	return mux
}
