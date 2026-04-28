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
}

const projectRenamePersistTimeout = 5 * time.Second

// ProjectRename renames a project by renaming the remote CalDAV calendar first, then persisting locally.
func ProjectRename(deps projectRenameDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form payload", http.StatusBadRequest)
			return
		}

		projectID := strings.TrimSpace(chi.URLParam(r, "projectID"))
		if projectID == "" {
			http.Error(w, "project id is required", http.StatusBadRequest)
			return
		}

		expectedVersion, err := strconv.Atoi(strings.TrimSpace(r.FormValue("expected_version")))
		if err != nil {
			http.Error(w, "expected_version is required", http.StatusBadRequest)
			return
		}

		displayName := strings.TrimSpace(r.FormValue("display_name"))
		if displayName == "" {
			http.Error(w, "display_name is required", http.StatusBadRequest)
			return
		}

		base, err := deps.database.LoadProjectRenameBase(r.Context(), projectID, expectedVersion, displayName)
		if err != nil {
			switch {
			case errors.Is(err, db.ErrProjectNotFound):
				http.Error(w, "project not found", http.StatusNotFound)
			case errors.Is(err, db.ErrProjectVersionMismatch):
				http.Error(w, "project version conflict", http.StatusConflict)
			default:
				http.Error(w, "failed to load project", http.StatusInternalServerError)
			}
			return
		}

		credentials, err := deps.database.LoadCalDAVCredentials(r.Context(), deps.encryptionKey)
		if err != nil {
			http.Error(w, "caldav credentials unavailable", http.StatusFailedDependency)
			return
		}

		renamedCalendar, err := deps.calendar.RenameCalendar(r.Context(), caldav.Credentials{
			URL:      credentials.URL,
			Username: credentials.Username,
			Password: credentials.Password,
		}, base.CalendarHref, base.RequestedName)
		if err != nil {
			http.Error(w, "failed to rename project on caldav server", http.StatusBadGateway)
			return
		}

		persistCtx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), projectRenamePersistTimeout)
		defer cancel()

		if err := deps.database.RenameProject(persistCtx, projectID, expectedVersion, renamedCalendar.DisplayName); err != nil {
			if errors.Is(err, db.ErrProjectVersionMismatch) {
				http.Error(w, "project version conflict", http.StatusConflict)
				return
			}
			http.Error(w, "failed to store project rename", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("project renamed"))
	}
}
