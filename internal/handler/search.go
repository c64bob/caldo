package handler

import (
	"net/http"
	"strings"

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
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		items := make([]view.SearchResultView, 0, len(results))
		for _, result := range results {
			items = append(items, view.SearchResultView{
				ID:          result.ID,
				Title:       result.Title,
				Description: result.Description,
				ProjectName: result.ProjectName,
				LabelNames:  result.LabelNames,
				Attachments: model.ParseVTODOFields(result.RawVTODO).Attachments,
			})
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := view.BaseLayout("Suche", view.SearchPage(query, items)).Render(r.Context(), w); err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
	}
}
