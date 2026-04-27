package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// NewRouter returns the HTTP router for Caldo.
func NewRouter() http.Handler {
	router := chi.NewRouter()
	router.Get("/health", Health)

	return router
}
