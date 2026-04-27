package handler

import (
	"net/http"

	"caldo/internal/view"
)

// Home renders a minimal example page using the base layout.
func Home(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := view.BaseLayout("Caldo", view.EmptyContent()).Render(r.Context(), w); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}
