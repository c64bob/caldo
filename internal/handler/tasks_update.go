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
		handleTaskUpdate(w, r, deps, false)
	}
}

// TaskLabels updates the labels of an existing task and performs synchronous CalDAV write-through.
func TaskLabels(deps taskUpdateDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		handleTaskUpdate(w, r, deps, true)
	}
}

func handleTaskUpdate(w http.ResponseWriter, r *http.Request, deps taskUpdateDependencies, requireLabels bool) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form payload", http.StatusBadRequest)
		return
	}
	if requireLabels {
		if _, ok := r.PostForm["labels"]; !ok {
			http.Error(w, "labels are required", http.StatusBadRequest)
			return
		}
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

	baseFields := model.ParseVTODOFields(base.RawVTODO)

	title := strings.TrimSpace(r.FormValue("title"))
	if title == "" {
		title = baseFields.Title
	}
	if title == "" {
		http.Error(w, "title is required", http.StatusBadRequest)
		return
	}

	_, statusProvided := r.PostForm["status"]
	status := strings.TrimSpace(strings.ToLower(r.FormValue("status")))
	if status == "" {
		status = baseFields.Status
	}
	if status == "" {
		status = "needs-action"
	}

	var description *string
	if _, ok := r.PostForm["description"]; ok {
		description = stringPointer(strings.TrimSpace(r.FormValue("description")))
	}

	existingRRule := baseFields.RRule

	patch := model.VTODOPatch{
		Summary:     &title,
		Description: description,
		Status:      &status,
	}
	applyDuePatch(r.PostForm, baseFields, &patch)
	applyPriorityPatch(r.PostForm, &patch)
	applyLabelPatch(r.PostForm, baseFields.Categories, &patch)
	if !model.IsComplexRRule(existingRRule) {
		if recurrence := buildExplicitRRuleUpdate(r.PostForm); recurrence != nil {
			if *recurrence != strings.TrimSpace(existingRRule) {
				patch.RRule = recurrence
			}
		}
	}
	if statusProvided && status == "completed" {
		now := time.Now().UTC()
		patch.CompletedAt = &now
	} else if statusProvided {
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
		CompletedAt:     nullableTime(parsed.CompletedAt),
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

	creds, err := deps.database.LoadCalDAVCredentials(r.Context(), deps.encryptionKey)
	if err != nil {
		persistCtx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), taskUpdatePersistTimeout)
		defer cancel()
		if markErr := deps.database.MarkTaskUpdateError(persistCtx, taskID, prepared.PendingVersion); markErr != nil {
			http.Error(w, "failed to persist task update error state", http.StatusInternalServerError)
			return
		}
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
		deps.broker.publish(appEvent{Type: "task", Resource: taskID, Version: prepared.PendingVersion + 1, OriginConnection: tabID})
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("task updated"))
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

func applyDuePatch(form map[string][]string, baseFields model.VTODOFields, patch *model.VTODOPatch) {
	if _, ok := form["due_at"]; ok {
		dueAtRaw := strings.TrimSpace(firstFormValue(form, "due_at"))
		if dueAtRaw == "" {
			patch.ClearDue = true
			return
		}
		patch.DueAt = parseOptionalDateTime(dueAtRaw)
		return
	}

	if _, ok := form["due_date"]; !ok {
		return
	}
	dueDateRaw := strings.TrimSpace(firstFormValue(form, "due_date"))
	if dueDateRaw == "" {
		patch.ClearDue = true
		return
	}
	parsedDate := parseOptionalDate(dueDateRaw)
	if parsedDate == nil {
		return
	}
	if baseFields.DueAt != nil {
		if date, err := time.Parse("2006-01-02", *parsedDate); err == nil {
			existingTime := baseFields.DueAt.UTC()
			combined := time.Date(date.Year(), date.Month(), date.Day(), existingTime.Hour(), existingTime.Minute(), existingTime.Second(), existingTime.Nanosecond(), time.UTC)
			patch.DueAt = &combined
			return
		}
	}
	patch.DueDate = parsedDate
}

func applyPriorityPatch(form map[string][]string, patch *model.VTODOPatch) {
	if _, ok := form["priority"]; !ok {
		return
	}
	priorityRaw := strings.TrimSpace(firstFormValue(form, "priority"))
	if priorityRaw == "" {
		patch.ClearPriority = true
		return
	}
	patch.Priority = parseOptionalInt(priorityRaw)
}

func applyLabelPatch(form map[string][]string, existingCategories []string, patch *model.VTODOPatch) {
	if _, ok := form["labels"]; !ok {
		return
	}
	_, isFavorite := model.CategoriesToLabelsAndFavorite(existingCategories)
	categories, err := model.LabelsAndFavoriteToCategories(parseLabels(firstFormValue(form, "labels")), isFavorite)
	if err != nil {
		return
	}
	patch.Categories = categories
}

func parseLabels(raw string) []string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return []string{}
	}
	parts := strings.Split(trimmed, ",")
	labels := make([]string, 0, len(parts))
	for _, part := range parts {
		label := strings.TrimSpace(part)
		if label != "" {
			labels = append(labels, label)
		}
	}
	return labels
}

func parseOptionalLabels(raw string) []string {
	labels := parseLabels(raw)
	if len(labels) == 0 {
		return nil
	}
	return labels
}

func buildExplicitRRuleUpdate(form map[string][]string) *string {
	if _, ok := form["repeat_update"]; !ok {
		return nil
	}

	freq := strings.ToUpper(strings.TrimSpace(firstFormValue(form, "repeat_freq")))
	if freq == "" || freq == "NONE" {
		empty := ""
		return &empty
	}

	ruleParts := []string{}
	interval := strings.TrimSpace(firstFormValue(form, "repeat_interval"))
	if interval == "" {
		interval = "1"
	}

	switch freq {
	case "DAILY", "WEEKLY", "MONTHLY", "YEARLY":
		ruleParts = append(ruleParts, "FREQ="+freq)
	case "WEEKDAYS":
		ruleParts = append(ruleParts, "FREQ=WEEKLY", "BYDAY=MO,TU,WE,TH,FR")
	case "BYDAY":
		day := strings.ToUpper(strings.TrimSpace(firstFormValue(form, "repeat_byday")))
		if !isValidByDay(day) {
			return nil
		}
		ruleParts = append(ruleParts, "FREQ=WEEKLY", "BYDAY="+day)
	default:
		return nil
	}

	if interval != "1" {
		if n, err := strconv.Atoi(interval); err == nil && n > 0 {
			ruleParts = append(ruleParts, "INTERVAL="+strconv.Itoa(n))
		}
	}

	endType := strings.ToLower(strings.TrimSpace(firstFormValue(form, "repeat_end")))
	switch endType {
	case "", "never":
	case "until":
		if until := parseOptionalDate(firstFormValue(form, "repeat_until")); until != nil {
			parsed, _ := time.Parse("2006-01-02", *until)
			untilEndOfDayUTC := parsed.UTC().Add(24*time.Hour - time.Second)
			ruleParts = append(ruleParts, "UNTIL="+untilEndOfDayUTC.Format("20060102T150405Z"))
		}
	case "count":
		if n, err := strconv.Atoi(strings.TrimSpace(firstFormValue(form, "repeat_count"))); err == nil && n > 0 {
			ruleParts = append(ruleParts, "COUNT="+strconv.Itoa(n))
		}
	}

	rule := strings.Join(ruleParts, ";")
	return &rule
}

func firstFormValue(form map[string][]string, key string) string {
	values := form[key]
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func isValidByDay(value string) bool {
	switch value {
	case "MO", "TU", "WE", "TH", "FR", "SA", "SU":
		return true
	default:
		return false
	}
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
