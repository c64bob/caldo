package handler

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"caldo/internal/caldav"
	"caldo/internal/db"
	"caldo/internal/model"
	"caldo/internal/view"
)

// CalDAVConnectionTester defines CalDAV connection tests for setup.
type CalDAVConnectionTester interface {
	TestConnection(ctx context.Context, credentials caldav.Credentials) (caldav.ServerCapabilities, error)
}

type setupDependencies struct {
	database      *db.Database
	encryptionKey []byte
	tester        CalDAVConnectionTester
	calendar      CalDAVCalendarClient
	todos         CalDAVTodoClient
	importBroker  *setupImportEventBroker
}

// CalDAVCalendarClient lists and creates CalDAV calendars during setup.
type CalDAVCalendarClient interface {
	ListCalendars(ctx context.Context, credentials caldav.Credentials) ([]caldav.Calendar, error)
	CreateCalendar(ctx context.Context, credentials caldav.Credentials, displayName string) (caldav.Calendar, error)
}

// CalDAVTodoClient lists VTODO resources for one calendar during initial import.
type CalDAVTodoClient interface {
	ListVTODOs(ctx context.Context, credentials caldav.Credentials, calendarHref string) ([]caldav.CalendarObject, error)
}

// SetupPage renders the setup step for CalDAV credential capture.
func SetupPage(w http.ResponseWriter, r *http.Request) {
	renderSetupCalDAVPage(w, r, "")
}

// SetupCalDAV stores encrypted credentials, executes a live CalDAV connection test, and advances the setup step on success.
func SetupCalDAV(deps setupDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.database == nil || deps.tester == nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		if err := r.ParseForm(); err != nil {
			renderSetupCalDAVPage(w, r, "ungültige eingabe")
			return
		}

		credentials := db.CalDAVCredentials{
			URL:      strings.TrimSpace(r.FormValue("caldav_url")),
			Username: strings.TrimSpace(r.FormValue("caldav_username")),
			Password: r.FormValue("caldav_password"),
		}

		if err := deps.database.SaveCalDAVCredentials(r.Context(), deps.encryptionKey, credentials); err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		capabilities, err := deps.tester.TestConnection(r.Context(), caldav.Credentials{
			URL:      credentials.URL,
			Username: credentials.Username,
			Password: credentials.Password,
		})
		if err != nil {
			renderSetupCalDAVPage(w, r, "verbindungstest fehlgeschlagen")
			return
		}

		if err := deps.database.SaveCalDAVServerCapabilities(r.Context(), db.CalDAVServerCapabilities{
			WebDAVSync: capabilities.WebDAVSync,
			CTag:       capabilities.CTag,
			ETag:       capabilities.ETag,
			FullScan:   capabilities.FullScan,
		}); err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		if err := deps.database.SaveSetupStep(r.Context(), "calendars"); err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/setup/calendars", http.StatusFound)
	}
}

func renderSetupCalDAVPage(w http.ResponseWriter, r *http.Request, errorMessage string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := view.BaseLayout("Caldo Setup", view.SetupCalDAVContent(errorMessage)).Render(r.Context(), w); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func SetupCalendarsPage(deps setupDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		calendars, err := loadCalendars(r.Context(), deps)
		if err != nil {
			if shouldRedirectToSetup(err) {
				http.Redirect(w, r, "/setup", http.StatusFound)
				return
			}
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		renderSetupCalendarsPage(w, r, calendars, "", nil)
	}
}

func SetupCalendars(deps setupDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		calendars, err := loadCalendars(r.Context(), deps)
		if err != nil {
			if shouldRedirectToSetup(err) {
				http.Redirect(w, r, "/setup", http.StatusFound)
				return
			}
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		selectedHrefs := r.Form["calendar_href"]
		defaultHref := strings.TrimSpace(r.FormValue("default_calendar_href"))
		newDefaultName := strings.TrimSpace(r.FormValue("new_default_project_name"))

		availableByHref := make(map[string]caldav.Calendar, len(calendars))
		for _, calendar := range calendars {
			availableByHref[calendar.Href] = calendar
		}

		selected := make([]db.SelectedCalendar, 0, len(selectedHrefs)+1)
		seen := make(map[string]struct{}, len(selectedHrefs))
		for _, href := range selectedHrefs {
			calendar, ok := availableByHref[href]
			if !ok {
				continue
			}
			if _, exists := seen[href]; exists {
				continue
			}
			seen[href] = struct{}{}
			selected = append(selected, db.SelectedCalendar{
				Href:        calendar.Href,
				DisplayName: calendar.DisplayName,
			})
		}

		if newDefaultName != "" {
			credentials, err := deps.database.LoadCalDAVCredentials(r.Context(), deps.encryptionKey)
			if err != nil {
				if shouldRedirectToSetup(err) {
					http.Redirect(w, r, "/setup", http.StatusFound)
					return
				}
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}

			createdCalendar, err := deps.calendar.CreateCalendar(r.Context(), caldav.Credentials{
				URL:      credentials.URL,
				Username: credentials.Username,
				Password: credentials.Password,
			}, newDefaultName)
			if err != nil {
				renderSetupCalendarsPage(w, r, calendars, "neues default-projekt konnte nicht angelegt werden", selectedHrefs)
				return
			}

			selected = append(selected, db.SelectedCalendar{
				Href:        createdCalendar.Href,
				DisplayName: createdCalendar.DisplayName,
			})
			defaultHref = createdCalendar.Href
		}

		if len(selected) == 0 {
			renderSetupCalendarsPage(w, r, calendars, "mindestens ein kalender muss ausgewählt werden", selectedHrefs)
			return
		}
		if defaultHref == "" || !slices.ContainsFunc(selected, func(calendar db.SelectedCalendar) bool {
			return calendar.Href == defaultHref
		}) {
			renderSetupCalendarsPage(w, r, calendars, "ein default-projekt ist erforderlich", selectedHrefs)
			return
		}

		capabilities, err := deps.database.LoadCalDAVServerCapabilities(r.Context())
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		if err := deps.database.SaveSetupCalendars(r.Context(), selected, defaultHref, initialSyncStrategy(capabilities)); err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/setup/import", http.StatusFound)
	}
}

// SetupImport starts the setup initial import in full-scan mode.
func SetupImport(deps setupDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.database == nil || deps.todos == nil || deps.importBroker == nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		status, err := deps.database.LoadSetupStatus(r.Context())
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		if status.Step != "import" {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		if !deps.importBroker.StartRun() {
			http.Error(w, http.StatusText(http.StatusConflict), http.StatusConflict)
			return
		}

		go executeSetupInitialImport(context.Background(), deps)
		w.WriteHeader(http.StatusAccepted)
	}
}

// SetupImportEvents streams setup import progress events via SSE.
func SetupImportEvents(deps setupDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.importBroker == nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		subscriber := deps.importBroker.Subscribe()
		defer deps.importBroker.Unsubscribe(subscriber.id)

		for {
			select {
			case <-r.Context().Done():
				return
			case event, ok := <-subscriber.ch:
				if !ok {
					return
				}
				if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Event, event.Data); err != nil {
					return
				}
				flusher.Flush()
			}
		}
	}
}

func executeSetupInitialImport(ctx context.Context, deps setupDependencies) {
	defer deps.importBroker.FinishRun()

	projects, err := deps.database.LoadSetupImportProjects(ctx)
	if err != nil {
		deps.importBroker.Publish(setupImportEvent{Event: "error", Data: "import failed"})
		return
	}
	if len(projects) == 0 {
		deps.importBroker.Publish(setupImportEvent{Event: "error", Data: "no setup projects found"})
		return
	}

	credentials, err := deps.database.LoadCalDAVCredentials(ctx, deps.encryptionKey)
	if err != nil {
		deps.importBroker.Publish(setupImportEvent{Event: "error", Data: "missing caldav credentials"})
		return
	}

	deps.importBroker.Publish(setupImportEvent{Event: "progress", Data: `{"phase":"start","completed":0}`})
	for index, project := range projects {
		objects, err := deps.todos.ListVTODOs(ctx, caldav.Credentials{
			URL:      credentials.URL,
			Username: credentials.Username,
			Password: credentials.Password,
		}, project.CalendarHref)
		if err != nil {
			deps.importBroker.Publish(setupImportEvent{Event: "error", Data: "import failed"})
			return
		}

		tasks := make([]db.ImportedTask, 0, len(objects))
		for _, object := range objects {
			parsed := model.ParseVTODOFields(object.RawVTODO)
			if parsed.UID == "" {
				continue
			}
			labels, _ := model.CategoriesToLabelsAndFavorite(parsed.Categories)
			completedAt := formatTimePointer(parsed.CompletedAt)
			dueAt := formatTimePointer(parsed.DueAt)
			title := parsed.Title
			if title == "" {
				title = parsed.UID
			}

			tasks = append(tasks, db.ImportedTask{
				UID:         parsed.UID,
				Href:        object.Href,
				ETag:        object.ETag,
				Title:       title,
				Description: parsed.Description,
				Status:      parsed.Status,
				CompletedAt: completedAt,
				DueDate:     parsed.DueDate,
				DueAt:       dueAt,
				Priority:    parsed.Priority,
				RRule:       parsed.RRule,
				ParentUID:   parsed.ParentUID,
				RawVTODO:    object.RawVTODO,
				BaseVTODO:   object.RawVTODO,
				LabelNames:  labels,
				ProjectName: project.DisplayName,
			})
		}

		if err := deps.database.ReplaceSetupProjectTasks(ctx, project.ID, tasks); err != nil {
			deps.importBroker.Publish(setupImportEvent{Event: "error", Data: "import failed"})
			return
		}

		deps.importBroker.Publish(setupImportEvent{
			Event: "progress",
			Data:  fmt.Sprintf(`{"phase":"calendar_done","completed":%d,"total":%d}`, index+1, len(projects)),
		})
	}

	deps.importBroker.Publish(setupImportEvent{Event: "done", Data: `{"phase":"done"}`})
}

func formatTimePointer(v *time.Time) *string {
	if v == nil {
		return nil
	}
	formatted := v.UTC().Format(time.RFC3339)
	return &formatted
}

func SetupComplete(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

func loadCalendars(ctx context.Context, deps setupDependencies) ([]caldav.Calendar, error) {
	if deps.database == nil || deps.calendar == nil {
		return nil, fmt.Errorf("setup calendars dependencies missing")
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

func renderSetupCalendarsPage(w http.ResponseWriter, r *http.Request, calendars []caldav.Calendar, errorMessage string, selectedHrefs []string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := view.BaseLayout("Caldo Setup", view.SetupCalendarsContent(calendars, errorMessage, selectedHrefs)).Render(r.Context(), w); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func initialSyncStrategy(capabilities db.CalDAVServerCapabilities) string {
	switch {
	case capabilities.WebDAVSync:
		return "webdav_sync"
	case capabilities.CTag:
		return "ctag"
	default:
		return "fullscan"
	}
}

func shouldRedirectToSetup(err error) bool {
	return errors.Is(err, db.ErrCalDAVCredentialsNotConfigured) || errors.Is(err, db.ErrCalDAVCredentialsUnavailable)
}
