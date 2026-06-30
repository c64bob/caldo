package handler

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"caldo/internal/caldav"
	"caldo/internal/db"
	"caldo/internal/view"
	"github.com/go-chi/chi/v5"
)

// ProjectsPage renders the projects navigation page.
func ProjectsPage(database *db.Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		renderProjectsPage(w, r, database, projectsPageState{}, http.StatusOK)
	}
}

// ProjectTasksPage renders the task list for one project.
func ProjectTasksPage(deps dateViewDependencies) http.HandlerFunc {
	nowFn := withDefaultNow(deps.now)

	return func(w http.ResponseWriter, r *http.Request) {
		if deps.database == nil {
			renderPageError(w, r, "Projekt", "Projekt laden", http.StatusInternalServerError)
			return
		}

		projectID := strings.TrimSpace(chi.URLParam(r, "projectID"))
		if projectID == "" {
			http.Error(w, "project id is required", http.StatusBadRequest)
			return
		}

		project, err := deps.database.ResolveTaskProject(r.Context(), projectID)
		if err != nil {
			if errors.Is(err, db.ErrTaskProjectNotFound) {
				renderPageError(w, r, "Projekt", "Projekt laden", http.StatusNotFound)
				return
			}
			renderPageError(w, r, "Projekt", "Projekt laden", http.StatusInternalServerError)
			return
		}

		reference := nowFn()
		tasks, err := deps.database.ListProjectTasks(r.Context(), project.ID, 200)
		if err != nil {
			renderPageError(w, r, project.DisplayName, "Projekt laden", http.StatusInternalServerError)
			return
		}

		projectOptions, err := taskEditProjectOptions(r.Context(), deps.database)
		if err != nil {
			renderPageError(w, r, project.DisplayName, "Projekt laden", http.StatusInternalServerError)
			return
		}

		snapshot, err := deps.database.LoadNavigationSnapshot(r.Context(), reference)
		if err != nil {
			renderPageError(w, r, project.DisplayName, "Projekt laden", http.StatusInternalServerError)
			return
		}

		ctx := view.WithNavigation(r.Context(), navigationSnapshotViewWithActiveProject(snapshot, project.ID))
		create := view.InlineTaskCreateView{
			Enabled:     true,
			ProjectID:   project.ID,
			ProjectName: project.DisplayName,
			Placeholder: "Aufgabe in " + project.DisplayName + " hinzufügen",
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := view.BaseLayout(project.DisplayName, view.DateScopedTasksPage(project.DisplayName, "Keine offenen Aufgaben in diesem Projekt.", datedTaskRows(tasks, projectOptions, reference), create)).Render(ctx, w); err != nil {
			http.Error(w, "render page", http.StatusInternalServerError)
		}
	}
}

type projectsPageState struct {
	PageSuccess     string
	CreateSuccess   string
	CreateError     string
	CreateValue     string
	RenameProjectID string
	RenameSuccess   string
	RenameError     string
	RenameValue     string
	DeleteProjectID string
	DeleteError     string
	DeleteValue     string
}

func renderProjectsPage(w http.ResponseWriter, r *http.Request, database *db.Database, pageState projectsPageState, status int) {
	if database == nil {
		renderPageError(w, r, "Projekte", "Projekte laden", http.StatusInternalServerError)
		return
	}

	snapshot, err := database.LoadNavigationSnapshot(r.Context(), time.Now())
	if err != nil {
		renderPageError(w, r, "Projekte", "Projekte laden", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if status != http.StatusOK {
		w.WriteHeader(status)
	}
	ctx := view.WithNavigation(r.Context(), navigationSnapshotView(snapshot))
	projects := navigationProjectsView(snapshot.Projects)
	for index := range projects {
		if projects[index].ID == pageState.RenameProjectID {
			projects[index].RenameSuccess = pageState.RenameSuccess
			projects[index].RenameError = pageState.RenameError
			projects[index].RenameValue = pageState.RenameValue
		}
		if projects[index].ID == pageState.DeleteProjectID {
			projects[index].DeleteError = pageState.DeleteError
			projects[index].DeleteValue = pageState.DeleteValue
		}
	}
	feedback := view.ProjectFeedback{
		PageSuccess:   pageState.PageSuccess,
		CreateSuccess: pageState.CreateSuccess,
		CreateError:   pageState.CreateError,
		CreateValue:   pageState.CreateValue,
	}
	if err := view.BaseLayout("Projekte", view.ProjectsOverviewPage(projects, feedback)).Render(ctx, w); err != nil {
		http.Error(w, "render page", http.StatusInternalServerError)
	}
}

// LabelsPage renders the labels navigation page.
func LabelsPage(database *db.Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		renderLabelsPage(w, r, database, labelsPageState{}, http.StatusOK)
	}
}

func renderLabelsPage(w http.ResponseWriter, r *http.Request, database *db.Database, pageState labelsPageState, status int) {
	if database == nil {
		renderPageError(w, r, "Labels", "Labels laden", http.StatusInternalServerError)
		return
	}

	snapshot, err := database.LoadNavigationSnapshot(r.Context(), time.Now())
	if err != nil {
		renderPageError(w, r, "Labels", "Labels laden", http.StatusInternalServerError)
		return
	}

	labels := navigationLabelsView(snapshot.Labels)
	for index := range labels {
		if labels[index].ID == pageState.RenameLabelID {
			labels[index].RenameError = pageState.RenameError
			labels[index].RenameValue = pageState.RenameValue
		}
		if labels[index].ID == pageState.DeleteLabelID {
			labels[index].DeleteError = pageState.DeleteError
			labels[index].DeleteValue = pageState.DeleteValue
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if status != http.StatusOK {
		w.WriteHeader(status)
	}
	ctx := view.WithNavigation(r.Context(), navigationSnapshotView(snapshot))
	if err := view.BaseLayout("Labels", view.LabelsOverviewPage(labels, view.LabelFeedback{PageSuccess: pageState.PageSuccess})).Render(ctx, w); err != nil {
		http.Error(w, "render page", http.StatusInternalServerError)
	}
}

// LabelTasksPage renders the task list for one label.
func LabelTasksPage(deps dateViewDependencies) http.HandlerFunc {
	nowFn := withDefaultNow(deps.now)

	return func(w http.ResponseWriter, r *http.Request) {
		if deps.database == nil {
			renderPageError(w, r, "Labels", "Label laden", http.StatusInternalServerError)
			return
		}

		labelID := strings.TrimSpace(chi.URLParam(r, "labelID"))
		if labelID == "" {
			http.Error(w, "label id is required", http.StatusBadRequest)
			return
		}

		label, err := deps.database.LoadLabelDetail(r.Context(), labelID)
		if err != nil {
			if errors.Is(err, db.ErrLabelNotFound) {
				renderPageError(w, r, "Label", "Label laden", http.StatusNotFound)
				return
			}
			renderPageError(w, r, "Label", "Label laden", http.StatusInternalServerError)
			return
		}

		reference := nowFn()
		tasks, err := deps.database.ListLabelTasks(r.Context(), label.ID, 200)
		if err != nil {
			renderPageError(w, r, label.Name, "Label laden", http.StatusInternalServerError)
			return
		}

		projectOptions, err := taskEditProjectOptions(r.Context(), deps.database)
		if err != nil {
			renderPageError(w, r, label.Name, "Label laden", http.StatusInternalServerError)
			return
		}

		snapshot, err := deps.database.LoadNavigationSnapshot(r.Context(), reference)
		if err != nil {
			renderPageError(w, r, label.Name, "Label laden", http.StatusInternalServerError)
			return
		}

		ctx := view.WithNavigation(r.Context(), navigationSnapshotViewWithActiveLabel(snapshot, label.ID))
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := view.BaseLayout(label.Name, view.DateScopedTasksPage(label.Name, "Keine Aufgaben mit diesem Label.", datedTaskRows(tasks, projectOptions, reference), view.InlineTaskCreateView{})).Render(ctx, w); err != nil {
			http.Error(w, "render page", http.StatusInternalServerError)
		}
	}
}

// FiltersPage renders the filters navigation page.
func FiltersPage(database *db.Database) http.HandlerFunc {
	return SavedFiltersPage(savedFilterDependencies{database: database})
}

// SettingsPage renders the settings page for normal operation.
func SettingsPage(deps settingsDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		state := settingsPageState{}
		if strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("caldav")), "saved") {
			state.CalDAVSuccess = "caldav-einstellungen gespeichert"
		}
		renderSettingsPage(w, r, deps, state, http.StatusOK)
	}
}

func renderSettingsPage(w http.ResponseWriter, r *http.Request, deps settingsDependencies, pageState settingsPageState, status int) {
	if deps.database == nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	settings, err := deps.database.LoadAppSettings(r.Context())
	if err != nil {
		renderPageError(w, r, "Einstellungen", "Einstellungen laden", http.StatusInternalServerError)
		return
	}
	syncStatus, err := deps.database.LoadSyncStatus(r.Context())
	if err != nil {
		renderPageError(w, r, "Einstellungen", "Einstellungen laden", http.StatusInternalServerError)
		return
	}
	if pageState.PreserveCalDAV {
		settings.CalDAVURL = pageState.CalDAVURL
		settings.CalDAVUsername = pageState.CalDAVUsername
	}
	ctx := view.WithUIPreferences(r.Context(), settings.UILanguage, settings.DarkMode)

	available := pageState.Available
	calendarLoadError := pageState.CalendarsError
	calendarsLoaded := available != nil
	if available == nil && deps.calendar != nil {
		calendars, err := loadSettingsCalendars(ctx, deps)
		if err != nil {
			if calendarLoadError == "" {
				calendarLoadError = "kalender konnten nicht geladen werden"
			}
		} else {
			available = calendars
			calendarsLoaded = true
		}
	}

	httpsConfigured := r.TLS != nil || strings.EqualFold(strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")), "https")
	proxyUserPresent := strings.TrimSpace(r.Header.Get(deps.proxyUserHeader)) != ""
	settingsView := view.SettingsPageView{
		Settings:         settings,
		SyncStatus:       syncStatus,
		Available:        available,
		CalendarsLoaded:  calendarsLoaded,
		CalDAVError:      pageState.CalDAVError,
		CalDAVSuccess:    pageState.CalDAVSuccess,
		CalendarsError:   calendarLoadError,
		SelectedHrefs:    pageState.SelectedHrefs,
		DefaultHref:      pageState.DefaultHref,
		ProxyUserHeader:  deps.proxyUserHeader,
		ProxyUserPresent: proxyUserPresent,
		HTTPSConfigured:  httpsConfigured,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if status != http.StatusOK {
		w.WriteHeader(status)
	}
	if err := view.BaseLayout("Einstellungen", view.SettingsPageContent(settingsView)).Render(ctx, w); err != nil {
		http.Error(w, "render page", http.StatusInternalServerError)
	}
}

func loadSettingsCalendars(ctx context.Context, deps settingsDependencies) ([]caldav.Calendar, error) {
	if deps.database == nil || deps.calendar == nil {
		return nil, errors.New("settings calendars dependencies missing")
	}

	credentials, err := deps.database.LoadCalDAVCredentials(ctx, deps.encryptionKey)
	if err != nil {
		return nil, err
	}
	capabilities, err := deps.database.LoadCalDAVServerCapabilities(ctx)
	if err != nil {
		return nil, err
	}

	return deps.calendar.ListCalendars(ctx, calendarOperationCredentials(credentials, capabilities))
}

func renderPageError(w http.ResponseWriter, r *http.Request, title string, action string, status int) {
	w.WriteHeader(status)
	if err := view.BaseLayout(title, view.ErrorState(action, "serverfehler", false)).Render(r.Context(), w); err != nil {
		http.Error(w, http.StatusText(status), status)
	}
}
