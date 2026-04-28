package handler

import (
	"context"
	"net/http"
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
}

const projectCreatePersistTimeout = 5 * time.Second

// ProjectCreate creates a new project by creating a remote CalDAV calendar first, then persisting locally.
func ProjectCreate(deps projectCreateDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		projectName := strings.TrimSpace(r.FormValue("display_name"))
		if projectName == "" {
			http.Error(w, "display_name is required", http.StatusBadRequest)
			return
		}

		credentials, err := deps.database.LoadCalDAVCredentials(r.Context(), deps.encryptionKey)
		if err != nil {
			http.Error(w, "caldav credentials unavailable", http.StatusFailedDependency)
			return
		}

		capabilities, err := deps.database.LoadCalDAVServerCapabilities(r.Context())
		if err != nil {
			http.Error(w, "failed to load caldav server capabilities", http.StatusInternalServerError)
			return
		}

		createdCalendar, err := deps.calendar.CreateCalendar(r.Context(), caldav.Credentials{
			URL:      credentials.URL,
			Username: credentials.Username,
			Password: credentials.Password,
		}, projectName)
		if err != nil {
			http.Error(w, "failed to create project on caldav server", http.StatusBadGateway)
			return
		}

		persistCtx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), projectCreatePersistTimeout)
		defer cancel()

		_, err = deps.database.InsertProject(persistCtx, db.NewProjectInput{
			CalendarHref: createdCalendar.Href,
			DisplayName:  createdCalendar.DisplayName,
			SyncStrategy: initialSyncStrategy(capabilities),
		})
		if err != nil {
			http.Error(w, "failed to store project", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("project created"))
	}
}
