package handler

import (
	"net/http"
	"strings"

	"caldo/internal/db"
	"caldo/internal/view"
)

func placeholderPage(title, heading string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := view.BaseLayout(title, view.PlaceholderPage(heading)).Render(r.Context(), w); err != nil {
			http.Error(w, "render page", http.StatusInternalServerError)
		}
	}
}

// ProjectsPage renders the projects navigation page.
func ProjectsPage() http.HandlerFunc { return placeholderPage("Projekte", "Projekte") }

// LabelsPage renders the labels navigation page.
func LabelsPage() http.HandlerFunc { return placeholderPage("Labels", "Labels") }

// FiltersPage renders the filters navigation page.
func FiltersPage() http.HandlerFunc { return placeholderPage("Filter", "Filter") }

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
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		if err := view.BaseLayout("Einstellungen", view.SettingsPageContent(settings, proxyUserHeader, httpsConfigured)).Render(r.Context(), w); err != nil {
			http.Error(w, "render page", http.StatusInternalServerError)
		}
	}
}
