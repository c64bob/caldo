package handler

import (
	"context"
	"net/http"
	"time"

	"caldo/internal/db"
	"caldo/internal/model"
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
		reference := nowFn()
		results, err := deps.database.ListTodayTasks(r.Context(), reference, 200)
		if err != nil {
			renderPageError(w, r, "Heute", "Heute laden", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		projectOptions, err := taskEditProjectOptions(r.Context(), deps.database)
		if err != nil {
			renderPageError(w, r, "Heute", "Heute laden", http.StatusInternalServerError)
			return
		}
		display, groups, err := datedTaskListPresentation(r.Context(), deps.database, taskListScope{Kind: model.TaskViewToday}, results, projectOptions, reference)
		if err != nil {
			renderPageError(w, r, "Heute", "Heute laden", http.StatusInternalServerError)
			return
		}
		if err := view.BaseLayout("Heute", view.ConfigurableTaskListPage("Heute", "Keine fälligen oder überfälligen Aufgaben.", groups, display)).Render(r.Context(), w); err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
	}
}

// Upcoming renders tasks in the configured upcoming window.
func Upcoming(deps dateViewDependencies) http.HandlerFunc {
	nowFn := withDefaultNow(deps.now)

	return func(w http.ResponseWriter, r *http.Request) {
		reference := nowFn()
		results, err := deps.database.ListUpcomingTasks(r.Context(), reference, 200)
		if err != nil {
			renderPageError(w, r, "Demnächst", "Demnächst laden", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		projectOptions, err := taskEditProjectOptions(r.Context(), deps.database)
		if err != nil {
			renderPageError(w, r, "Demnächst", "Demnächst laden", http.StatusInternalServerError)
			return
		}
		display, groups, err := datedTaskListPresentation(r.Context(), deps.database, taskListScope{Kind: model.TaskViewUpcoming}, results, projectOptions, reference)
		if err != nil {
			renderPageError(w, r, "Demnächst", "Demnächst laden", http.StatusInternalServerError)
			return
		}
		if err := view.BaseLayout("Demnächst", view.ConfigurableTaskListPage("Demnächst", "Keine demnächst fälligen Aufgaben.", groups, display)).Render(r.Context(), w); err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
	}
}

// Overdue renders overdue tasks.
func Overdue(deps dateViewDependencies) http.HandlerFunc {
	nowFn := withDefaultNow(deps.now)

	return func(w http.ResponseWriter, r *http.Request) {
		reference := nowFn()
		results, err := deps.database.ListOverdueTasks(r.Context(), reference, 200)
		if err != nil {
			renderPageError(w, r, "Überfällig", "Überfällig laden", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		projectOptions, err := taskEditProjectOptions(r.Context(), deps.database)
		if err != nil {
			renderPageError(w, r, "Überfällig", "Überfällig laden", http.StatusInternalServerError)
			return
		}
		display, groups, err := datedTaskListPresentation(r.Context(), deps.database, taskListScope{Kind: model.TaskViewOverdue}, results, projectOptions, reference)
		if err != nil {
			renderPageError(w, r, "Überfällig", "Überfällig laden", http.StatusInternalServerError)
			return
		}
		if err := view.BaseLayout("Überfällig", view.ConfigurableTaskListPage("Überfällig", "Keine überfälligen Aufgaben.", groups, display)).Render(r.Context(), w); err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
	}
}

// Favorites renders favorite tasks.
func Favorites(deps dateViewDependencies) http.HandlerFunc {
	nowFn := withDefaultNow(deps.now)

	return func(w http.ResponseWriter, r *http.Request) {
		reference := nowFn()
		results, err := deps.database.ListFavoriteTasks(r.Context(), 200)
		if err != nil {
			renderPageError(w, r, "Favoriten", "Favoriten laden", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		projectOptions, err := taskEditProjectOptions(r.Context(), deps.database)
		if err != nil {
			renderPageError(w, r, "Favoriten", "Favoriten laden", http.StatusInternalServerError)
			return
		}
		display, groups, err := datedTaskListPresentation(r.Context(), deps.database, taskListScope{Kind: model.TaskViewFavorites}, results, projectOptions, reference)
		if err != nil {
			renderPageError(w, r, "Favoriten", "Favoriten laden", http.StatusInternalServerError)
			return
		}
		if err := view.BaseLayout("Favoriten", view.ConfigurableTaskListPage("Favoriten", "Keine favorisierten Aufgaben.", groups, display)).Render(r.Context(), w); err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
	}
}

// NoDate renders tasks without due date.
func NoDate(deps dateViewDependencies) http.HandlerFunc {
	nowFn := withDefaultNow(deps.now)

	return func(w http.ResponseWriter, r *http.Request) {
		reference := nowFn()
		results, err := deps.database.ListNoDateTasks(r.Context(), 200)
		if err != nil {
			renderPageError(w, r, "Ohne Datum", "Ohne Datum laden", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		projectOptions, err := taskEditProjectOptions(r.Context(), deps.database)
		if err != nil {
			renderPageError(w, r, "Ohne Datum", "Ohne Datum laden", http.StatusInternalServerError)
			return
		}
		display, groups, err := datedTaskListPresentation(r.Context(), deps.database, taskListScope{Kind: model.TaskViewNoDate}, results, projectOptions, reference)
		if err != nil {
			renderPageError(w, r, "Ohne Datum", "Ohne Datum laden", http.StatusInternalServerError)
			return
		}
		if err := view.BaseLayout("Ohne Datum", view.ConfigurableTaskListPage("Ohne Datum", "Keine Aufgaben ohne Datum.", groups, display)).Render(r.Context(), w); err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
	}
}

// Completed renders completed tasks if visibility is enabled.
func Completed(deps dateViewDependencies) http.HandlerFunc {
	nowFn := withDefaultNow(deps.now)

	return func(w http.ResponseWriter, r *http.Request) {
		reference := nowFn()
		results, err := deps.database.ListCompletedTasks(r.Context(), 200)
		if err != nil {
			renderPageError(w, r, "Erledigt", "Erledigte Aufgaben laden", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		projectOptions, err := taskEditProjectOptions(r.Context(), deps.database)
		if err != nil {
			renderPageError(w, r, "Erledigt", "Erledigte Aufgaben laden", http.StatusInternalServerError)
			return
		}
		display, groups, err := datedTaskListPresentation(r.Context(), deps.database, taskListScope{Kind: model.TaskViewCompleted}, results, projectOptions, reference)
		if err != nil {
			renderPageError(w, r, "Erledigt", "Erledigte Aufgaben laden", http.StatusInternalServerError)
			return
		}
		if err := view.BaseLayout("Erledigt", view.ConfigurableTaskListPage("Erledigte Aufgaben", "Erledigte Aufgaben sind ausgeblendet.", groups, display)).Render(r.Context(), w); err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
	}
}

func datedTaskRows(rows []db.DatedTaskViewRow, projectOptions []view.TaskProjectOption, referenceDate time.Time) []view.TaskRowView {
	tasks := make([]view.TaskRowView, 0, len(rows))
	todayISODate := referenceDate.UTC().Format("2006-01-02")
	for _, row := range rows {
		tasks = append(tasks, taskRowFromDatedRow(row, projectOptions, todayISODate))
	}
	return tasks
}

func datedTaskListPresentation(ctx context.Context, database *db.Database, scope taskListScope, rows []db.DatedTaskViewRow, projectOptions []view.TaskProjectOption, referenceDate time.Time) (view.TaskListDisplayView, []view.TaskListGroupView, error) {
	return loadTaskListPresentation(ctx, database, scope, datedTaskRows(rows, projectOptions, referenceDate), referenceDate)
}

func taskEditProjectOptions(ctx context.Context, database *db.Database) ([]view.TaskProjectOption, error) {
	projects, err := database.ListProjectOptions(ctx)
	if err != nil {
		return nil, err
	}
	options := make([]view.TaskProjectOption, 0, len(projects))
	for _, project := range projects {
		options = append(options, view.TaskProjectOption{ID: project.ID, Name: project.DisplayName})
	}
	return options, nil
}
