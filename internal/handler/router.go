package handler

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// NewRouter returns the HTTP router for Caldo.
func NewRouter(logger *slog.Logger) http.Handler {
	router := chi.NewRouter()
	router.Use(RequestIDMiddleware())
	router.Use(SafeLoggingMiddleware(logger))
	router.Get("/health", Health)

	return router
}
