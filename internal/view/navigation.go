package view

import (
	"context"
	"net/url"
	"strconv"

	"github.com/a-h/templ"
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
	ProjectID    string
	Active       bool
	ActiveTitles []string
}

// NavigationOverviewItem contains one row on a navigation overview page.
type NavigationOverviewItem struct {
	ID              string
	Name            string
	Href            string
	Count           int
	HasCount        bool
	Active          bool
	Meta            string
	ServerVersion   int
	RenameSuccess   string
	RenameError     string
	RenameValue     string
	DeleteTaskCount int
	DeleteError     string
	DeleteValue     string
}

// ProjectFeedback contains visible project management status messages.
type ProjectFeedback struct {
	PageSuccess   string
	CreateSuccess string
	CreateError   string
	CreateValue   string
}

// WithNavigation stores app-shell navigation data in request context.
func WithNavigation(ctx context.Context, snapshot NavigationSnapshot) context.Context {
	return context.WithValue(ctx, navigationKey, snapshot)
}

// Navigation returns app-shell navigation data from request context or static fallback entries.
func Navigation(ctx context.Context) NavigationSnapshot {
	snapshot, ok := ctx.Value(navigationKey).(NavigationSnapshot)
	if !ok {
		return localizeNavigationSnapshot(StaticNavigation(), Text(ctx))
	}
	return localizeNavigationSnapshot(snapshot, Text(ctx))
}

// StaticNavigation returns route-only shell navigation without counters.
func StaticNavigation() NavigationSnapshot {
	return NavigationSnapshot{System: systemNavigationItems(0, 0, 0, 0, 0, 0, 0, false)}
}

// BuildNavigationSnapshot builds a view navigation snapshot from counted entries.
func BuildNavigationSnapshot(todayCount, upcomingCount, favoriteCount, overdueCount, noDateCount, completedCount, conflictCount int, projects, labels, savedFilters []NavigationOverviewItem) NavigationSnapshot {
	return NavigationSnapshot{
		System:       systemNavigationItems(todayCount, upcomingCount, favoriteCount, overdueCount, noDateCount, completedCount, conflictCount, true),
		Projects:     projectOverviewItemsToNavigationItems(projects),
		Labels:       overviewItemsToNavigationItems(labels),
		SavedFilters: overviewItemsToNavigationItems(savedFilters),
	}
}

// ProjectSearchHref returns the existing search route scoped to a project token.
func ProjectSearchHref(name string) string {
	return "/search?q=" + url.QueryEscape("#"+name)
}

// ProjectHref returns the canonical project task view route.
func ProjectHref(projectID string) string {
	return "/projects/" + url.PathEscape(projectID)
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

func localizeNavigationSnapshot(snapshot NavigationSnapshot, text Texts) NavigationSnapshot {
	snapshot.System = localizeSystemNavigationItems(snapshot.System, text)
	return snapshot
}

func localizeSystemNavigationItems(items []NavigationItem, text Texts) []NavigationItem {
	localized := make([]NavigationItem, len(items))
	copy(localized, items)
	for i := range localized {
		switch localized[i].Href {
		case "/today":
			localized[i].Label = text.Today
			localized[i].ActiveTitles = mergeActiveTitles(localized[i].ActiveTitles, text.Today)
		case "/upcoming":
			localized[i].Label = text.Upcoming
			localized[i].ActiveTitles = mergeActiveTitles(localized[i].ActiveTitles, text.Upcoming)
		case "/favorites":
			localized[i].Label = text.Favorites
			localized[i].ActiveTitles = mergeActiveTitles(localized[i].ActiveTitles, text.Favorites)
		case "/overdue":
			localized[i].Label = text.Overdue
			localized[i].ActiveTitles = mergeActiveTitles(localized[i].ActiveTitles, text.Overdue)
		case "/no-date":
			localized[i].Label = text.NoDate
			localized[i].ActiveTitles = mergeActiveTitles(localized[i].ActiveTitles, text.NoDate)
		case "/completed":
			localized[i].Label = text.Completed
			localized[i].ActiveTitles = mergeActiveTitles(localized[i].ActiveTitles, text.Completed)
		case "/search":
			localized[i].Label = text.Search
			localized[i].ActiveTitles = mergeActiveTitles(localized[i].ActiveTitles, text.Search)
		case "/conflicts":
			localized[i].Label = text.Conflicts
			localized[i].ActiveTitles = mergeActiveTitles(localized[i].ActiveTitles, text.Conflicts)
		case "/settings":
			localized[i].Label = text.Settings
			localized[i].ActiveTitles = mergeActiveTitles(localized[i].ActiveTitles, text.Settings)
		}
	}
	return localized
}

func mergeActiveTitles(existing []string, extra string) []string {
	if extra == "" {
		return existing
	}
	for _, item := range existing {
		if item == extra {
			return existing
		}
	}
	result := make([]string, 0, len(existing)+1)
	result = append(result, existing...)
	result = append(result, extra)
	return result
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

func projectOverviewItemsToNavigationItems(items []NavigationOverviewItem) []NavigationItem {
	result := make([]NavigationItem, 0, len(items))
	for _, item := range items {
		result = append(result, NavigationItem{
			Label:     item.Name,
			Href:      item.Href,
			Count:     item.Count,
			HasCount:  item.HasCount,
			ProjectID: item.ID,
			Active:    item.Active,
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

func navProjectDropAttributes(item NavigationItem) templ.Attributes {
	if item.ProjectID == "" {
		return nil
	}
	return templ.Attributes{
		"data-project-drop-target": "",
		"data-project-id":          item.ProjectID,
		"data-project-name":        item.Label,
	}
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

func projectCanDelete(item NavigationOverviewItem) bool {
	return item.ID != "" && item.ServerVersion > 0
}

func projectCanAcceptTaskDrop(item NavigationOverviewItem) bool {
	return projectCanRename(item)
}

func projectDropAttributes(item NavigationOverviewItem) templ.Attributes {
	if !projectCanAcceptTaskDrop(item) {
		return nil
	}
	return templ.Attributes{
		"data-project-drop-target": "",
		"data-project-id":          item.ID,
		"data-project-name":        item.Name,
	}
}

func projectDeletePath(item NavigationOverviewItem) string {
	return "/projects/" + url.PathEscape(item.ID)
}

func projectDeleteValue(item NavigationOverviewItem) string {
	return item.DeleteValue
}

func projectDeleteConfirmationText(item NavigationOverviewItem) string {
	return "Dieses Projekt wird auf dem CalDAV-Server gelöscht. " + projectDeleteLocalRemovalText(item.DeleteTaskCount) + " Zum Löschen " + item.Name + " eingeben."
}

func projectDeleteTaskCountText(count int) string {
	if count == 1 {
		return "1 Aufgabe"
	}
	return strconv.Itoa(count) + " Aufgaben"
}

func projectDeleteLocalRemovalText(count int) string {
	if count == 1 {
		return "1 Aufgabe wird lokal entfernt."
	}
	return strconv.Itoa(count) + " Aufgaben werden lokal entfernt."
}
