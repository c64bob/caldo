package view

import (
	"context"
	"net/url"
	"strconv"
)

// NavigationSnapshot contains app-shell navigation entries.
type NavigationSnapshot struct {
	System       []NavigationItem
	Projects     []NavigationItem
	Labels       []NavigationItem
	SavedFilters []NavigationItem
}

// NavigationItem contains one app-shell navigation entry.
type NavigationItem struct {
	Label        string
	Href         string
	Count        int
	HasCount     bool
	ActiveTitles []string
}

// NavigationOverviewItem contains one row on a navigation overview page.
type NavigationOverviewItem struct {
	ID            string
	Name          string
	Href          string
	Count         int
	HasCount      bool
	Meta          string
	ServerVersion int
	RenameError   string
	RenameValue   string
}

// WithNavigation stores app-shell navigation data in request context.
func WithNavigation(ctx context.Context, snapshot NavigationSnapshot) context.Context {
	return context.WithValue(ctx, navigationKey, snapshot)
}

// Navigation returns app-shell navigation data from request context or static fallback entries.
func Navigation(ctx context.Context) NavigationSnapshot {
	snapshot, ok := ctx.Value(navigationKey).(NavigationSnapshot)
	if !ok {
		return StaticNavigation()
	}
	return snapshot
}

// StaticNavigation returns route-only shell navigation without counters.
func StaticNavigation() NavigationSnapshot {
	return NavigationSnapshot{System: systemNavigationItems(0, 0, 0, 0, 0, 0, 0, false)}
}

// BuildNavigationSnapshot builds a view navigation snapshot from counted entries.
func BuildNavigationSnapshot(todayCount, upcomingCount, favoriteCount, overdueCount, noDateCount, completedCount, conflictCount int, projects, labels, savedFilters []NavigationOverviewItem) NavigationSnapshot {
	return NavigationSnapshot{
		System:       systemNavigationItems(todayCount, upcomingCount, favoriteCount, overdueCount, noDateCount, completedCount, conflictCount, true),
		Projects:     overviewItemsToNavigationItems(projects),
		Labels:       overviewItemsToNavigationItems(labels),
		SavedFilters: overviewItemsToNavigationItems(savedFilters),
	}
}

// ProjectSearchHref returns the existing search route scoped to a project token.
func ProjectSearchHref(name string) string {
	return "/search?q=" + url.QueryEscape("#"+name)
}

// LabelSearchHref returns the existing search route scoped to a label token.
func LabelSearchHref(name string) string {
	return "/search?q=" + url.QueryEscape("@"+name)
}

func systemNavigationItems(todayCount, upcomingCount, favoriteCount, overdueCount, noDateCount, completedCount, conflictCount int, counted bool) []NavigationItem {
	return []NavigationItem{
		{Label: "Heute", Href: "/today", Count: todayCount, HasCount: counted, ActiveTitles: []string{"Heute"}},
		{Label: "Demnächst", Href: "/upcoming", Count: upcomingCount, HasCount: counted, ActiveTitles: []string{"Demnächst"}},
		{Label: "Favoriten", Href: "/favorites", Count: favoriteCount, HasCount: counted, ActiveTitles: []string{"Favoriten"}},
		{Label: "Überfällig", Href: "/overdue", Count: overdueCount, HasCount: counted, ActiveTitles: []string{"Überfällig"}},
		{Label: "Ohne Datum", Href: "/no-date", Count: noDateCount, HasCount: counted, ActiveTitles: []string{"Ohne Datum"}},
		{Label: "Abgeschlossen", Href: "/completed", Count: completedCount, HasCount: counted, ActiveTitles: []string{"Abgeschlossen", "Erledigt", "Erledigte Aufgaben"}},
		{Label: "Suche", Href: "/search", ActiveTitles: []string{"Suche"}},
		{Label: "Konflikte", Href: "/conflicts", Count: conflictCount, HasCount: counted, ActiveTitles: []string{"Konflikte", "Konfliktdetail"}},
		{Label: "Einstellungen", Href: "/settings", ActiveTitles: []string{"Einstellungen"}},
	}
}

func overviewItemsToNavigationItems(items []NavigationOverviewItem) []NavigationItem {
	result := make([]NavigationItem, 0, len(items))
	for _, item := range items {
		result = append(result, NavigationItem{
			Label:    item.Name,
			Href:     item.Href,
			Count:    item.Count,
			HasCount: item.HasCount,
		})
	}
	return result
}

func navCountText(count int) string {
	if count > 99 {
		return "99+"
	}
	return strconv.Itoa(count)
}

func projectCanRename(item NavigationOverviewItem) bool {
	return item.ID != "" && item.ServerVersion > 0
}

func projectRenamePath(item NavigationOverviewItem) string {
	return "/projects/" + url.PathEscape(item.ID)
}

func projectExpectedVersion(item NavigationOverviewItem) string {
	return strconv.Itoa(item.ServerVersion)
}

func projectRenameValue(item NavigationOverviewItem) string {
	if item.RenameValue != "" {
		return item.RenameValue
	}
	return item.Name
}
