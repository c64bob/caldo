package handler

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"caldo/internal/caldav"
	"caldo/internal/db"
	"github.com/go-chi/chi/v5"
)

type projectDeleteCalendarClient interface {
	DeleteCalendar(ctx context.Context, credentials caldav.Credentials, calendarHref string) error
}

type projectDeleteDependencies struct {
	database      *db.Database
	encryptionKey []byte
	calendar      projectDeleteCalendarClient
	broker        *eventBroker
}

const projectDeletePersistTimeout = 5 * time.Second

// ProjectDelete deletes a project by deleting the remote CalDAV calendar first, then deleting local project and tasks.
func ProjectDelete(deps projectDeleteDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		projectID := strings.TrimSpace(chi.URLParam(r, "projectID"))
		if projectID == "" {
			http.Error(w, "project id is required", http.StatusBadRequest)
			return
		}

		formValues := r.URL.Query()
		if strings.HasPrefix(r.Header.Get("Content-Type"), "application/x-www-form-urlencoded") {
			bodyBytes, err := io.ReadAll(io.LimitReader(r.Body, 8*1024))
			if err != nil {
				renderProjectsPage(w, r, deps.database, deletePageState(projectID, "ungültige eingabe", ""), http.StatusOK)
				return
			}
			if len(bodyBytes) > 0 {
				parsed, err := url.ParseQuery(string(bodyBytes))
				if err != nil {
					renderProjectsPage(w, r, deps.database, deletePageState(projectID, "ungültige eingabe", ""), http.StatusOK)
					return
				}
				for key, values := range parsed {
					for _, value := range values {
						formValues.Add(key, value)
					}
				}
			}
		}

		expectedVersion, err := strconv.Atoi(strings.TrimSpace(formValues.Get("expected_version")))
		if err != nil {
			renderProjectsPage(w, r, deps.database, deletePageState(projectID, "projektversion fehlt", formValues.Get("confirmation_name")), http.StatusOK)
			return
		}

		confirmationName := strings.TrimSpace(formValues.Get("confirmation_name"))
		if confirmationName == "" {
			renderProjectsPage(w, r, deps.database, deletePageState(projectID, "projektname als bestätigung erforderlich", formValues.Get("confirmation_name")), http.StatusOK)
			return
		}

		base, err := deps.database.LoadProjectDeleteBase(r.Context(), projectID, expectedVersion, confirmationName)
		if err != nil {
			switch {
			case errors.Is(err, db.ErrProjectNotFound):
				http.Error(w, "project not found", http.StatusNotFound)
			case errors.Is(err, db.ErrProjectVersionMismatch):
				renderProjectsPage(w, r, deps.database, deletePageState(projectID, "projekt wurde zwischenzeitlich geändert", confirmationName), http.StatusOK)
			case errors.Is(err, db.ErrProjectDeleteConfirmationMismatch):
				renderProjectsPage(w, r, deps.database, deletePageState(projectID, "bestätigung stimmt nicht mit dem projektnamen überein", confirmationName), http.StatusOK)
			default:
				renderProjectsPage(w, r, deps.database, deletePageState(projectID, "projekt konnte nicht geladen werden", confirmationName), http.StatusOK)
			}
			return
		}

		credentials, err := deps.database.LoadCalDAVCredentials(r.Context(), deps.encryptionKey)
		if err != nil {
			cancelCtx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), projectDeletePersistTimeout)
			defer cancel()
			_ = deps.database.CancelProjectDeleteReservation(cancelCtx, projectID, base.ReservedVersion)
			renderProjectsPage(w, r, deps.database, deletePageState(projectID, "caldav-zugangsdaten sind nicht verfügbar", confirmationName), http.StatusOK)
			return
		}

		if err := deps.calendar.DeleteCalendar(r.Context(), caldav.Credentials{
			URL:      credentials.URL,
			Username: credentials.Username,
			Password: credentials.Password,
		}, base.CalendarHref); err != nil {
			cancelCtx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), projectDeletePersistTimeout)
			defer cancel()
			_ = deps.database.CancelProjectDeleteReservation(cancelCtx, projectID, base.ReservedVersion)
			renderProjectsPage(w, r, deps.database, deletePageState(projectID, "projekt konnte nicht auf dem caldav-server gelöscht werden", confirmationName), http.StatusOK)
			return
		}

		persistCtx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), projectDeletePersistTimeout)
		defer cancel()

		if err := deps.database.DeleteProject(persistCtx, projectID, base.ReservedVersion); err != nil {
			if errors.Is(err, db.ErrProjectVersionMismatch) {
				renderProjectsPage(w, r, deps.database, deletePageState(projectID, "projekt wurde zwischenzeitlich geändert", confirmationName), http.StatusOK)
				return
			}
			renderProjectsPage(w, r, deps.database, deletePageState(projectID, "projektlöschung konnte nicht gespeichert werden", confirmationName), http.StatusOK)
			return
		}

		if deps.broker != nil {
			deps.broker.publish(appEvent{Type: "project", Resource: projectID, Version: base.ReservedVersion, OriginConnection: strings.TrimSpace(r.Header.Get("X-Tab-ID"))})
		}

		renderProjectsPage(w, r, deps.database, projectsPageState{}, http.StatusOK)
	}
}

func deletePageState(projectID string, errorMessage string, confirmationName string) projectsPageState {
	return projectsPageState{
		DeleteProjectID: strings.TrimSpace(projectID),
		DeleteError:     errorMessage,
		DeleteValue:     strings.TrimSpace(confirmationName),
	}
}
