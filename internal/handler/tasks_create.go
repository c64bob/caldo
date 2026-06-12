package handler

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"path"
	"strings"
	"time"

	"caldo/internal/caldav"
	"caldo/internal/db"
	"caldo/internal/model"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type taskCreateTodoClient interface {
	PutVTODOCreate(ctx context.Context, credentials caldav.Credentials, todoHref string, rawVTODO string) (string, error)
}

type taskCreateDependencies struct {
	database      *db.Database
	encryptionKey []byte
	todos         taskCreateTodoClient
	calendar      projectCreateCalendarClient
	broker        *eventBroker
}

const taskCreatePersistTimeout = 5 * time.Second
const quickAddExistingProjectSelectionPrefix = "existing:"

var (
	errQuickAddProjectNameRequired  = errors.New("quick add project name required")
	errQuickAddProjectSelection     = errors.New("quick add project selection required")
	errQuickAddProjectCreateClient  = errors.New("quick add project create client unavailable")
	errQuickAddProjectCreateFailed  = errors.New("quick add project create failed")
	errQuickAddProjectPersistFailed = errors.New("quick add project persist failed")
	errQuickAddRecurrenceInvalid    = errors.New("quick add recurrence invalid")
)

// TaskCreate creates a new task and performs synchronous CalDAV write-through.
func TaskCreate(deps taskCreateDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if strings.TrimSpace(r.FormValue("parent_task_id")) != "" {
			http.Error(w, "subtasks can only be created from the subtask action", http.StatusBadRequest)
			return
		}
		createTask(w, r, deps, "", "")
	}
}

// TaskCreateSubtask creates a direct subtask for a root task.
func TaskCreateSubtask(deps taskCreateDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		parentTaskID := strings.TrimSpace(chi.URLParam(r, "taskID"))
		project, parentUID, err := deps.database.ResolveSubtaskParent(r.Context(), parentTaskID)
		if err != nil {
			statusCode := http.StatusInternalServerError
			message := "failed to resolve subtask parent"
			switch {
			case errors.Is(err, db.ErrSubtaskParentNotFound):
				statusCode = http.StatusNotFound
				message = "parent task not found"
			case errors.Is(err, db.ErrSubtaskParentIsSubtask):
				statusCode = http.StatusConflict
				message = "subtasks cannot have subtasks"
			}
			http.Error(w, message, statusCode)
			return
		}
		createTask(w, r, deps, project.ID, parentUID)
	}
}

func createTask(w http.ResponseWriter, r *http.Request, deps taskCreateDependencies, forcedProjectID string, parentUID string) {
	title := strings.TrimSpace(r.FormValue("title"))
	if title == "" {
		http.Error(w, "title is required", http.StatusBadRequest)
		return
	}
	recurrence, err := parseQuickAddRecurrence(r.FormValue("recurrence"))
	if err != nil {
		http.Error(w, "recurrence is invalid", http.StatusBadRequest)
		return
	}

	project, err := resolveCreateTaskProject(r.Context(), r, deps, forcedProjectID)
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
		case errors.Is(err, errQuickAddProjectNameRequired):
			statusCode = http.StatusBadRequest
			errMessage = "project name is required"
		case errors.Is(err, errQuickAddProjectSelection):
			statusCode = http.StatusBadRequest
			errMessage = "project selection is required"
		case errors.Is(err, errQuickAddProjectCreateFailed):
			statusCode = http.StatusBadGateway
			errMessage = "failed to create project on caldav server"
		case errors.Is(err, errQuickAddProjectPersistFailed):
			statusCode = http.StatusInternalServerError
			errMessage = "failed to persist project"
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
	if strings.TrimSpace(parentUID) != "" {
		linked := strings.TrimSpace(parentUID)
		rawVTODO = strings.Replace(rawVTODO, "\r\nEND:VTODO\r\n", "\r\nRELATED-TO;RELTYPE=PARENT:"+linked+"\r\nEND:VTODO\r\n", 1)
	}
	rawVTODO = model.PatchVTODO(rawVTODO, model.VTODOPatch{
		Priority:   parseQuickAddPriority(r.FormValue("priority")),
		DueDate:    parseOptionalDate(r.FormValue("due_date")),
		Categories: parseQuickAddLabels(r.FormValue("labels")),
		RRule:      recurrence,
	})
	parsed := model.ParseVTODOFields(rawVTODO)

	taskID, err := deps.database.InsertPendingTask(r.Context(), db.NewTaskInput{
		ProjectID:   project.ID,
		ProjectName: project.DisplayName,
		UID:         todoUID,
		Href:        todoHref,
		Title:       parsed.Title,
		Description: parsed.Description,
		DueDate:     nullableDate(parsed.DueDate),
		DueAt:       nullableTime(parsed.DueAt),
		Priority:    nullableInt(parsed.Priority),
		RRule:       parsed.RRule,
		LabelNames:  nullableCSV(parsed.Categories),
		RawVTODO:    rawVTODO,
		ParentID:    firstNonEmpty(strings.TrimSpace(chi.URLParam(r, "taskID")), ""),
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

func resolveCreateTaskProject(ctx context.Context, r *http.Request, deps taskCreateDependencies, forcedProjectID string) (db.TaskProject, error) {
	if strings.TrimSpace(forcedProjectID) != "" {
		return deps.database.ResolveTaskProject(ctx, forcedProjectID)
	}
	projectSelection := strings.TrimSpace(r.FormValue("project_selection"))
	if projectSelection == "create" {
		return createQuickAddProject(ctx, r, deps)
	}
	if strings.HasPrefix(projectSelection, quickAddExistingProjectSelectionPrefix) {
		projectID := strings.TrimSpace(strings.TrimPrefix(projectSelection, quickAddExistingProjectSelectionPrefix))
		if projectID == "" {
			return db.TaskProject{}, db.ErrTaskProjectNotFound
		}
		return deps.database.ResolveTaskProject(ctx, projectID)
	}
	if shouldCreateQuickAddProject(r) {
		return createQuickAddProject(ctx, r, deps)
	}
	if strings.TrimSpace(r.FormValue("project_new_name")) != "" {
		return db.TaskProject{}, errQuickAddProjectSelection
	}
	return deps.database.ResolveTaskProject(ctx, r.FormValue("project_id"))
}

func shouldCreateQuickAddProject(r *http.Request) bool {
	return strings.EqualFold(strings.TrimSpace(r.FormValue("create_project")), "1") ||
		strings.EqualFold(strings.TrimSpace(r.FormValue("create_project")), "true")
}

func createQuickAddProject(ctx context.Context, r *http.Request, deps taskCreateDependencies) (db.TaskProject, error) {
	projectName := strings.TrimSpace(r.FormValue("project_new_name"))
	if projectName == "" {
		return db.TaskProject{}, errQuickAddProjectNameRequired
	}

	if existing, err := deps.database.LoadProjectByName(ctx, projectName); err == nil {
		return existing, nil
	} else if !errors.Is(err, sql.ErrNoRows) {
		return db.TaskProject{}, fmt.Errorf("create quick add project: load existing project: %w", err)
	}

	if deps.calendar == nil {
		return db.TaskProject{}, errQuickAddProjectCreateClient
	}

	credentials, err := deps.database.LoadCalDAVCredentials(ctx, deps.encryptionKey)
	if err != nil {
		return db.TaskProject{}, fmt.Errorf("create quick add project: load caldav credentials: %w", err)
	}
	capabilities, err := deps.database.LoadCalDAVServerCapabilities(ctx)
	if err != nil {
		return db.TaskProject{}, fmt.Errorf("create quick add project: load caldav capabilities: %w", err)
	}
	createdCalendar, err := deps.calendar.CreateCalendar(ctx, caldav.Credentials{
		URL:      credentials.URL,
		Username: credentials.Username,
		Password: credentials.Password,
	}, projectName)
	if err != nil {
		return db.TaskProject{}, fmt.Errorf("%w: %w", errQuickAddProjectCreateFailed, err)
	}

	persistCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), projectCreatePersistTimeout)
	defer cancel()
	project, err := deps.database.InsertProject(persistCtx, db.NewProjectInput{
		CalendarHref: createdCalendar.Href,
		DisplayName:  createdCalendar.DisplayName,
		SyncStrategy: initialSyncStrategy(capabilities),
	})
	if err != nil {
		return db.TaskProject{}, fmt.Errorf("%w: %w", errQuickAddProjectPersistFailed, err)
	}

	if deps.broker != nil {
		deps.broker.publish(appEvent{Type: "project", Resource: project.ID, Version: 1, OriginConnection: strings.TrimSpace(r.Header.Get("X-Tab-ID"))})
	}

	return db.TaskProject{ID: project.ID, CalendarHref: project.CalendarHref, DisplayName: project.DisplayName}, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
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

func parseQuickAddRecurrence(value string) (*string, error) {
	recurrence := strings.TrimSpace(value)
	if recurrence == "" {
		return nil, nil
	}
	if !validQuickAddRecurrence(recurrence) {
		return nil, errQuickAddRecurrenceInvalid
	}
	return &recurrence, nil
}

func validQuickAddRecurrence(recurrence string) bool {
	if strings.ContainsAny(recurrence, "\r\n") {
		return false
	}

	parts := strings.Split(recurrence, ";")
	hasFreq := false
	seen := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			return false
		}
		key, value, ok := strings.Cut(part, "=")
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if !ok || key == "" || value == "" {
			return false
		}
		normalizedKey := strings.ToUpper(key)
		if _, ok := seen[normalizedKey]; ok {
			return false
		}
		seen[normalizedKey] = struct{}{}
		if !validQuickAddRecurrenceKey(key) || strings.ContainsAny(value, "\r\n:;") {
			return false
		}
		if normalizedKey == "FREQ" {
			hasFreq = true
		}
	}
	return hasFreq
}

func validQuickAddRecurrenceKey(key string) bool {
	for _, r := range key {
		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			continue
		}
		return false
	}
	return true
}

func taskHref(calendarHref string, uid string) string {
	trimmed := strings.TrimSpace(calendarHref)
	if strings.HasSuffix(trimmed, "/") {
		return trimmed + uid + ".ics"
	}

	return path.Clean(fmt.Sprintf("%s/%s.ics", trimmed, uid))
}
