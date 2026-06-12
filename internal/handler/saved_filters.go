package handler

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"caldo/internal/db"
	"caldo/internal/view"
	"github.com/go-chi/chi/v5"
)

type savedFilterDependencies struct {
	database *db.Database
	now      func() time.Time
}

type savedFiltersPageState struct {
	CreateError       string
	CreateName        string
	CreateQuery       string
	CreateFavorite    bool
	EditFilterID      string
	EditError         string
	EditName          string
	EditQuery         string
	EditFavorite      bool
	EditFavoriteValid bool
	DeleteFilterID    string
	DeleteError       string
}

// SavedFiltersPage renders saved-filter management.
func SavedFiltersPage(deps savedFilterDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		renderSavedFiltersPage(w, r, deps.database, savedFiltersPageState{}, http.StatusOK)
	}
}

// SavedFilterTasks renders the task list for one saved filter.
func SavedFilterTasks(deps savedFilterDependencies) http.HandlerFunc {
	nowFn := withDefaultNow(deps.now)

	return func(w http.ResponseWriter, r *http.Request) {
		filterID := strings.TrimSpace(chi.URLParam(r, "filterID"))
		if filterID == "" {
			http.Error(w, "filter id is required", http.StatusBadRequest)
			return
		}

		reference := nowFn()
		filter, results, _, err := deps.database.ListSavedFilterTasks(r.Context(), filterID, reference, 200)
		if err != nil {
			if errors.Is(err, db.ErrSavedFilterNotFound) {
				http.Error(w, "filter not found", http.StatusNotFound)
				return
			}
			renderPageError(w, r, "Filter", "Filter laden", http.StatusInternalServerError)
			return
		}

		projectOptions, err := taskEditProjectOptions(r.Context(), deps.database)
		if err != nil {
			renderPageError(w, r, filter.Name, "Filter laden", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := view.BaseLayout(filter.Name, view.DateScopedTasksPage(filter.Name, "Keine Aufgaben für diesen Filter.", datedTaskRows(results, projectOptions, reference), view.InlineTaskCreateView{})).Render(r.Context(), w); err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
	}
}

// SavedFilterCreate creates a local saved filter.
func SavedFilterCreate(deps savedFilterDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			renderSavedFiltersPage(w, r, deps.database, savedFiltersPageState{CreateError: "ungültige eingabe"}, http.StatusOK)
			return
		}

		name := strings.TrimSpace(r.FormValue("name"))
		filterQuery := strings.TrimSpace(r.FormValue("query"))
		favorite := formBool(r.FormValue("favorite"))
		if name == "" {
			renderSavedFiltersPage(w, r, deps.database, savedFiltersPageState{CreateError: "filtername ist erforderlich", CreateName: r.FormValue("name"), CreateQuery: r.FormValue("query"), CreateFavorite: favorite}, http.StatusOK)
			return
		}
		if filterQuery == "" {
			renderSavedFiltersPage(w, r, deps.database, savedFiltersPageState{CreateError: "filterquery ist erforderlich", CreateName: name, CreateQuery: r.FormValue("query"), CreateFavorite: favorite}, http.StatusOK)
			return
		}
		if !savedFilterQueryValid(filterQuery) {
			renderSavedFiltersPage(w, r, deps.database, savedFiltersPageState{CreateError: "filterquery ist ungültig", CreateName: name, CreateQuery: filterQuery, CreateFavorite: favorite}, http.StatusOK)
			return
		}

		if _, err := deps.database.CreateSavedFilter(r.Context(), name, filterQuery, favorite); err != nil {
			renderSavedFiltersPage(w, r, deps.database, savedFiltersPageState{CreateError: "filter konnte nicht gespeichert werden", CreateName: name, CreateQuery: filterQuery, CreateFavorite: favorite}, http.StatusOK)
			return
		}

		renderSavedFiltersPage(w, r, deps.database, savedFiltersPageState{}, http.StatusCreated)
	}
}

// SavedFilterUpdate updates a local saved filter using optimistic locking.
func SavedFilterUpdate(deps savedFilterDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		filterID := strings.TrimSpace(chi.URLParam(r, "filterID"))
		if filterID == "" {
			http.Error(w, "filter id is required", http.StatusBadRequest)
			return
		}
		if err := r.ParseForm(); err != nil {
			renderSavedFiltersPage(w, r, deps.database, editFilterPageState(filterID, "ungültige eingabe", "", "", false, false), http.StatusOK)
			return
		}

		name := strings.TrimSpace(r.FormValue("name"))
		filterQuery := strings.TrimSpace(r.FormValue("query"))
		favorite := formBool(r.FormValue("favorite"))
		expectedVersion, err := strconv.Atoi(strings.TrimSpace(r.FormValue("expected_version")))
		if err != nil {
			renderSavedFiltersPage(w, r, deps.database, editFilterPageState(filterID, "filterversion fehlt", name, filterQuery, favorite, true), http.StatusOK)
			return
		}
		if name == "" {
			renderSavedFiltersPage(w, r, deps.database, editFilterPageState(filterID, "filtername ist erforderlich", r.FormValue("name"), filterQuery, favorite, true), http.StatusOK)
			return
		}
		if filterQuery == "" {
			renderSavedFiltersPage(w, r, deps.database, editFilterPageState(filterID, "filterquery ist erforderlich", name, r.FormValue("query"), favorite, true), http.StatusOK)
			return
		}
		if !savedFilterQueryValid(filterQuery) {
			renderSavedFiltersPage(w, r, deps.database, editFilterPageState(filterID, "filterquery ist ungültig", name, filterQuery, favorite, true), http.StatusOK)
			return
		}

		if _, err := deps.database.UpdateSavedFilter(r.Context(), filterID, name, filterQuery, favorite, expectedVersion); err != nil {
			switch {
			case errors.Is(err, db.ErrSavedFilterNotFound):
				http.Error(w, "filter not found", http.StatusNotFound)
			case errors.Is(err, db.ErrSavedFilterVersionConflict):
				renderSavedFiltersPage(w, r, deps.database, editFilterPageState(filterID, "filter wurde zwischenzeitlich geändert", name, filterQuery, favorite, true), http.StatusOK)
			default:
				renderSavedFiltersPage(w, r, deps.database, editFilterPageState(filterID, "filter konnte nicht gespeichert werden", name, filterQuery, favorite, true), http.StatusOK)
			}
			return
		}

		renderSavedFiltersPage(w, r, deps.database, savedFiltersPageState{}, http.StatusOK)
	}
}

// SavedFilterDelete deletes a local saved filter using optimistic locking.
func SavedFilterDelete(deps savedFilterDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		filterID := strings.TrimSpace(chi.URLParam(r, "filterID"))
		if filterID == "" {
			http.Error(w, "filter id is required", http.StatusBadRequest)
			return
		}

		formValues, ok := parseDeleteFormValues(w, r, deps.database, filterID)
		if !ok {
			return
		}
		expectedVersion, err := strconv.Atoi(strings.TrimSpace(formValues.Get("expected_version")))
		if err != nil {
			renderSavedFiltersPage(w, r, deps.database, deleteFilterPageState(filterID, "filterversion fehlt"), http.StatusOK)
			return
		}

		if err := deps.database.DeleteSavedFilter(r.Context(), filterID, expectedVersion); err != nil {
			switch {
			case errors.Is(err, db.ErrSavedFilterNotFound):
				http.Error(w, "filter not found", http.StatusNotFound)
			case errors.Is(err, db.ErrSavedFilterVersionConflict):
				renderSavedFiltersPage(w, r, deps.database, deleteFilterPageState(filterID, "filter wurde zwischenzeitlich geändert"), http.StatusOK)
			default:
				renderSavedFiltersPage(w, r, deps.database, deleteFilterPageState(filterID, "filter konnte nicht gelöscht werden"), http.StatusOK)
			}
			return
		}

		renderSavedFiltersPage(w, r, deps.database, savedFiltersPageState{}, http.StatusOK)
	}
}

func savedFilterQueryValid(filterQuery string) bool {
	_, _, ok, err := db.EvaluateSavedFilter(filterQuery, 7)
	return err == nil && ok
}

func renderSavedFiltersPage(w http.ResponseWriter, r *http.Request, database *db.Database, pageState savedFiltersPageState, status int) {
	if database == nil {
		renderPageError(w, r, "Filter", "Filter laden", http.StatusInternalServerError)
		return
	}
	filters, err := database.ListSavedFilters(r.Context())
	if err != nil {
		renderPageError(w, r, "Filter", "Filter laden", http.StatusInternalServerError)
		return
	}
	snapshot, err := database.LoadNavigationSnapshot(r.Context(), time.Now())
	if err != nil {
		renderPageError(w, r, "Filter", "Filter laden", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if status != http.StatusOK {
		w.WriteHeader(status)
	}
	ctx := view.WithNavigation(r.Context(), navigationSnapshotView(snapshot))

	items := savedFilterViews(filters, pageState)
	create := view.SavedFilterCreateFormView{
		Error:      pageState.CreateError,
		Name:       strings.TrimSpace(pageState.CreateName),
		Query:      strings.TrimSpace(pageState.CreateQuery),
		IsFavorite: pageState.CreateFavorite,
	}
	if err := view.BaseLayout("Filter", view.SavedFiltersPage(items, create)).Render(ctx, w); err != nil {
		http.Error(w, "render page", http.StatusInternalServerError)
	}
}

func savedFilterViews(filters []db.SavedFilter, pageState savedFiltersPageState) []view.SavedFilterItemView {
	items := make([]view.SavedFilterItemView, 0, len(filters))
	for _, filter := range filters {
		item := view.SavedFilterItemView{
			ID:            filter.ID,
			Name:          filter.Name,
			Query:         filter.Query,
			IsFavorite:    filter.IsFavorite,
			ServerVersion: filter.ServerVersion,
		}
		if filter.ID == pageState.EditFilterID {
			item.EditError = pageState.EditError
			item.EditName = strings.TrimSpace(pageState.EditName)
			item.EditQuery = strings.TrimSpace(pageState.EditQuery)
			item.EditFavorite = pageState.EditFavorite
			item.EditFavoriteValid = pageState.EditFavoriteValid
		}
		if filter.ID == pageState.DeleteFilterID {
			item.DeleteError = pageState.DeleteError
		}
		items = append(items, item)
	}
	return items
}

func editFilterPageState(filterID string, errorMessage string, name string, filterQuery string, favorite bool, favoriteValid bool) savedFiltersPageState {
	return savedFiltersPageState{
		EditFilterID:      strings.TrimSpace(filterID),
		EditError:         errorMessage,
		EditName:          strings.TrimSpace(name),
		EditQuery:         strings.TrimSpace(filterQuery),
		EditFavorite:      favorite,
		EditFavoriteValid: favoriteValid,
	}
}

func deleteFilterPageState(filterID string, errorMessage string) savedFiltersPageState {
	return savedFiltersPageState{
		DeleteFilterID: strings.TrimSpace(filterID),
		DeleteError:    errorMessage,
	}
}

func formBool(value string) bool {
	trimmed := strings.TrimSpace(value)
	return strings.EqualFold(trimmed, "1") || strings.EqualFold(trimmed, "true") || strings.EqualFold(trimmed, "on")
}

func parseDeleteFormValues(w http.ResponseWriter, r *http.Request, database *db.Database, filterID string) (url.Values, bool) {
	formValues := r.URL.Query()
	if strings.HasPrefix(r.Header.Get("Content-Type"), "application/x-www-form-urlencoded") {
		bodyBytes, err := io.ReadAll(io.LimitReader(r.Body, 8*1024))
		if err != nil {
			renderSavedFiltersPage(w, r, database, deleteFilterPageState(filterID, "ungültige eingabe"), http.StatusOK)
			return nil, false
		}
		if len(bodyBytes) > 0 {
			parsed, err := url.ParseQuery(string(bodyBytes))
			if err != nil {
				renderSavedFiltersPage(w, r, database, deleteFilterPageState(filterID, "ungültige eingabe"), http.StatusOK)
				return nil, false
			}
			for key, values := range parsed {
				for _, value := range values {
					formValues.Add(key, value)
				}
			}
		}
	}
	return formValues, true
}
