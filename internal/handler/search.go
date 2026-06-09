package handler

import (
	"context"
	"net/http"
	"strings"
	"time"

	"caldo/internal/db"
	"caldo/internal/model"
	"caldo/internal/view"
)

type searchDependencies struct {
	database *db.Database
}

// Search renders global search results for active tasks.
func Search(deps searchDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := strings.TrimSpace(r.URL.Query().Get("q"))

		results, err := deps.database.SearchActiveTasks(r.Context(), query, 50)
		if err != nil {
			renderPageError(w, r, "Suche", "Suche laden", http.StatusInternalServerError)
			return
		}

		projectOptions, err := taskEditProjectOptions(r.Context(), deps.database)
		if err != nil {
			renderPageError(w, r, "Suche", "Suche laden", http.StatusInternalServerError)
			return
		}

		items := make([]view.TaskRowView, 0, len(results))
		todayISODate := time.Now().UTC().Format("2006-01-02")
		for _, result := range results {
			fields := model.ParseVTODOFields(result.RawVTODO)
			items = append(items, view.TaskRowView{
				ID:             result.ID,
				ProjectID:      result.ProjectID,
				Title:          result.Title,
				Description:    result.Description,
				ProjectName:    result.ProjectName,
				LabelNames:     result.LabelNames,
				DueISODate:     result.DueISODate,
				TodayISODate:   todayISODate,
				Status:         result.Status,
				SyncStatus:     result.SyncStatus,
				Priority:       result.Priority,
				HasPriority:    result.HasPriority,
				ServerVersion:  result.ServerVersion,
				RRule:          fields.RRule,
				Attachments:    fields.Attachments,
				ProjectOptions: projectOptions,
			})
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		create := inlineCreateForSearch(r.Context(), deps.database, query)
		if err := view.BaseLayout("Suche", view.SearchPage(query, items, create)).Render(r.Context(), w); err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
	}
}

func inlineCreateForSearch(ctx context.Context, database *db.Database, query string) view.InlineTaskCreateView {
	if database == nil {
		return view.InlineTaskCreateView{}
	}
	projectName, ok := projectOnlySearchQuery(query)
	if !ok {
		return view.InlineTaskCreateView{}
	}
	project, err := database.LoadProjectByName(ctx, projectName)
	if err != nil {
		return view.InlineTaskCreateView{}
	}
	return view.InlineTaskCreateView{
		Enabled:     true,
		ProjectID:   project.ID,
		ProjectName: project.DisplayName,
		Placeholder: "Aufgabe in " + project.DisplayName + " hinzufügen",
	}
}

func projectOnlySearchQuery(query string) (string, bool) {
	trimmed := strings.TrimSpace(query)
	if !strings.HasPrefix(trimmed, "#") {
		return "", false
	}
	projectName := strings.TrimSpace(strings.TrimPrefix(trimmed, "#"))
	if projectName == "" || strings.Contains(projectName, "#") || strings.Contains(projectName, "@") {
		return "", false
	}
	return projectName, true
}
