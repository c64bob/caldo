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
	"caldo/internal/model"
	"github.com/go-chi/chi/v5"
)

// TaskComplete marks a task as completed with synchronous CalDAV write-through.
func TaskComplete(deps taskUpdateDependencies) http.HandlerFunc {
	return taskSetCompletion(deps, true)
}

// TaskReopen marks a completed task as open with synchronous CalDAV write-through.
func TaskReopen(deps taskUpdateDependencies) http.HandlerFunc {
	return taskSetCompletion(deps, false)
}

func taskSetCompletion(deps taskUpdateDependencies, completed bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form payload", http.StatusBadRequest)
			return
		}

		taskID := chi.URLParam(r, "taskID")
		expectedVersion, err := strconv.Atoi(strings.TrimSpace(r.FormValue("expected_version")))
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

		base, err := deps.database.LoadTaskUpdateBase(r.Context(), taskID, "")
		if err != nil {
			switch {
			case errors.Is(err, db.ErrTaskNotFound):
				http.Error(w, "task not found", http.StatusNotFound)
			default:
				http.Error(w, "failed to load task", http.StatusInternalServerError)
			}
			return
		}

		status := "needs-action"
		patch := model.VTODOPatch{Status: &status, ClearCompleted: true}
		if completed {
			status = "completed"
			now := time.Now().UTC()
			patch.CompletedAt = &now
			patch.ClearCompleted = false
		}

		rawVTODO := model.PatchVTODO(base.RawVTODO, patch)
		parsed := model.ParseVTODOFields(rawVTODO)
		creds, err := deps.database.LoadCalDAVCredentials(r.Context(), deps.encryptionKey)
		if err != nil {
			http.Error(w, "caldav credentials unavailable", http.StatusFailedDependency)
			return
		}
		input := db.TaskUpdateInput{
			TaskID:          taskID,
			ExpectedVersion: expectedVersion,
			SessionID:       sessionID,
			TabID:           tabID,
			ProjectID:       base.ProjectID,
			ProjectName:     base.ProjectName,
			Href:            base.Href,
			ETag:            base.ETag,
			RawVTODO:        rawVTODO,
			Title:           parsed.Title,
			Description:     parsed.Description,
			Status:          parsed.Status,
			DueDate:         nullableDate(parsed.DueDate),
			DueAt:           nullableTime(parsed.DueAt),
			Priority:        nullableInt(parsed.Priority),
			LabelNames:      nullableCSV(parsed.Categories),
		}

		prepared, err := deps.database.PrepareTaskUpdate(r.Context(), input)
		if err != nil {
			if errors.Is(err, db.ErrTaskVersionMismatch) {
				http.Error(w, "task version conflict", http.StatusConflict)
				return
			}
			http.Error(w, "failed to save pending task update", http.StatusInternalServerError)
			return
		}

		todoCredentials := caldav.Credentials{URL: creds.URL, Username: creds.Username, Password: creds.Password}

		newETag, err := deps.todos.PutVTODOUpdate(r.Context(), todoCredentials, prepared.PreviousHref, rawVTODO, prepared.PreviousETag)
		if err != nil {
			persistCtx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), taskUpdatePersistTimeout)
			defer cancel()
			if errors.Is(err, caldav.ErrPreconditionFailed) {
				if markErr := deps.database.MarkTaskUpdateConflict(persistCtx, taskID, prepared.PendingVersion); markErr != nil {
					http.Error(w, "failed to persist task update conflict state", http.StatusInternalServerError)
					return
				}
				http.Error(w, "task version conflict", http.StatusConflict)
				return
			}
			if markErr := deps.database.MarkTaskUpdateError(persistCtx, taskID, prepared.PendingVersion); markErr != nil {
				http.Error(w, "failed to persist task update error state", http.StatusInternalServerError)
				return
			}
			http.Error(w, "failed to update task on caldav server", http.StatusBadGateway)
			return
		}

		persistCtx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), taskUpdatePersistTimeout)
		defer cancel()
		if err := deps.database.MarkTaskUpdateSynced(persistCtx, taskID, prepared.PendingVersion, newETag); err != nil {
			http.Error(w, "failed to persist synced task update", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("task completion updated"))
	}
}
