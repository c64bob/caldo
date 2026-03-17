package handlers

import (
	"net/http"
	"strings"

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
		logTaskLoadError("tasks.page", principal, r.URL.Query().Get("list"), err)
		message, status := taskLoadError(err)
		http.Error(w, message, status)
		return
	}
	vm := render.TaskPageViewModel{
		PrincipalID:    principal,
		Lists:          render.BuildTaskLists(data.Lists, data.ActiveListID),
		ActiveListID:   data.ActiveListID,
		ActiveView:     activeView(r.URL.Query().Get("view")),
		Rows:           render.BuildTaskRows(data.Tasks, data.Lists),
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

func activeView(view string) string {
	v := strings.TrimSpace(view)
	if v == "" {
		return "main"
	}
	return v
}
