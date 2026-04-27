package handler

import (
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"caldo/internal/assets"
	"caldo/internal/view"
	"github.com/go-chi/chi/v5"
)

const staticAssetsCacheControl = "public, max-age=31536000, immutable"

var staticAssetsRoot = defaultStaticAssetsRoot()

// NewRouter returns the HTTP router for Caldo.
func NewRouter(logger *slog.Logger, proxyUserHeader string, manifest assets.Manifest) http.Handler {
	router := chi.NewRouter()
	router.Use(RequestIDMiddleware())
	router.Use(RecoveryMiddleware(logger))
	router.Use(SafeLoggingMiddleware(logger))
	router.Use(SecurityHeadersMiddleware())
	router.Use(ReverseProxyAuthMiddleware(proxyUserHeader))
	router.Use(AssetManifestMiddleware(manifest))

	router.Get("/health", Health)
	router.Get("/", Home)
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
	executablePath, err := os.Executable()
	if err == nil {
		executableDir := filepath.Dir(executablePath)
		candidate := filepath.Clean(filepath.Join(executableDir, "web", "static"))
		if directoryExists(candidate) {
			return candidate
		}
	}

	workingDir, err := os.Getwd()
	if err == nil {
		currentDir := workingDir
		for range 8 {
			candidate := filepath.Clean(filepath.Join(currentDir, "web", "static"))
			if directoryExists(candidate) {
				return candidate
			}

			parentDir := filepath.Dir(currentDir)
			if parentDir == currentDir {
				break
			}
			currentDir = parentDir
		}
	}

	return filepath.Clean(filepath.Join("web", "static"))
}

func directoryExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	return info.IsDir()
}

// AssetManifestMiddleware injects static asset resolution data and CSRF token into request context.
func AssetManifestMiddleware(manifest assets.Manifest) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := view.WithAssetManifest(r.Context(), manifest)
			ctx = view.WithCSRFToken(ctx, "")
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
