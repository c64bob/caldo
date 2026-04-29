package handler

import (
	"context"
	"database/sql"
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

type taskUpdateTodoClient interface {
	PutVTODOUpdate(ctx context.Context, credentials caldav.Credentials, todoHref string, rawVTODO string, etag string) (string, error)
	PutVTODOCreate(ctx context.Context, credentials caldav.Credentials, todoHref string, rawVTODO string) (string, error)
	DeleteVTODO(ctx context.Context, credentials caldav.Credentials, todoHref string, etag string) error
}

type taskUpdateDependencies struct {
	database      *db.Database
	encryptionKey []byte
	todos         taskUpdateTodoClient
	broker        *eventBroker
}

const taskUpdatePersistTimeout = 5 * time.Second

// TaskUpdate updates an existing task and performs synchronous CalDAV write-through.
func TaskUpdate(deps taskUpdateDependencies) http.HandlerFunc {
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

		base, err := deps.database.LoadTaskUpdateBase(r.Context(), taskID, r.FormValue("project_id"))
		if err != nil {
			switch {
			case errors.Is(err, db.ErrTaskNotFound):
				http.Error(w, "task not found", http.StatusNotFound)
			case errors.Is(err, db.ErrTaskProjectNotFound):
				http.Error(w, "selected project does not exist", http.StatusBadRequest)
			default:
				http.Error(w, "failed to load task", http.StatusInternalServerError)
			}
			return
		}

		title := strings.TrimSpace(r.FormValue("title"))
		if title == "" {
			title = model.ParseVTODOFields(base.RawVTODO).Title
		}
		if title == "" {
			http.Error(w, "title is required", http.StatusBadRequest)
			return
		}

		status := strings.TrimSpace(strings.ToLower(r.FormValue("status")))
		if status == "" {
			status = model.ParseVTODOFields(base.RawVTODO).Status
		}
		if status == "" {
			status = "needs-action"
		}

		var description *string
		if _, ok := r.PostForm["description"]; ok {
			description = stringPointer(strings.TrimSpace(r.FormValue("description")))
		}

		patch := model.VTODOPatch{
			Summary:     &title,
			Description: description,
			Status:      &status,
			DueDate:     parseOptionalDate(r.FormValue("due_date")),
			DueAt:       parseOptionalDateTime(r.FormValue("due_at")),
			Categories:  parseOptionalLabels(r.FormValue("labels")),
			Priority:    parseOptionalInt(r.FormValue("priority")),
		}
		if status == "completed" {
			now := time.Now().UTC()
			patch.CompletedAt = &now
		} else {
			patch.ClearCompleted = true
		}

		rawVTODO := model.PatchVTODO(base.RawVTODO, patch)
		parsed := model.ParseVTODOFields(rawVTODO)

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
				if deps.broker != nil {
					deps.broker.publish(appEvent{Type: "conflict", Resource: taskID, Version: prepared.PendingVersion, OriginConnection: tabID})
				}
				http.Error(w, "task version conflict", http.StatusConflict)
				return
			}
			http.Error(w, "failed to save pending task update", http.StatusInternalServerError)
			return
		}

		creds, err := deps.database.LoadCalDAVCredentials(r.Context(), deps.encryptionKey)
		if err != nil {
			http.Error(w, "caldav credentials unavailable", http.StatusFailedDependency)
			return
		}
		todoCredentials := caldav.Credentials{URL: creds.URL, Username: creds.Username, Password: creds.Password}

		var newETag string
		if prepared.ProjectChanged {
			newETag, err = deps.todos.PutVTODOCreate(r.Context(), todoCredentials, prepared.NextHref, rawVTODO)
			if err == nil {
				err = deps.todos.DeleteVTODO(r.Context(), todoCredentials, prepared.PreviousHref, prepared.PreviousETag)
				if err != nil {
					persistCtx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), taskUpdatePersistTimeout)
					defer cancel()
					if markErr := deps.database.MarkTaskUpdateErrorWithETag(persistCtx, taskID, prepared.PendingVersion, newETag); markErr != nil {
						http.Error(w, "failed to persist task update error state", http.StatusInternalServerError)
						return
					}
					http.Error(w, "failed to finalize task move on caldav server", http.StatusBadGateway)
					return
				}
			}
		} else {
			newETag, err = deps.todos.PutVTODOUpdate(r.Context(), todoCredentials, prepared.PreviousHref, rawVTODO, prepared.PreviousETag)
		}
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

		if deps.broker != nil {
			deps.broker.publish(appEvent{Type: "task", Resource: taskID, Version: prepared.PendingVersion, OriginConnection: tabID})
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("task updated"))
	}
}

func parseOptionalInt(raw string) *int {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil
	}
	parsed, err := strconv.Atoi(trimmed)
	if err != nil {
		return nil
	}
	return &parsed
}

func parseOptionalDate(raw string) *string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil
	}
	if _, err := time.Parse("2006-01-02", trimmed); err != nil {
		return nil
	}
	return &trimmed
}

func parseOptionalDateTime(raw string) *time.Time {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil
	}
	parsed, err := time.Parse(time.RFC3339, trimmed)
	if err != nil {
		return nil
	}
	utc := parsed.UTC()
	return &utc
}

func parseOptionalLabels(raw string) []string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil
	}
	parts := strings.Split(trimmed, ",")
	labels := make([]string, 0, len(parts))
	for _, part := range parts {
		label := strings.TrimSpace(part)
		if label != "" {
			labels = append(labels, label)
		}
	}
	if len(labels) == 0 {
		return nil
	}
	return labels
}

func stringPointer(value string) *string {
	return &value
}

func nullableDate(value *string) sql.NullString {
	if value == nil || strings.TrimSpace(*value) == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: strings.TrimSpace(*value), Valid: true}
}

func nullableCSV(values []string) sql.NullString {
	if len(values) == 0 {
		return sql.NullString{}
	}
	return sql.NullString{String: strings.Join(values, ","), Valid: true}
}

func nullableTime(value *time.Time) sql.NullTime {
	if value == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: value.UTC(), Valid: true}
}

func nullableInt(value *int) sql.NullInt64 {
	if value == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: int64(*value), Valid: true}
}
