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
	allRows := render.BuildTaskRows(data.Tasks, data.Lists)
	rows := filterRows(allRows, activeView(r.URL.Query().Get("view")), r.URL.Query().Get("context"), r.URL.Query().Get("goal"))
	vm := render.TaskPageViewModel{
		PrincipalID:    principal,
		Lists:          render.BuildTaskLists(data.Lists, data.ActiveListID),
		Contexts:       render.BuildContexts(allRows),
		Goals:          render.BuildGoals(allRows),
		ActiveListID:   data.ActiveListID,
		ActiveView:     activeView(r.URL.Query().Get("view")),
		Rows:           rows,
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

func filterRows(rows []render.TaskRow, view, contextValue, goalValue string) []render.TaskRow {
	if view != "contexts" && view != "goals" {
		return rows
	}
	ctx := strings.TrimSpace(contextValue)
	goal := strings.TrimSpace(goalValue)
	if view == "contexts" && ctx == "" {
		return rows
	}
	if view == "goals" && goal == "" {
		return rows
	}
	out := make([]render.TaskRow, 0, len(rows))
	for _, row := range rows {
		if view == "contexts" && strings.EqualFold(row.Context, ctx) {
			out = append(out, row)
		}
		if view == "goals" && strings.EqualFold(row.Goal, goal) {
			out = append(out, row)
		}
	}
	return out
}
