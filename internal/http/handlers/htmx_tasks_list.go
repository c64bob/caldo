package handlers

import (
	"net/http"

	"caldo/internal/http/middleware"
)

func (h *TasksHandler) HTMXTasksList(w http.ResponseWriter, r *http.Request) {
	principal, ok := middleware.PrincipalFromContext(r.Context())
	if !ok {
		http.Error(w, "Authentifizierungsheader fehlt", http.StatusUnauthorized)
		return
	}

	vm, err := h.buildTaskVM(r, principal)
	if err != nil {
		message, status := taskLoadError(err)
		http.Error(w, message, status)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.Templates.RenderTasksList(w, vm); err != nil {
		http.Error(w, "Template-Rendering fehlgeschlagen", http.StatusInternalServerError)
		return
	}
}
