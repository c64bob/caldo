package handlers

import (
	"net/http"

	"caldo/internal/http/middleware"
	"caldo/internal/http/render"
)

func (h *TasksHandler) HTMXTasksList(w http.ResponseWriter, r *http.Request) {
	principal, ok := middleware.PrincipalFromContext(r.Context())
	if !ok {
		http.Error(w, "Authentifizierungsheader fehlt", http.StatusUnauthorized)
		return
	}

	data, err := h.Service.LoadTaskPage(r.Context(), principal, r.URL.Query().Get("list"))
	if err != nil {
		message, status := taskLoadError(err)
		http.Error(w, message, status)
		return
	}

	vm := render.TaskPageViewModel{
		PrincipalID:    principal,
		Lists:          render.BuildTaskLists(data.Lists, data.ActiveListID),
		ActiveListID:   data.ActiveListID,
		Rows:           render.BuildTaskRows(data.Tasks),
		HasCredentials: data.HasCredentials,
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.Templates.RenderTasksList(w, vm); err != nil {
		http.Error(w, "Template-Rendering fehlgeschlagen", http.StatusInternalServerError)
		return
	}
}
