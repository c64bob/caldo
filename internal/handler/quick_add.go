package handler

import (
	"database/sql"
	"net/http"
	"strings"

	"caldo/internal/db"
	"caldo/internal/parser"
	"caldo/internal/view"
)

type quickAddDependencies struct {
	database *db.Database
}

// QuickAddPage renders quick-add with optional preview.
func QuickAddPage(deps quickAddDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		ctx := withUIPreferences(r.Context(), deps.database)
		if err := view.BaseLayout("Quick Add", view.QuickAddPage(nil, "", "")).Render(ctx, w); err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
	}
}

// QuickAddPreview renders the parsed quick-add preview.
func QuickAddPreview(deps quickAddDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		text := strings.TrimSpace(r.FormValue("text"))
		language := "de"
		ctx := r.Context()
		if preferences, err := deps.database.LoadUIPreferences(r.Context()); err == nil {
			language = preferences.UILanguage
			ctx = view.WithUIPreferences(ctx, preferences.UILanguage, preferences.DarkMode)
		}
		draft := parser.ParseQuickAddWithLanguage(text, language)
		requestedProject := draft.Project
		if draft.Title == "" {
			w.WriteHeader(http.StatusBadRequest)
			if err := view.ErrorState("Vorschau erstellen", "validierungsfehler", false).Render(ctx, w); err != nil {
				http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			}
			return
		}
		project, err := deps.database.ResolveTaskProject(r.Context(), "")
		if err == nil {
			draft.ProjectID = project.ID
			draft.Project = project.DisplayName
		}
		if requestedProject != "" {
			tokenProject, tokenErr := deps.database.LoadProjectByName(r.Context(), requestedProject)
			if tokenErr == nil {
				draft.ProjectID = tokenProject.ID
				draft.Project = tokenProject.DisplayName
			} else if tokenErr == sql.ErrNoRows {
				draft.Project = requestedProject
				draft.ProjectNew = true
			}
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := view.QuickAddPreview(draft, text).Render(ctx, w); err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
	}
}
