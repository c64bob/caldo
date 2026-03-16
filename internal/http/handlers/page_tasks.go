package handlers

import (
	"net/http"

	"caldo/internal/http/middleware"
	"caldo/internal/http/render"
	"caldo/internal/service"
)

type TasksHandler struct {
	Service     *service.TaskService
	SyncService *service.SyncService
	Templates   *render.Templates
}

func (h *TasksHandler) Page(w http.ResponseWriter, r *http.Request) {
	principal, ok := middleware.PrincipalFromContext(r.Context())
	if !ok {
		http.Error(w, "Authentifizierungsheader fehlt", http.StatusUnauthorized)
		return
	}

	data, err := h.Service.LoadTaskPage(r.Context(), principal, r.URL.Query().Get("list"))
	if err != nil {
		http.Error(w, "Tasks konnten nicht geladen werden", http.StatusBadGateway)
		return
	}
	vm := render.TaskPageViewModel{
		PrincipalID:    principal,
		Lists:          render.BuildTaskLists(data.Lists, data.ActiveListID),
		ActiveListID:   data.ActiveListID,
		Rows:           render.BuildTaskRows(data.Tasks),
		HasCredentials: data.HasCredentials,
	}
	if !data.HasCredentials {
		vm.Error = "Kein DAV-Account hinterlegt. Bitte zuerst unter Einstellungen verbinden."
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.Templates.RenderTasksPage(w, vm); err != nil {
		http.Error(w, "Template-Rendering fehlgeschlagen", http.StatusInternalServerError)
		return
	}
}
