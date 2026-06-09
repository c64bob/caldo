package handler

import (
	"net/http"
	"strings"
	"time"

	"caldo/internal/db"
	"caldo/internal/view"
)

func navigationMiddleware(database *db.Database, setupState *SetupState) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !shouldLoadNavigation(database, setupState, r) {
				next.ServeHTTP(w, r)
				return
			}

			snapshot, err := database.LoadNavigationSnapshot(r.Context(), time.Now())
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			ctx := view.WithNavigation(r.Context(), navigationSnapshotView(snapshot))
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func shouldLoadNavigation(database *db.Database, setupState *SetupState, r *http.Request) bool {
	if database == nil || setupState == nil || !setupState.IsComplete() {
		return false
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		return false
	}
	path := r.URL.Path
	if path == "/health" || path == "/events" || strings.HasPrefix(path, "/static/") || strings.HasPrefix(path, "/api/") || strings.HasPrefix(path, "/setup/") || path == "/setup" {
		return false
	}
	return true
}

func navigationSnapshotView(snapshot db.NavigationSnapshot) view.NavigationSnapshot {
	projects := navigationProjectsView(snapshot.Projects)
	labels := navigationLabelsView(snapshot.Labels)
	filters := navigationFiltersView(snapshot.SavedFilters)
	return view.BuildNavigationSnapshot(
		snapshot.TodayCount,
		snapshot.UpcomingCount,
		snapshot.FavoriteCount,
		snapshot.OverdueCount,
		snapshot.NoDateCount,
		snapshot.CompletedCount,
		snapshot.ConflictCount,
		projects,
		labels,
		filters,
	)
}

func navigationProjectsView(items []db.NavigationListItem) []view.NavigationOverviewItem {
	result := make([]view.NavigationOverviewItem, 0, len(items))
	for _, item := range items {
		result = append(result, view.NavigationOverviewItem{
			Name:     item.Name,
			Href:     view.ProjectSearchHref(item.Name),
			Count:    item.OpenTaskCount,
			HasCount: true,
			Meta:     "Offene Aufgaben",
		})
	}
	return result
}

func navigationLabelsView(items []db.NavigationListItem) []view.NavigationOverviewItem {
	result := make([]view.NavigationOverviewItem, 0, len(items))
	for _, item := range items {
		result = append(result, view.NavigationOverviewItem{
			Name:     item.Name,
			Href:     view.LabelSearchHref(item.Name),
			Count:    item.OpenTaskCount,
			HasCount: true,
			Meta:     "Offene Aufgaben",
		})
	}
	return result
}

func navigationFiltersView(items []db.NavigationListItem) []view.NavigationOverviewItem {
	result := make([]view.NavigationOverviewItem, 0, len(items))
	for _, item := range items {
		result = append(result, view.NavigationOverviewItem{
			Name: item.Name,
			Href: "/filters#" + item.ID,
			Meta: "Gespeicherter Filter",
		})
	}
	return result
}
