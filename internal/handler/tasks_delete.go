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

const taskDeletePersistTimeout = 5 * time.Second

// TaskDelete deletes a task with synchronous CalDAV write-through.
func TaskDelete(deps taskUpdateDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form payload", http.StatusBadRequest)
			return
		}

		taskID := chi.URLParam(r, "taskID")
		expectedVersionValue := strings.TrimSpace(r.FormValue("expected_version"))
		if expectedVersionValue == "" && r.Method == http.MethodDelete {
			expectedVersionValue = strings.TrimSpace(r.URL.Query().Get("expected_version"))
			if expectedVersionValue == "" {
				body, readErr := io.ReadAll(r.Body)
				if readErr != nil {
					http.Error(w, "invalid form payload", http.StatusBadRequest)
					return
				}
				_ = r.Body.Close()
				r.Body = io.NopCloser(strings.NewReader(string(body)))
				values, parseErr := url.ParseQuery(string(body))
				if parseErr != nil {
					http.Error(w, "invalid form payload", http.StatusBadRequest)
					return
				}
				expectedVersionValue = strings.TrimSpace(values.Get("expected_version"))
			}
		}

		expectedVersion, err := strconv.Atoi(expectedVersionValue)
		if err != nil {
			http.Error(w, "expected_version is required", http.StatusBadRequest)
			return
		}

		tabID := strings.TrimSpace(r.Header.Get("X-Tab-ID"))
		if tabID == "" {
			http.Error(w, "X-Tab-ID header is required", http.StatusBadRequest)
			return
		}
		sessionID := strings.TrimSpace(r.Header.Get("X-Forwarded-User"))
		if sessionID == "" {
			sessionID = "single-user-session"
		}

		creds, err := deps.database.LoadCalDAVCredentials(r.Context(), deps.encryptionKey)
		if err != nil {
			http.Error(w, "caldav credentials unavailable", http.StatusFailedDependency)
			return
		}

		prepared, err := deps.database.PrepareTaskDelete(r.Context(), db.TaskDeleteInput{
			TaskID:          taskID,
			ExpectedVersion: expectedVersion,
			SessionID:       sessionID,
			TabID:           tabID,
		})
		if err != nil {
			if errors.Is(err, db.ErrTaskVersionMismatch) {
				http.Error(w, "task version conflict", http.StatusConflict)
				return
			}
			http.Error(w, "failed to save pending task delete", http.StatusInternalServerError)
			return
		}

		todoCredentials := caldav.Credentials{URL: creds.URL, Username: creds.Username, Password: creds.Password}

		err = deps.todos.DeleteVTODO(r.Context(), todoCredentials, prepared.Href, prepared.ETag)
		if err != nil {
			persistCtx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), taskDeletePersistTimeout)
			defer cancel()
			if errors.Is(err, caldav.ErrPreconditionFailed) {
				if markErr := deps.database.MarkTaskDeleteConflict(persistCtx, taskID, prepared.PendingVersion); markErr != nil {
					http.Error(w, "failed to persist task delete conflict state", http.StatusInternalServerError)
					return
				}
				http.Error(w, "task version conflict", http.StatusConflict)
				return
			}
			if markErr := deps.database.MarkTaskDeleteError(persistCtx, taskID, prepared.PendingVersion); markErr != nil {
				http.Error(w, "failed to persist task delete error state", http.StatusInternalServerError)
				return
			}
			http.Error(w, "failed to delete task on caldav server", http.StatusBadGateway)
			return
		}

		persistCtx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), taskDeletePersistTimeout)
		defer cancel()
		if err := deps.database.MarkTaskDeleteSynced(persistCtx, taskID, prepared.PendingVersion); err != nil {
			http.Error(w, "failed to persist synced task delete", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("task deleted"))
	}
}
