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
)

// ProjectsPage renders the projects navigation page.
func ProjectsPage(database *db.Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		renderProjectsPage(w, r, database, projectsPageState{}, http.StatusOK)
	}
}

type projectsPageState struct {
	CreateError     string
	CreateValue     string
	RenameProjectID string
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
			projects[index].RenameError = pageState.RenameError
			projects[index].RenameValue = pageState.RenameValue
		}
		if projects[index].ID == pageState.DeleteProjectID {
			projects[index].DeleteError = pageState.DeleteError
			projects[index].DeleteValue = pageState.DeleteValue
		}
	}
	if err := view.BaseLayout("Projekte", view.ProjectsOverviewPage(projects, pageState.CreateError, pageState.CreateValue)).Render(ctx, w); err != nil {
		http.Error(w, "render page", http.StatusInternalServerError)
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
	return SavedFiltersPage(savedFilterDependencies{database: database})
}

// SettingsPage renders the settings page for normal operation.
func SettingsPage(deps settingsDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		renderSettingsPage(w, r, deps, settingsPageState{}, http.StatusOK)
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
	if pageState.PreserveCalDAV {
		settings.CalDAVURL = pageState.CalDAVURL
		settings.CalDAVUsername = pageState.CalDAVUsername
	}

	available := pageState.Available
	calendarLoadError := pageState.CalendarsError
	if available == nil && deps.calendar != nil {
		calendars, err := loadSettingsCalendars(r.Context(), deps)
		if err != nil {
			if calendarLoadError == "" {
				calendarLoadError = "kalender konnten nicht geladen werden"
			}
		} else {
			available = calendars
		}
	}

	httpsConfigured := r.TLS != nil || strings.EqualFold(strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")), "https")
	proxyUserPresent := strings.TrimSpace(r.Header.Get(deps.proxyUserHeader)) != ""
	settingsView := view.SettingsPageView{
		Settings:         settings,
		Available:        available,
		CalDAVError:      pageState.CalDAVError,
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
	if err := view.BaseLayout("Einstellungen", view.SettingsPageContent(settingsView)).Render(r.Context(), w); err != nil {
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

	return deps.calendar.ListCalendars(ctx, caldav.Credentials{
		URL:      credentials.URL,
		Username: credentials.Username,
		Password: credentials.Password,
	})
}

func renderPageError(w http.ResponseWriter, r *http.Request, title string, action string, status int) {
	w.WriteHeader(status)
	if err := view.BaseLayout(title, view.ErrorState(action, "serverfehler", false)).Render(r.Context(), w); err != nil {
		http.Error(w, http.StatusText(status), status)
	}
}
