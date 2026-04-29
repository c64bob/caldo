package handler

import (
	"database/sql"
	"errors"
	"net/http"

	"caldo/internal/db"
	"caldo/internal/view"
	"github.com/go-chi/chi/v5"
)

type conflictDependencies struct {
	database *db.Database
}

// Conflicts renders the global unresolved conflict list.
func Conflicts(deps conflictDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		results, err := deps.database.ListUnresolvedConflicts(r.Context())
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := view.BaseLayout("Konflikte", view.ConflictListPage(results)).Render(r.Context(), w); err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
	}
}

// ConflictDetail renders one unresolved conflict detail view.
func ConflictDetail(deps conflictDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conflictID := chi.URLParam(r, "conflictID")
		detail, err := deps.database.GetUnresolvedConflictByID(r.Context(), conflictID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				http.NotFound(w, r)
				return
			}
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := view.BaseLayout("Konfliktdetail", view.ConflictDetailPage(detail)).Render(r.Context(), w); err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
	}
}
