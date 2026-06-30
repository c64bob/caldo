package handler

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"caldo/internal/caldav"
	"caldo/internal/db"
)

type projectCreateCalendarClient interface {
	CreateCalendar(ctx context.Context, credentials caldav.Credentials, displayName string) (caldav.Calendar, error)
}

type projectCreateDependencies struct {
	database      *db.Database
	encryptionKey []byte
	calendar      projectCreateCalendarClient
	broker        *eventBroker
}

const projectCreatePersistTimeout = 5 * time.Second

// ProjectCreate creates a new project by creating a remote CalDAV calendar first, then persisting locally.
func ProjectCreate(deps projectCreateDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		projectName := strings.TrimSpace(r.FormValue("display_name"))
		if projectName == "" {
			renderProjectsPage(w, r, deps.database, projectsPageState{CreateError: "projektname ist erforderlich", CreateValue: r.FormValue("display_name")}, http.StatusOK)
			return
		}

		credentials, err := deps.database.LoadCalDAVCredentials(r.Context(), deps.encryptionKey)
		if err != nil {
			renderProjectsPage(w, r, deps.database, projectsPageState{CreateError: "caldav-zugangsdaten sind nicht verfügbar", CreateValue: projectName}, http.StatusOK)
			return
		}

		capabilities, err := deps.database.LoadCalDAVServerCapabilities(r.Context())
		if err != nil {
			renderProjectsPage(w, r, deps.database, projectsPageState{CreateError: "caldav-fähigkeiten konnten nicht geladen werden", CreateValue: projectName}, http.StatusOK)
			return
		}

		createdCalendar, err := deps.calendar.CreateCalendar(r.Context(), calendarOperationCredentials(credentials, capabilities), projectName)
		if err != nil {
			renderProjectsPage(w, r, deps.database, projectsPageState{CreateError: projectCreateRemoteErrorMessage(err), CreateValue: projectName}, http.StatusOK)
			return
		}

		persistCtx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), projectCreatePersistTimeout)
		defer cancel()

		project, err := deps.database.InsertProject(persistCtx, db.NewProjectInput{
			CalendarHref: createdCalendar.Href,
			DisplayName:  createdCalendar.DisplayName,
			SyncStrategy: initialSyncStrategy(capabilities),
		})
		if err != nil {
			renderProjectsPage(w, r, deps.database, projectsPageState{CreateError: "projekt konnte nicht gespeichert werden", CreateValue: projectName}, http.StatusOK)
			return
		}

		if deps.broker != nil {
			deps.broker.publish(appEvent{Type: "project", Resource: project.ID, Version: 1, OriginConnection: strings.TrimSpace(r.Header.Get("X-Tab-ID"))})
		}

		renderProjectsPage(w, r, deps.database, projectsPageState{CreateSuccess: "projekt wurde angelegt"}, http.StatusCreated)
	}
}

func projectCreateRemoteErrorMessage(err error) string {
	statusCode, ok := caldav.HTTPStatusCode(err)
	if !ok {
		return "projekt konnte nicht auf dem caldav-server angelegt werden"
	}
	statusText := strconv.Itoa(statusCode)

	switch {
	case statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden:
		return "caldav-server hat die kalenderanlage verweigert (status " + statusText + ")"
	case statusCode == http.StatusNotFound:
		return "caldav-kalenderziel wurde nicht gefunden (status 404)"
	case statusCode == http.StatusMethodNotAllowed || statusCode == http.StatusNotImplemented:
		return "caldav-server unterstützt kalenderanlage hier nicht (status " + statusText + ")"
	case statusCode == http.StatusConflict:
		return "projekt konnte wegen eines kalenderpfad-konflikts nicht angelegt werden (status 409)"
	case statusCode >= http.StatusInternalServerError:
		return "caldav-serverfehler beim anlegen des projekts (status " + statusText + ")"
	default:
		return "projekt konnte nicht auf dem caldav-server angelegt werden (status " + statusText + ")"
	}
}
