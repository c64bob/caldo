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
		subtaskAction := strings.TrimSpace(r.FormValue("subtasks_action"))

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
		if expectedVersion != base.ExpectedVersion {
			http.Error(w, "task version conflict", http.StatusConflict)
			return
		}

		if completed {
			openSubtaskIDs, listErr := deps.database.ListOpenDirectSubtaskIDs(r.Context(), taskID)
			if listErr != nil {
				http.Error(w, "failed to load subtasks", http.StatusInternalServerError)
				return
			}
			if len(openSubtaskIDs) > 0 {
				switch subtaskAction {
				case "":
					http.Error(w, "open subtasks require confirmation", http.StatusConflict)
					return
				case "cancel":
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte("task completion canceled"))
					return
				case "parent_only":
					break
				case "complete_open":
					for _, subtaskID := range openSubtaskIDs {
						subtaskBase, baseErr := deps.database.LoadTaskUpdateBase(r.Context(), subtaskID, "")
						if baseErr != nil {
							if errors.Is(baseErr, db.ErrTaskNotFound) {
								http.Error(w, "task version conflict", http.StatusConflict)
								return
							}
							http.Error(w, "failed to load subtask", http.StatusInternalServerError)
							return
						}
						if updateErr := performTaskCompletionUpdate(r.Context(), deps, subtaskID, subtaskBase.ExpectedVersion, sessionID, tabID, true, subtaskBase); updateErr != nil {
							writeTaskCompletionError(w, updateErr)
							return
						}
					}
				default:
					http.Error(w, "invalid subtasks_action", http.StatusBadRequest)
					return
				}
			}
		}

		if err := performTaskCompletionUpdate(r.Context(), deps, taskID, expectedVersion, sessionID, tabID, completed, base); err != nil {
			writeTaskCompletionError(w, err)
			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("task completion updated"))
	}
}

func performTaskCompletionUpdate(
	ctx context.Context,
	deps taskUpdateDependencies,
	taskID string,
	expectedVersion int,
	sessionID string,
	tabID string,
	completed bool,
	base db.TaskUpdateInput,
) error {
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
	creds, err := deps.database.LoadCalDAVCredentials(ctx, deps.encryptionKey)
	if err != nil {
		return errTaskCompletionCredentialsUnavailable
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

	prepared, err := deps.database.PrepareTaskUpdate(ctx, input)
	if err != nil {
		if errors.Is(err, db.ErrTaskVersionMismatch) {
			return errTaskCompletionVersionConflict
		}
		return errTaskCompletionPersistPending
	}

	todoCredentials := caldav.Credentials{URL: creds.URL, Username: creds.Username, Password: creds.Password}

	newETag, err := deps.todos.PutVTODOUpdate(ctx, todoCredentials, prepared.PreviousHref, rawVTODO, prepared.PreviousETag)
	if err != nil {
		persistCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), taskUpdatePersistTimeout)
		defer cancel()
		if errors.Is(err, caldav.ErrPreconditionFailed) {
			if markErr := deps.database.MarkTaskUpdateConflict(persistCtx, taskID, prepared.PendingVersion); markErr != nil {
				return errTaskCompletionPersistConflict
			}
			return errTaskCompletionVersionConflict
		}
		if markErr := deps.database.MarkTaskUpdateError(persistCtx, taskID, prepared.PendingVersion); markErr != nil {
			return errTaskCompletionPersistError
		}
		return errTaskCompletionCalDAV
	}

	persistCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), taskUpdatePersistTimeout)
	defer cancel()
	if err := deps.database.MarkTaskUpdateSynced(persistCtx, taskID, prepared.PendingVersion, newETag); err != nil {
		return errTaskCompletionPersistSynced
	}
	return nil
}

var (
	errTaskCompletionVersionConflict        = errors.New("task completion version conflict")
	errTaskCompletionPersistPending         = errors.New("task completion persist pending failed")
	errTaskCompletionPersistConflict        = errors.New("task completion persist conflict failed")
	errTaskCompletionPersistError           = errors.New("task completion persist error failed")
	errTaskCompletionPersistSynced          = errors.New("task completion persist synced failed")
	errTaskCompletionCalDAV                 = errors.New("task completion caldav write failed")
	errTaskCompletionCredentialsUnavailable = errors.New("task completion caldav credentials unavailable")
)

func writeTaskCompletionError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errTaskCompletionVersionConflict):
		http.Error(w, "task version conflict", http.StatusConflict)
	case errors.Is(err, errTaskCompletionPersistPending):
		http.Error(w, "failed to save pending task update", http.StatusInternalServerError)
	case errors.Is(err, errTaskCompletionPersistConflict):
		http.Error(w, "failed to persist task update conflict state", http.StatusInternalServerError)
	case errors.Is(err, errTaskCompletionPersistError):
		http.Error(w, "failed to persist task update error state", http.StatusInternalServerError)
	case errors.Is(err, errTaskCompletionPersistSynced):
		http.Error(w, "failed to persist synced task update", http.StatusInternalServerError)
	case errors.Is(err, errTaskCompletionCalDAV):
		http.Error(w, "failed to update task on caldav server", http.StatusBadGateway)
	case errors.Is(err, errTaskCompletionCredentialsUnavailable):
		http.Error(w, "caldav credentials unavailable", http.StatusFailedDependency)
	default:
		http.Error(w, "failed to update task", http.StatusInternalServerError)
	}
}
