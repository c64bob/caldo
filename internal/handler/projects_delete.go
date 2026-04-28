package handler

import (
	"context"
	"errors"
	"fmt"
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
}

const projectDeletePersistTimeout = 5 * time.Second

// ProjectDelete deletes a project by deleting the remote CalDAV calendar first, then deleting local project and tasks.
func ProjectDelete(deps projectDeleteDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		formValues := r.URL.Query()
		if strings.HasPrefix(r.Header.Get("Content-Type"), "application/x-www-form-urlencoded") {
			bodyBytes, err := io.ReadAll(io.LimitReader(r.Body, 8*1024))
			if err != nil {
				http.Error(w, "invalid form payload", http.StatusBadRequest)
				return
			}
			if len(bodyBytes) > 0 {
				parsed, err := url.ParseQuery(string(bodyBytes))
				if err != nil {
					http.Error(w, "invalid form payload", http.StatusBadRequest)
					return
				}
				for key, values := range parsed {
					for _, value := range values {
						formValues.Add(key, value)
					}
				}
			}
		}

		projectID := strings.TrimSpace(chi.URLParam(r, "projectID"))
		if projectID == "" {
			http.Error(w, "project id is required", http.StatusBadRequest)
			return
		}

		expectedVersion, err := strconv.Atoi(strings.TrimSpace(formValues.Get("expected_version")))
		if err != nil {
			http.Error(w, "expected_version is required", http.StatusBadRequest)
			return
		}

		confirmationName := strings.TrimSpace(formValues.Get("confirmation_name"))
		if confirmationName == "" {
			http.Error(w, "confirmation_name is required", http.StatusBadRequest)
			return
		}

		base, err := deps.database.LoadProjectDeleteBase(r.Context(), projectID, expectedVersion, confirmationName)
		if err != nil {
			switch {
			case errors.Is(err, db.ErrProjectNotFound):
				http.Error(w, "project not found", http.StatusNotFound)
			case errors.Is(err, db.ErrProjectVersionMismatch):
				http.Error(w, "project version conflict", http.StatusConflict)
			case errors.Is(err, db.ErrProjectDeleteConfirmationMismatch):
				http.Error(w, fmt.Sprintf("confirmation required for project %q with %d tasks", base.CurrentName, base.AffectedTaskCount), http.StatusConflict)
			default:
				http.Error(w, "failed to load project", http.StatusInternalServerError)
			}
			return
		}

		credentials, err := deps.database.LoadCalDAVCredentials(r.Context(), deps.encryptionKey)
		if err != nil {
			cancelCtx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), projectDeletePersistTimeout)
			defer cancel()
			_ = deps.database.CancelProjectDeleteReservation(cancelCtx, projectID, base.ReservedVersion)
			http.Error(w, "caldav credentials unavailable", http.StatusFailedDependency)
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
			http.Error(w, "failed to delete project on caldav server", http.StatusBadGateway)
			return
		}

		persistCtx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), projectDeletePersistTimeout)
		defer cancel()

		if err := deps.database.DeleteProject(persistCtx, projectID, base.ReservedVersion); err != nil {
			if errors.Is(err, db.ErrProjectVersionMismatch) {
				http.Error(w, "project version conflict", http.StatusConflict)
				return
			}
			http.Error(w, "failed to store project deletion", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("project deleted"))
	}
}
