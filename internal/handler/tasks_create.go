package handler

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"path"
	"strings"
	"time"

	"caldo/internal/caldav"
	"caldo/internal/db"
	"caldo/internal/model"
	"github.com/google/uuid"
)

type taskCreateTodoClient interface {
	PutVTODOCreate(ctx context.Context, credentials caldav.Credentials, todoHref string, rawVTODO string) (string, error)
}

type taskCreateDependencies struct {
	database      *db.Database
	encryptionKey []byte
	todos         taskCreateTodoClient
	broker        *eventBroker
}

const taskCreatePersistTimeout = 5 * time.Second

// TaskCreate creates a new task and performs synchronous CalDAV write-through.
func TaskCreate(deps taskCreateDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		title := strings.TrimSpace(r.FormValue("title"))
		if title == "" {
			http.Error(w, "title is required", http.StatusBadRequest)
			return
		}

		project, err := deps.database.ResolveTaskProject(r.Context(), r.FormValue("project_id"))
		if err != nil {
			statusCode := http.StatusInternalServerError
			errMessage := "failed to resolve project"
			switch {
			case errors.Is(err, db.ErrTaskProjectNotFound):
				statusCode = http.StatusBadRequest
				errMessage = "selected project does not exist"
			case errors.Is(err, db.ErrTaskProjectUnavailable):
				statusCode = http.StatusConflict
				errMessage = "no valid default project configured"
			}
			http.Error(w, errMessage, statusCode)
			return
		}

		creds, err := deps.database.LoadCalDAVCredentials(r.Context(), deps.encryptionKey)
		if err != nil {
			http.Error(w, "caldav credentials unavailable", http.StatusFailedDependency)
			return
		}

		todoUID := uuid.NewString()
		todoHref := taskHref(project.CalendarHref, todoUID)
		rawVTODO := model.BuildTaskVTODO(todoUID, title, time.Now().UTC())
		rawVTODO = model.PatchVTODO(rawVTODO, model.VTODOPatch{
			Priority:   parseQuickAddPriority(r.FormValue("priority")),
			Categories: parseQuickAddLabels(r.FormValue("labels")),
			RRule:      parseQuickAddRecurrence(r.FormValue("recurrence")),
		})

		taskID, err := deps.database.InsertPendingTask(r.Context(), db.NewTaskInput{
			ProjectID:   project.ID,
			ProjectName: project.DisplayName,
			UID:         todoUID,
			Href:        todoHref,
			Title:       title,
			RawVTODO:    rawVTODO,
		})
		if err != nil {
			http.Error(w, "failed to prepare local task", http.StatusInternalServerError)
			return
		}

		todoClientCredentials := caldav.Credentials{
			URL:      creds.URL,
			Username: creds.Username,
			Password: creds.Password,
		}
		etag, err := deps.todos.PutVTODOCreate(r.Context(), todoClientCredentials, todoHref, rawVTODO)
		if err != nil {
			persistCtx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), taskCreatePersistTimeout)
			defer cancel()
			if markErr := deps.database.MarkTaskCreateError(persistCtx, taskID); markErr != nil {
				http.Error(w, "failed to persist create error state", http.StatusInternalServerError)
				return
			}
			http.Error(w, "failed to create task on caldav server", http.StatusBadGateway)
			return
		}

		persistCtx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), taskCreatePersistTimeout)
		defer cancel()
		serverVersion, err := deps.database.MarkTaskCreateSynced(persistCtx, taskID, etag)
		if err != nil {
			http.Error(w, "failed to persist synced task", http.StatusInternalServerError)
			return
		}

		if deps.broker != nil {
			deps.broker.publish(appEvent{Type: "task", Resource: taskID, Version: serverVersion, OriginConnection: strings.TrimSpace(r.Header.Get("X-Tab-ID"))})
		}

		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("task created"))
	}
}

func parseQuickAddPriority(value string) *int {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "high":
		priority := 1
		return &priority
	case "medium":
		priority := 5
		return &priority
	case "low":
		priority := 9
		return &priority
	default:
		return nil
	}
}

func parseQuickAddLabels(value string) []string {
	parts := strings.Split(value, ",")
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

func parseQuickAddRecurrence(value string) *string {
	recurrence := strings.TrimSpace(value)
	if recurrence == "" {
		return nil
	}
	return &recurrence
}

func taskHref(calendarHref string, uid string) string {
	trimmed := strings.TrimSpace(calendarHref)
	if strings.HasSuffix(trimmed, "/") {
		return trimmed + uid + ".ics"
	}

	return path.Clean(fmt.Sprintf("%s/%s.ics", trimmed, uid))
}
