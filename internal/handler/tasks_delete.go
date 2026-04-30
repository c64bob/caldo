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
		subtasksAction := strings.TrimSpace(r.FormValue("subtasks_action"))
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
				if subtasksAction == "" {
					subtasksAction = strings.TrimSpace(values.Get("subtasks_action"))
				}
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

		directSubtaskIDs, err := deps.database.ListDirectSubtaskIDs(r.Context(), taskID)
		if err != nil {
			http.Error(w, "failed to load subtasks", http.StatusInternalServerError)
			return
		}
		if len(directSubtaskIDs) > 0 {
			switch subtasksAction {
			case "":
				http.Error(w, fmt.Sprintf("direct subtasks require confirmation: %d", len(directSubtaskIDs)), http.StatusConflict)
				return
			case "cancel":
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("task deletion canceled"))
				return
			case "delete_all":
			default:
				http.Error(w, "invalid subtasks_action", http.StatusBadRequest)
				return
			}
		}
		parentBase, err := deps.database.LoadTaskUpdateBase(r.Context(), taskID, "")
		if err != nil {
			http.Error(w, "task version conflict", http.StatusConflict)
			return
		}
		if parentBase.ExpectedVersion != expectedVersion {
			http.Error(w, "task version conflict", http.StatusConflict)
			return
		}

		deleteTaskIDs := append(append([]string{}, directSubtaskIDs...), taskID)
		todoCredentials := caldav.Credentials{URL: creds.URL, Username: creds.Username, Password: creds.Password}

		for _, deleteTaskID := range deleteTaskIDs {
			base, baseErr := deps.database.LoadTaskUpdateBase(r.Context(), deleteTaskID, "")
			if baseErr != nil {
				http.Error(w, "task version conflict", http.StatusConflict)
				return
			}
			deleteExpectedVersion := base.ExpectedVersion
			if deleteTaskID == taskID {
				deleteExpectedVersion = expectedVersion
			}
			prepared, prepErr := deps.database.PrepareTaskDelete(r.Context(), db.TaskDeleteInput{
				TaskID:          deleteTaskID,
				ExpectedVersion: deleteExpectedVersion,
				SessionID:       sessionID,
				TabID:           tabID,
			})
			if prepErr != nil {
				if errors.Is(prepErr, db.ErrTaskVersionMismatch) {
					http.Error(w, "task version conflict", http.StatusConflict)
					return
				}
				http.Error(w, "failed to save pending task delete", http.StatusInternalServerError)
				return
			}

			err = deps.todos.DeleteVTODO(r.Context(), todoCredentials, prepared.Href, prepared.ETag)
			if err != nil {
				persistCtx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), taskDeletePersistTimeout)
				defer cancel()
				if errors.Is(err, caldav.ErrPreconditionFailed) {
					if markErr := deps.database.MarkTaskDeleteConflict(persistCtx, deleteTaskID, prepared.PendingVersion); markErr != nil {
						http.Error(w, "failed to persist task delete conflict state", http.StatusInternalServerError)
						return
					}
					http.Error(w, "task version conflict", http.StatusConflict)
					return
				}
				if markErr := deps.database.MarkTaskDeleteError(persistCtx, deleteTaskID, prepared.PendingVersion); markErr != nil {
					http.Error(w, "failed to persist task delete error state", http.StatusInternalServerError)
					return
				}
				http.Error(w, "failed to delete task on caldav server", http.StatusBadGateway)
				return
			}

			persistCtx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), taskDeletePersistTimeout)
			defer cancel()
			if err := deps.database.MarkTaskDeleteSynced(persistCtx, deleteTaskID, prepared.PendingVersion); err != nil {
				http.Error(w, "failed to persist synced task delete", http.StatusInternalServerError)
				return
			}
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("task deleted"))
	}
}
