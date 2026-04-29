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
		if err := view.QuickAddPage(nil, "", "").Render(r.Context(), w); err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
	}
}

// QuickAddPreview renders the parsed quick-add preview.
func QuickAddPreview(deps quickAddDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		text := strings.TrimSpace(r.FormValue("text"))
		draft := parser.ParseQuickAdd(text)
		if draft.Title == "" {
			http.Error(w, "title is required", http.StatusBadRequest)
			return
		}
		project, err := deps.database.ResolveTaskProject(r.Context(), "")
		if draft.Project != "" {
			tokenProject, tokenErr := deps.database.LoadProjectByName(r.Context(), draft.Project)
			if tokenErr == nil {
				draft.ProjectID = tokenProject.ID
				draft.Project = tokenProject.DisplayName
			} else if tokenErr == sql.ErrNoRows {
				draft.ProjectNew = true
			}
		} else if err == nil {
			draft.ProjectID = project.ID
			draft.Project = project.DisplayName
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := view.QuickAddPreview(draft, text).Render(r.Context(), w); err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
	}
}
