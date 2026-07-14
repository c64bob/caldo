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

type searchPageData struct {
	query      string
	groups     []view.TaskListGroupView
	display    view.TaskListDisplayView
	saveFilter view.SearchSaveFilterView
}

// Search renders global search results for active tasks.
func Search(deps searchDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data, err := loadSearchPageData(r.Context(), deps.database, r.URL.Query().Get("q"))
		if err != nil {
			renderPageError(w, r, "Suche", "Suche laden", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := view.BaseLayoutWithTopbarAction("Suche", view.TaskListDisplayControls(data.display), view.ConfigurableSearchPage(data.query, data.groups, data.saveFilter)).Render(r.Context(), w); err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
	}
}

// SearchResults renders the live-updating search result fragment.
func SearchResults(deps searchDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data, err := loadSearchPageData(r.Context(), deps.database, r.URL.Query().Get("q"))
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := view.ConfigurableSearchLiveResults(data.query, data.groups, data.saveFilter).Render(r.Context(), w); err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
	}
}

func loadSearchPageData(ctx context.Context, database *db.Database, rawQuery string) (searchPageData, error) {
	query := strings.TrimSpace(rawQuery)

	results, err := database.SearchActiveTasks(ctx, query, 50)
	if err != nil {
		return searchPageData{}, err
	}

	projectOptions, err := taskEditProjectOptions(ctx, database)
	if err != nil {
		return searchPageData{}, err
	}

	items := make([]view.TaskRowView, 0, len(results))
	todayISODate := time.Now().UTC().Format("2006-01-02")
	for _, result := range results {
		fields := model.ParseVTODOFields(result.RawVTODO)
		items = append(items, view.TaskRowView{
			ID:               result.ID,
			ProjectID:        result.ProjectID,
			Title:            result.Title,
			Description:      result.Description,
			ProjectName:      result.ProjectName,
			LabelNames:       result.LabelNames,
			DueISODate:       result.DueISODate,
			TodayISODate:     todayISODate,
			ParentID:         result.ParentID,
			ParentTitle:      result.ParentTitle,
			Status:           result.Status,
			SyncStatus:       result.SyncStatus,
			Priority:         result.Priority,
			HasPriority:      result.HasPriority,
			ServerVersion:    result.ServerVersion,
			IsSubtask:        result.IsSubtask,
			SubtaskCount:     result.SubtaskCount,
			OpenSubtaskCount: result.OpenSubtaskCount,
			ConflictID:       result.UnresolvedConflictID,
			RRule:            fields.RRule,
			Attachments:      fields.Attachments,
			ProjectOptions:   projectOptions,
			CreatedAt:        result.CreatedAt,
		})
	}

	display, groups, err := loadTaskListPresentation(ctx, database, taskListScope{Kind: model.TaskViewSearch, SearchQuery: query}, items, time.Now())
	if err != nil {
		return searchPageData{}, err
	}

	return searchPageData{
		query:      query,
		groups:     groups,
		display:    display,
		saveFilter: saveFilterForSearchQuery(query),
	}, nil
}

func saveFilterForSearchQuery(rawQuery string) view.SearchSaveFilterView {
	filterQuery := strings.TrimSpace(rawQuery)
	if filterQuery == "" {
		return view.SearchSaveFilterView{}
	}
	_, _, ok, err := db.EvaluateSavedFilter(filterQuery, 7)
	if err != nil || !ok {
		return view.SearchSaveFilterView{}
	}
	return view.SearchSaveFilterView{
		Enabled:    true,
		Query:      filterQuery,
		IsFavorite: true,
	}
}
