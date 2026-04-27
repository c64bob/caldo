package handler

import (
	"log/slog"
	"net/http"
	"path/filepath"
	"runtime"

	"github.com/go-chi/chi/v5"
)

const staticAssetsCacheControl = "public, max-age=31536000, immutable"

var staticAssetsRoot = defaultStaticAssetsRoot()

// NewRouter returns the HTTP router for Caldo.
func NewRouter(logger *slog.Logger, proxyUserHeader string) http.Handler {
	router := chi.NewRouter()
	router.Use(RequestIDMiddleware())
	router.Use(RecoveryMiddleware(logger))
	router.Use(SafeLoggingMiddleware(logger))
	router.Use(SecurityHeadersMiddleware())
	router.Use(ReverseProxyAuthMiddleware(proxyUserHeader))

	router.Get("/health", Health)
	router.Handle("/static/*", staticFileServer(staticAssetsRoot))

	return router
}

func staticFileServer(root string) http.Handler {
	fileServer := http.FileServer(http.Dir(root))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", staticAssetsCacheControl)
		http.StripPrefix("/static", fileServer).ServeHTTP(w, r)
	})
}

func defaultStaticAssetsRoot() string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return "web/static"
	}

	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "..", "web", "static"))
}
