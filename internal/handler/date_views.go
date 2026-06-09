package handler

import (
	"net/http"
	"time"

	"caldo/internal/db"
	"caldo/internal/view"
)

type dateViewDependencies struct {
	database *db.Database
	now      func() time.Time
}

func withDefaultNow(now func() time.Time) func() time.Time {
	if now == nil {
		return time.Now
	}
	return now
}

// Today renders tasks due today and overdue tasks.
func Today(deps dateViewDependencies) http.HandlerFunc {
	nowFn := withDefaultNow(deps.now)

	return func(w http.ResponseWriter, r *http.Request) {
		results, err := deps.database.ListTodayTasks(r.Context(), nowFn(), 200)
		if err != nil {
			renderPageError(w, r, "Heute", "Heute laden", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := view.BaseLayout("Heute", view.DateScopedTasksPage("Heute", "Keine fälligen oder überfälligen Aufgaben.", datedTaskRows(results))).Render(r.Context(), w); err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
	}
}

// Upcoming renders tasks in the configured upcoming window.
func Upcoming(deps dateViewDependencies) http.HandlerFunc {
	nowFn := withDefaultNow(deps.now)

	return func(w http.ResponseWriter, r *http.Request) {
		results, err := deps.database.ListUpcomingTasks(r.Context(), nowFn(), 200)
		if err != nil {
			renderPageError(w, r, "Demnächst", "Demnächst laden", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := view.BaseLayout("Demnächst", view.DateScopedTasksPage("Demnächst", "Keine demnächst fälligen Aufgaben.", datedTaskRows(results))).Render(r.Context(), w); err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
	}
}

// Overdue renders overdue tasks.
func Overdue(deps dateViewDependencies) http.HandlerFunc {
	nowFn := withDefaultNow(deps.now)

	return func(w http.ResponseWriter, r *http.Request) {
		results, err := deps.database.ListOverdueTasks(r.Context(), nowFn(), 200)
		if err != nil {
			renderPageError(w, r, "Überfällig", "Überfällig laden", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := view.BaseLayout("Überfällig", view.DateScopedTasksPage("Überfällig", "Keine überfälligen Aufgaben.", datedTaskRows(results))).Render(r.Context(), w); err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
	}
}

// Favorites renders favorite tasks.
func Favorites(deps dateViewDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		results, err := deps.database.ListFavoriteTasks(r.Context(), 200)
		if err != nil {
			renderPageError(w, r, "Favoriten", "Favoriten laden", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := view.BaseLayout("Favoriten", view.DateScopedTasksPage("Favoriten", "Keine favorisierten Aufgaben.", datedTaskRows(results))).Render(r.Context(), w); err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
	}
}

// NoDate renders tasks without due date.
func NoDate(deps dateViewDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		results, err := deps.database.ListNoDateTasks(r.Context(), 200)
		if err != nil {
			renderPageError(w, r, "Ohne Datum", "Ohne Datum laden", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := view.BaseLayout("Ohne Datum", view.DateScopedTasksPage("Ohne Datum", "Keine Aufgaben ohne Datum.", datedTaskRows(results))).Render(r.Context(), w); err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
	}
}

// Completed renders completed tasks if visibility is enabled.
func Completed(deps dateViewDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		results, err := deps.database.ListCompletedTasks(r.Context(), 200)
		if err != nil {
			renderPageError(w, r, "Erledigt", "Erledigte Aufgaben laden", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := view.BaseLayout("Erledigt", view.DateScopedTasksPage("Erledigte Aufgaben", "Erledigte Aufgaben sind ausgeblendet.", datedTaskRows(results))).Render(r.Context(), w); err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
	}
}

func datedTaskRows(rows []db.DatedTaskViewRow) []view.TaskRowView {
	tasks := make([]view.TaskRowView, 0, len(rows))
	for _, row := range rows {
		tasks = append(tasks, view.TaskRowView{
			ID:            row.ID,
			Title:         row.Title,
			Description:   row.Description,
			ProjectName:   row.ProjectName,
			LabelNames:    row.LabelNames,
			DueISODate:    row.DueISODate,
			Status:        row.Status,
			SyncStatus:    row.SyncStatus,
			Priority:      row.Priority,
			HasPriority:   row.HasPriority,
			ServerVersion: row.ServerVersion,
			IsSubtask:     row.IsSubtask,
		})
	}
	return tasks
}
