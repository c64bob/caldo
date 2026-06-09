package handler

import (
	"net/http"
	"strings"
	"time"

	"caldo/internal/db"
	"caldo/internal/view"
)

// ProjectsPage renders the projects navigation page.
func ProjectsPage(database *db.Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if database == nil {
			renderPageError(w, r, "Projekte", "Projekte laden", http.StatusInternalServerError)
			return
		}

		snapshot, err := database.LoadNavigationSnapshot(r.Context(), time.Now())
		if err != nil {
			renderPageError(w, r, "Projekte", "Projekte laden", http.StatusInternalServerError)
			return
		}

		if err := view.BaseLayout("Projekte", view.NavigationOverviewPage("Projekte", "Keine Projekte", navigationProjectsView(snapshot.Projects))).Render(r.Context(), w); err != nil {
			http.Error(w, "render page", http.StatusInternalServerError)
		}
	}
}

// LabelsPage renders the labels navigation page.
func LabelsPage(database *db.Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if database == nil {
			renderPageError(w, r, "Labels", "Labels laden", http.StatusInternalServerError)
			return
		}

		snapshot, err := database.LoadNavigationSnapshot(r.Context(), time.Now())
		if err != nil {
			renderPageError(w, r, "Labels", "Labels laden", http.StatusInternalServerError)
			return
		}

		if err := view.BaseLayout("Labels", view.NavigationOverviewPage("Labels", "Keine Labels", navigationLabelsView(snapshot.Labels))).Render(r.Context(), w); err != nil {
			http.Error(w, "render page", http.StatusInternalServerError)
		}
	}
}

// FiltersPage renders the filters navigation page.
func FiltersPage(database *db.Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if database == nil {
			renderPageError(w, r, "Filter", "Filter laden", http.StatusInternalServerError)
			return
		}

		snapshot, err := database.LoadNavigationSnapshot(r.Context(), time.Now())
		if err != nil {
			renderPageError(w, r, "Filter", "Filter laden", http.StatusInternalServerError)
			return
		}

		if err := view.BaseLayout("Filter", view.NavigationOverviewPage("Filter", "Keine gespeicherten Filter", navigationFiltersView(snapshot.SavedFilters))).Render(r.Context(), w); err != nil {
			http.Error(w, "render page", http.StatusInternalServerError)
		}
	}
}

// SettingsPage renders the settings page for normal operation.
func SettingsPage(database *db.Database, proxyUserHeader string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		httpsConfigured := r.TLS != nil || strings.EqualFold(strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")), "https")
		if database == nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		settings, err := database.LoadAppSettings(r.Context())
		if err != nil {
			renderPageError(w, r, "Einstellungen", "Einstellungen laden", http.StatusInternalServerError)
			return
		}
		if err := view.BaseLayout("Einstellungen", view.SettingsPageContent(settings, proxyUserHeader, httpsConfigured)).Render(r.Context(), w); err != nil {
			http.Error(w, "render page", http.StatusInternalServerError)
		}
	}
}

func renderPageError(w http.ResponseWriter, r *http.Request, title string, action string, status int) {
	w.WriteHeader(status)
	if err := view.BaseLayout(title, view.ErrorState(action, "serverfehler", false)).Render(r.Context(), w); err != nil {
		http.Error(w, http.StatusText(status), status)
	}
}
