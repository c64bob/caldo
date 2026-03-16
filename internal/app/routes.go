package app

import (
	"net/http"

	"caldo/internal/http/handlers"
	"caldo/internal/http/middleware"
	"caldo/internal/service"
)

func NewRouter(cfg Config, settingsSvc *service.SettingsService) http.Handler {
	mux := http.NewServeMux()
	settingsHandler := &handlers.SettingsHandler{
		Service:          settingsSvc,
		DefaultServerURL: cfg.CalDAV.ServerURL,
	}

	mux.HandleFunc("GET /health", handlers.Health)
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/settings", http.StatusFound)
	})
	mux.Handle("GET /settings", middleware.ProxyAuth(cfg.Server.AuthHeader)(http.HandlerFunc(settingsHandler.Page)))
	mux.Handle("POST /settings/dav-account", middleware.ProxyAuth(cfg.Server.AuthHeader)(http.HandlerFunc(settingsHandler.SaveDAVAccount)))

	return mux
}
