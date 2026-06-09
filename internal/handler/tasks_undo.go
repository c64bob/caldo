package handler

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"caldo/internal/caldav"
	"caldo/internal/db"
	"caldo/internal/view"
)

const taskUndoPersistTimeout = 5 * time.Second

// TaskUndoStatus renders the current tab's available undo action.
func TaskUndoStatus(database *db.Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tabID := strings.TrimSpace(r.Header.Get("X-Tab-ID"))
		if tabID == "" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		sessionID := undoSessionID(r)

		status, err := database.LoadUndoSnapshotStatus(r.Context(), sessionID, tabID)
		if err != nil {
			if errors.Is(err, db.ErrUndoSnapshotNotFound) {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			http.Error(w, "failed to load undo status", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := view.UndoNotification(view.UndoSnapshotView{ActionType: status.ActionType, ExpiresAtISO: status.ExpiresAtISO}).Render(r.Context(), w); err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
	}
}

// TaskUndo executes undo for the latest session/tab snapshot via synchronous CalDAV write.
func TaskUndo(deps taskUpdateDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tabID := strings.TrimSpace(r.Header.Get("X-Tab-ID"))
		if tabID == "" {
			http.Error(w, "X-Tab-ID header is required", http.StatusBadRequest)
			return
		}
		sessionID := undoSessionID(r)

		prepared, err := deps.database.PrepareTaskUndo(r.Context(), sessionID, tabID)
		if err != nil {
			switch {
			case errors.Is(err, db.ErrUndoSnapshotNotFound):
				http.Error(w, "undo snapshot not found", http.StatusNotFound)
			case errors.Is(err, db.ErrUndoSnapshotExpired):
				http.Error(w, "undo snapshot expired", http.StatusConflict)
			case errors.Is(err, db.ErrUndoETagMismatch):
				http.Error(w, "task version conflict", http.StatusConflict)
			case errors.Is(err, db.ErrUndoActionNotSupported):
				http.Error(w, "undo action not supported", http.StatusBadRequest)
			case errors.Is(err, db.ErrTaskNotFound):
				http.Error(w, "task not found", http.StatusNotFound)
			default:
				http.Error(w, "failed to prepare undo", http.StatusInternalServerError)
			}
			return
		}

		creds, err := deps.database.LoadCalDAVCredentials(r.Context(), deps.encryptionKey)
		if err != nil {
			http.Error(w, "caldav credentials unavailable", http.StatusFailedDependency)
			return
		}
		todoCredentials := caldav.Credentials{URL: creds.URL, Username: creds.Username, Password: creds.Password}

		var newETag string
		if prepared.ActionType == "task_deleted" {
			newETag, err = deps.todos.PutVTODOCreate(r.Context(), todoCredentials, prepared.TodoHref, prepared.RawVTODO)
		} else {
			newETag, err = deps.todos.PutVTODOUpdate(r.Context(), todoCredentials, prepared.TodoHref, prepared.RawVTODO, prepared.ExpectedETag)
		}
		if err != nil {
			persistCtx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), taskUndoPersistTimeout)
			defer cancel()
			if errors.Is(err, caldav.ErrPreconditionFailed) && prepared.ActionType != "task_deleted" {
				if markErr := deps.database.MarkTaskUpdateConflict(persistCtx, prepared.TaskID, prepared.PendingVersion); markErr != nil {
					http.Error(w, "failed to persist undo conflict state", http.StatusInternalServerError)
					return
				}
				http.Error(w, "task version conflict", http.StatusConflict)
				return
			}
			if prepared.ActionType == "task_deleted" {
				if markErr := deps.database.MarkTaskCreateError(persistCtx, prepared.TaskID); markErr != nil {
					http.Error(w, "failed to persist undo error state", http.StatusInternalServerError)
					return
				}
			} else {
				if markErr := deps.database.MarkTaskUpdateError(persistCtx, prepared.TaskID, prepared.PendingVersion); markErr != nil {
					http.Error(w, "failed to persist undo error state", http.StatusInternalServerError)
					return
				}
			}
			http.Error(w, "failed to execute undo on caldav server", http.StatusBadGateway)
			return
		}

		persistCtx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), taskUndoPersistTimeout)
		defer cancel()
		if prepared.ActionType == "task_deleted" {
			if _, err := deps.database.MarkTaskCreateSynced(persistCtx, prepared.TaskID, newETag); err != nil {
				http.Error(w, "failed to persist synced undo", http.StatusInternalServerError)
				return
			}
		} else {
			if err := deps.database.MarkTaskUpdateSynced(persistCtx, prepared.TaskID, prepared.PendingVersion, newETag); err != nil {
				http.Error(w, "failed to persist synced undo", http.StatusInternalServerError)
				return
			}
		}
		if err := deps.database.DeleteUndoSnapshot(persistCtx, prepared.SnapshotID); err != nil {
			http.Error(w, "failed to delete undo snapshot", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := view.UndoResultNotification("Rückgängig ausgeführt.").Render(r.Context(), w); err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
	}
}

func undoSessionID(r *http.Request) string {
	sessionID := strings.TrimSpace(r.Header.Get("X-Forwarded-User"))
	if sessionID == "" {
		sessionID = "single-user-session"
	}
	return sessionID
}
