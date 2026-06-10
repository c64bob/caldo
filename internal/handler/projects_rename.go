package handler

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"caldo/internal/caldav"
	"caldo/internal/db"
	"github.com/go-chi/chi/v5"
)

type projectRenameCalendarClient interface {
	RenameCalendar(ctx context.Context, credentials caldav.Credentials, calendarHref string, displayName string) (caldav.Calendar, error)
}

type projectRenameDependencies struct {
	database      *db.Database
	encryptionKey []byte
	calendar      projectRenameCalendarClient
	broker        *eventBroker
}

const projectRenamePersistTimeout = 5 * time.Second

// ProjectRename renames a project by renaming the remote CalDAV calendar first, then persisting locally.
func ProjectRename(deps projectRenameDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		projectID := strings.TrimSpace(chi.URLParam(r, "projectID"))
		if projectID == "" {
			http.Error(w, "project id is required", http.StatusBadRequest)
			return
		}

		if err := r.ParseForm(); err != nil {
			renderProjectsPage(w, r, deps.database, renamePageState(projectID, "ungültige eingabe", ""), http.StatusOK)
			return
		}

		expectedVersion, err := strconv.Atoi(strings.TrimSpace(r.FormValue("expected_version")))
		if err != nil {
			renderProjectsPage(w, r, deps.database, renamePageState(projectID, "projektversion fehlt", r.FormValue("display_name")), http.StatusOK)
			return
		}

		displayName := strings.TrimSpace(r.FormValue("display_name"))
		if displayName == "" {
			renderProjectsPage(w, r, deps.database, renamePageState(projectID, "projektname ist erforderlich", r.FormValue("display_name")), http.StatusOK)
			return
		}

		base, err := deps.database.LoadProjectRenameBase(r.Context(), projectID, expectedVersion, displayName)
		if err != nil {
			switch {
			case errors.Is(err, db.ErrProjectNotFound):
				http.Error(w, "project not found", http.StatusNotFound)
			case errors.Is(err, db.ErrProjectVersionMismatch):
				renderProjectsPage(w, r, deps.database, renamePageState(projectID, "projekt wurde zwischenzeitlich geändert", displayName), http.StatusOK)
			default:
				renderProjectsPage(w, r, deps.database, renamePageState(projectID, "projekt konnte nicht geladen werden", displayName), http.StatusOK)
			}
			return
		}

		credentials, err := deps.database.LoadCalDAVCredentials(r.Context(), deps.encryptionKey)
		if err != nil {
			renderProjectsPage(w, r, deps.database, renamePageState(projectID, "caldav-zugangsdaten sind nicht verfügbar", displayName), http.StatusOK)
			return
		}

		renamedCalendar, err := deps.calendar.RenameCalendar(r.Context(), caldav.Credentials{
			URL:      credentials.URL,
			Username: credentials.Username,
			Password: credentials.Password,
		}, base.CalendarHref, base.RequestedName)
		if err != nil {
			cancelCtx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), projectRenamePersistTimeout)
			defer cancel()
			_ = deps.database.CancelProjectRenameReservation(cancelCtx, projectID, base.ReservedVersion)
			renderProjectsPage(w, r, deps.database, renamePageState(projectID, "projekt konnte nicht auf dem caldav-server umbenannt werden", displayName), http.StatusOK)
			return
		}

		persistCtx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), projectRenamePersistTimeout)
		defer cancel()

		if err := deps.database.RenameProject(persistCtx, projectID, base.ReservedVersion, renamedCalendar.DisplayName); err != nil {
			if errors.Is(err, db.ErrProjectVersionMismatch) {
				renderProjectsPage(w, r, deps.database, renamePageState(projectID, "projekt wurde zwischenzeitlich geändert", displayName), http.StatusOK)
				return
			}
			renderProjectsPage(w, r, deps.database, renamePageState(projectID, "projektumbenennung konnte nicht gespeichert werden", displayName), http.StatusOK)
			return
		}

		if deps.broker != nil {
			deps.broker.publish(appEvent{Type: "project", Resource: projectID, Version: base.ReservedVersion, OriginConnection: strings.TrimSpace(r.Header.Get("X-Tab-ID"))})
		}

		renderProjectsPage(w, r, deps.database, projectsPageState{}, http.StatusOK)
	}
}

func renamePageState(projectID string, errorMessage string, displayName string) projectsPageState {
	return projectsPageState{
		RenameProjectID: strings.TrimSpace(projectID),
		RenameError:     errorMessage,
		RenameValue:     strings.TrimSpace(displayName),
	}
}
