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
		create := inlineCreateForDate("Aufgabe für heute hinzufügen", reference)
		if err := view.BaseLayout("Heute", view.DateScopedTasksPage("Heute", "Keine fälligen oder überfälligen Aufgaben.", datedTaskRows(results, projectOptions, reference), create)).Render(r.Context(), w); err != nil {
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
		create := inlineCreateForDate("Aufgabe für demnächst hinzufügen", reference.AddDate(0, 0, 1))
		if err := view.BaseLayout("Demnächst", view.DateScopedTasksPage("Demnächst", "Keine demnächst fälligen Aufgaben.", datedTaskRows(results, projectOptions, reference), create)).Render(r.Context(), w); err != nil {
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
		if err := view.BaseLayout("Überfällig", view.DateScopedTasksPage("Überfällig", "Keine überfälligen Aufgaben.", datedTaskRows(results, projectOptions, reference), view.InlineTaskCreateView{})).Render(r.Context(), w); err != nil {
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
		if err := view.BaseLayout("Favoriten", view.DateScopedTasksPage("Favoriten", "Keine favorisierten Aufgaben.", datedTaskRows(results, projectOptions, reference), view.InlineTaskCreateView{})).Render(r.Context(), w); err != nil {
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
		create := view.InlineTaskCreateView{Enabled: true, Placeholder: "Aufgabe ohne Datum hinzufügen"}
		if err := view.BaseLayout("Ohne Datum", view.DateScopedTasksPage("Ohne Datum", "Keine Aufgaben ohne Datum.", datedTaskRows(results, projectOptions, reference), create)).Render(r.Context(), w); err != nil {
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
		if err := view.BaseLayout("Erledigt", view.DateScopedTasksPage("Erledigte Aufgaben", "Erledigte Aufgaben sind ausgeblendet.", datedTaskRows(results, projectOptions, reference), view.InlineTaskCreateView{})).Render(r.Context(), w); err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
	}
}

func inlineCreateForDate(placeholder string, dueDate time.Time) view.InlineTaskCreateView {
	return view.InlineTaskCreateView{
		Enabled:     true,
		DueDate:     dueDate.UTC().Format("2006-01-02"),
		Placeholder: placeholder,
	}
}

func datedTaskRows(rows []db.DatedTaskViewRow, projectOptions []view.TaskProjectOption, referenceDate time.Time) []view.TaskRowView {
	tasks := make([]view.TaskRowView, 0, len(rows))
	todayISODate := referenceDate.UTC().Format("2006-01-02")
	for _, row := range rows {
		fields := model.ParseVTODOFields(row.RawVTODO)
		tasks = append(tasks, view.TaskRowView{
			ID:             row.ID,
			ProjectID:      row.ProjectID,
			Title:          row.Title,
			Description:    row.Description,
			ProjectName:    row.ProjectName,
			LabelNames:     row.LabelNames,
			DueISODate:     row.DueISODate,
			TodayISODate:   todayISODate,
			ParentID:       row.ParentID,
			ParentTitle:    row.ParentTitle,
			Status:         row.Status,
			SyncStatus:     row.SyncStatus,
			Priority:       row.Priority,
			HasPriority:    row.HasPriority,
			ServerVersion:  row.ServerVersion,
			IsSubtask:      row.IsSubtask,
			SubtaskCount:   row.SubtaskCount,
			RRule:          fields.RRule,
			Attachments:    fields.Attachments,
			ProjectOptions: projectOptions,
		})
	}
	return tasks
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
