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
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := view.BaseLayout("Heute", view.DateScopedTasksPage("Heute", "Keine fälligen oder überfälligen Aufgaben.", results)).Render(r.Context(), w); err != nil {
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
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := view.BaseLayout("Demnächst", view.DateScopedTasksPage("Demnächst", "Keine demnächst fälligen Aufgaben.", results)).Render(r.Context(), w); err != nil {
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
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := view.BaseLayout("Überfällig", view.DateScopedTasksPage("Überfällig", "Keine überfälligen Aufgaben.", results)).Render(r.Context(), w); err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
	}
}
