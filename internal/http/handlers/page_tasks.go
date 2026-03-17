package handlers

import (
	"net/http"
	"sort"
	"strings"
	"time"

	"caldo/internal/http/middleware"
	"caldo/internal/http/render"
	"caldo/internal/service"
)

type TasksHandler struct {
	Service            *service.TaskService
	PreferencesService *service.PreferencesService
	SyncService        *service.SyncService
	Templates          *render.Templates
}

func (h *TasksHandler) Page(w http.ResponseWriter, r *http.Request) {
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
	if err := h.Templates.RenderTasksPage(w, vm); err != nil {
		http.Error(w, "Template-Rendering fehlgeschlagen", http.StatusInternalServerError)
		return
	}
}

func (h *TasksHandler) buildTaskVM(r *http.Request, principal string) (render.TaskPageViewModel, error) {
	data, err := h.Service.LoadTaskPage(r.Context(), principal, r.URL.Query().Get("list"))
	if err != nil {
		logTaskLoadError("tasks.build_vm", principal, r.URL.Query().Get("list"), err)
		return render.TaskPageViewModel{}, err
	}
	allRows := render.BuildTaskRows(data.Tasks, data.Lists)
	view := activeView(r.URL.Query().Get("view"))
	rows := filterRows(allRows, view, r.URL.Query().Get("context"), r.URL.Query().Get("goal"), r.URL.Query().Get("q"))
	vm := render.TaskPageViewModel{
		PrincipalID:    principal,
		Lists:          render.BuildTaskLists(data.Lists, data.ActiveListID),
		Contexts:       render.BuildContexts(allRows),
		Goals:          render.BuildGoals(allRows),
		ActiveListID:   data.ActiveListID,
		ActiveView:     view,
		ActiveContext:  strings.TrimSpace(r.URL.Query().Get("context")),
		ActiveGoal:     strings.TrimSpace(r.URL.Query().Get("goal")),
		Query:          strings.TrimSpace(r.URL.Query().Get("q")),
		Rows:           rows,
		HasCredentials: data.HasCredentials,
		VisibleColumns: map[string]bool{},
	}
	if h.PreferencesService != nil {
		prefs, prefErr := h.PreferencesService.GetOrDefault(r.Context(), principal)
		if prefErr == nil {
			if strings.TrimSpace(r.URL.Query().Get("view")) == "" && strings.TrimSpace(prefs.DefaultView) != "" {
				vm.ActiveView = prefs.DefaultView
				vm.Rows = filterRows(allRows, vm.ActiveView, vm.ActiveContext, vm.ActiveGoal, vm.Query)
			}
			if vm.VisibleColumns == nil {
				vm.VisibleColumns = map[string]bool{}
			}
			for _, col := range prefs.VisibleColumns {
				vm.VisibleColumns[col] = true
			}
		}
	}
	if !data.HasCredentials {
		vm.Error = "Kein DAV-Account hinterlegt. Bitte zuerst unter Einstellungen verbinden."
	}
	return vm, nil
}

func activeView(view string) string {
	v := strings.TrimSpace(view)
	if v == "" {
		return "main"
	}
	return v
}

func filterRows(rows []render.TaskRow, view, contextValue, goalValue, query string) []render.TaskRow {
	ctx := strings.TrimSpace(contextValue)
	goal := strings.TrimSpace(goalValue)
	q := strings.ToLower(strings.TrimSpace(query))
	now := time.Now()
	out := make([]render.TaskRow, 0, len(rows))
	for _, row := range rows {
		if view == "main" && row.IsCompleted {
			continue
		}
		if view == "contexts" && ctx != "" && !strings.EqualFold(row.Context, ctx) {
			continue
		}
		if view == "goals" && goal != "" && !strings.EqualFold(row.Goal, goal) {
			continue
		}
		if view == "hotlist" {
			if !(row.IsStarred || row.Priority >= 6 || isDueSoon(row.DueInput, now)) {
				continue
			}
		}
		if view == "due" && strings.TrimSpace(row.DueInput) == "" {
			continue
		}
		if view == "priority" && row.Priority < 1 {
			continue
		}
		if view == "search" && q == "" {
			continue
		}
		if q != "" {
			haystack := strings.ToLower(strings.Join([]string{row.Summary, row.Description, row.Categories, row.Context, row.Goal, row.Folder}, " "))
			if !strings.Contains(haystack, q) {
				continue
			}
		}
		out = append(out, row)
	}
	if view == "priority" {
		sort.SliceStable(out, func(i, j int) bool { return out[i].Priority > out[j].Priority })
	}
	if view == "due" {
		sort.SliceStable(out, func(i, j int) bool { return out[i].DueInput < out[j].DueInput })
	}
	return out
}

func isDueSoon(due string, now time.Time) bool {
	t, err := time.Parse("2006-01-02", strings.TrimSpace(due))
	if err != nil {
		return false
	}
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	return !t.After(today.AddDate(0, 0, 3))
}
