package handler

import (
	"net/http"

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

// SettingsPage renders the settings navigation page.
func SettingsPage() http.HandlerFunc { return placeholderPage("Einstellungen", "Einstellungen") }
