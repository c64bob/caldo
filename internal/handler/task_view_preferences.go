package handler

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"caldo/internal/db"
	"caldo/internal/model"
	"caldo/internal/view"
)

type taskListScope struct {
	Kind        string
	ID          string
	SearchQuery string
}

func loadTaskListPresentation(ctx context.Context, database *db.Database, scope taskListScope, tasks []view.TaskRowView, referenceDate time.Time) (view.TaskListDisplayView, []view.TaskListGroupView, error) {
	preference, err := database.LoadTaskViewPreference(ctx, scope.Kind, scope.ID)
	if err != nil {
		return view.TaskListDisplayView{}, nil, err
	}
	display := view.TaskListDisplayView{
		Preference:           preference,
		SearchQuery:          scope.SearchQuery,
		Language:             view.UILanguage(ctx),
		AllowProjectGrouping: scope.Kind != model.TaskViewProject,
		AllowDueDateGrouping: scope.Kind != model.TaskViewNoDate && scope.Kind != model.TaskViewOverdue,
	}
	if !taskListGroupAvailable(scope.Kind, preference.GroupBy) {
		display.Preference.GroupBy = model.TaskGroupNone
	}
	return display, view.BuildTaskListGroups(tasks, display, referenceDate), nil
}

// TaskViewPreferenceUpdate saves display-only preferences for one task-list view.
func TaskViewPreferenceUpdate(database *db.Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if database == nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form payload", http.StatusBadRequest)
			return
		}

		preference := model.TaskViewPreference{
			ViewKind:  strings.TrimSpace(r.FormValue("view_kind")),
			ViewID:    strings.TrimSpace(r.FormValue("view_id")),
			SortBy:    strings.TrimSpace(r.FormValue("sort_by")),
			SortOrder: strings.TrimSpace(r.FormValue("sort_order")),
			GroupBy:   strings.TrimSpace(r.FormValue("group_by")),
		}
		if err := model.ValidateTaskViewPreference(preference); err != nil || !taskListGroupAvailable(preference.ViewKind, preference.GroupBy) {
			http.Error(w, "invalid task view preference", http.StatusBadRequest)
			return
		}
		if preference.SortBy == model.TaskSortDefault {
			preference.SortOrder = model.TaskSortAscending
		}

		if strings.EqualFold(strings.TrimSpace(r.FormValue("action")), "reset") {
			if err := database.DeleteTaskViewPreference(r.Context(), preference.ViewKind, preference.ViewID); err != nil {
				http.Error(w, "failed to reset task view preference", http.StatusInternalServerError)
				return
			}
		} else if err := database.SaveTaskViewPreference(r.Context(), preference); err != nil {
			if errors.Is(err, model.ErrInvalidTaskViewPreference) {
				http.Error(w, "invalid task view preference", http.StatusBadRequest)
				return
			}
			http.Error(w, "failed to save task view preference", http.StatusInternalServerError)
			return
		}

		redirectURL := taskListScopeURL(taskListScope{Kind: preference.ViewKind, ID: preference.ViewID, SearchQuery: r.FormValue("search_query")})
		if strings.EqualFold(strings.TrimSpace(r.Header.Get("HX-Request")), "true") {
			w.Header().Set("HX-Redirect", redirectURL)
			w.WriteHeader(http.StatusNoContent)
			return
		}
		http.Redirect(w, r, redirectURL, http.StatusSeeOther)
	}
}

func taskListGroupAvailable(viewKind, groupBy string) bool {
	if groupBy == model.TaskGroupProject && viewKind == model.TaskViewProject {
		return false
	}
	if groupBy == model.TaskGroupDue && (viewKind == model.TaskViewNoDate || viewKind == model.TaskViewOverdue) {
		return false
	}
	return true
}

func taskListScopeURL(scope taskListScope) string {
	switch scope.Kind {
	case model.TaskViewToday:
		return "/today"
	case model.TaskViewUpcoming:
		return "/upcoming"
	case model.TaskViewOverdue:
		return "/overdue"
	case model.TaskViewFavorites:
		return "/favorites"
	case model.TaskViewNoDate:
		return "/no-date"
	case model.TaskViewCompleted:
		return "/completed"
	case model.TaskViewProject:
		return "/projects/" + url.PathEscape(scope.ID)
	case model.TaskViewLabel:
		return "/labels/" + url.PathEscape(scope.ID)
	case model.TaskViewFilter:
		return "/filters/" + url.PathEscape(scope.ID)
	case model.TaskViewSearch:
		values := url.Values{}
		if query := strings.TrimSpace(scope.SearchQuery); query != "" {
			values.Set("q", query)
		}
		if encoded := values.Encode(); encoded != "" {
			return "/search?" + encoded
		}
		return "/search"
	default:
		return "/today"
	}
}
