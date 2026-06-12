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

const quickAddProjectSuggestionLimit = 5

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
		projectOptions, projectOptionsErr := deps.database.ListProjectOptions(r.Context())
		if projectOptionsErr == nil {
			draft.ProjectOptions = quickAddProjectOptions(projectOptions)
		}
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
				if projectOptionsErr == nil {
					draft.ProjectSuggestions = quickAddProjectSuggestions(projectOptions, requestedProject)
				}
			}
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		component := view.QuickAddPreview(draft, text)
		if strings.TrimSpace(r.FormValue("surface")) == "overlay" {
			component = view.QuickAddOverlayPreview(draft, text)
		}
		if err := component.Render(ctx, w); err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
	}
}

func quickAddProjectSuggestions(projects []db.ProjectOption, requestedProject string) []parser.QuickAddProjectSuggestion {
	requested := normalizeQuickAddSuggestionText(requestedProject)
	if requested == "" {
		return nil
	}

	matches := make([]parser.QuickAddProjectSuggestion, 0, quickAddProjectSuggestionLimit)
	seen := make(map[string]struct{}, len(projects))
	appendSuggestion := func(project db.ProjectOption) {
		id := strings.TrimSpace(project.ID)
		name := strings.TrimSpace(project.DisplayName)
		if id == "" || name == "" || len(matches) >= quickAddProjectSuggestionLimit {
			return
		}
		if _, ok := seen[id]; ok {
			return
		}
		seen[id] = struct{}{}
		matches = append(matches, parser.QuickAddProjectSuggestion{ID: id, Name: name})
	}

	for _, project := range projects {
		name := normalizeQuickAddSuggestionText(project.DisplayName)
		if name == "" {
			continue
		}
		if strings.Contains(name, requested) || strings.Contains(requested, name) {
			appendSuggestion(project)
		}
	}
	for _, project := range projects {
		appendSuggestion(project)
	}

	return matches
}

func quickAddProjectOptions(projects []db.ProjectOption) []parser.QuickAddProjectSuggestion {
	options := make([]parser.QuickAddProjectSuggestion, 0, len(projects))
	seen := make(map[string]struct{}, len(projects))
	for _, project := range projects {
		id := strings.TrimSpace(project.ID)
		name := strings.TrimSpace(project.DisplayName)
		if id == "" || name == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		options = append(options, parser.QuickAddProjectSuggestion{ID: id, Name: name})
	}
	return options
}

func normalizeQuickAddSuggestionText(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}
