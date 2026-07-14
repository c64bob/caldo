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

	"caldo/internal/caldav"
	"caldo/internal/db"
	"caldo/internal/model"
	"github.com/go-chi/chi/v5"
)

type labelUpdateDependencies struct {
	database      *db.Database
	encryptionKey []byte
	todos         taskUpdateTodoClient
	broker        *eventBroker
}

type labelsPageState struct {
	PageSuccess   string
	RenameLabelID string
	RenameError   string
	RenameValue   string
	DeleteLabelID string
	DeleteError   string
}

type labelTaskWriteError struct {
	completed int
	total     int
	err       error
}

func (e *labelTaskWriteError) Error() string {
	return fmt.Sprintf("label task write failed after %d of %d tasks: %v", e.completed, e.total, e.err)
}

func (e *labelTaskWriteError) Unwrap() error {
	return e.err
}

var (
	errLabelNameRequired       = errors.New("label name required")
	errLabelNameInvalid        = errors.New("label name invalid")
	errLabelNameReserved       = errors.New("label name reserved")
	errLabelConfirmationFailed = errors.New("label confirmation failed")
	errLabelTabIDRequired      = errors.New("label tab id required")
	errLabelCalDAVUnavailable  = errors.New("label caldav unavailable")
)

// LabelRename renames a label across all affected tasks using CalDAV write-through.
func LabelRename(deps labelUpdateDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			renderLabelsPage(w, r, deps.database, labelsPageState{RenameError: "ungültige eingabe"}, http.StatusOK)
			return
		}

		labelID := strings.TrimSpace(chi.URLParam(r, "labelID"))
		newName := strings.TrimSpace(r.FormValue("name"))
		if err := renameLabel(r.Context(), r, deps, labelID, newName); err != nil {
			renderLabelsPage(w, r, deps.database, labelsPageState{
				RenameLabelID: labelID,
				RenameError:   labelMutationErrorMessage("umbenannt", err),
				RenameValue:   newName,
			}, http.StatusOK)
			return
		}

		renderLabelsPage(w, r, deps.database, labelsPageState{PageSuccess: "label wurde umbenannt"}, http.StatusOK)
	}
}

// LabelDelete removes a label from all affected tasks using CalDAV write-through.
func LabelDelete(deps labelUpdateDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		labelID := strings.TrimSpace(chi.URLParam(r, "labelID"))
		formValues, ok := parseLabelDeleteFormValues(w, r, deps.database, labelID)
		if !ok {
			return
		}

		confirmed := formValues.Get("confirmed") == "true"
		if err := deleteLabel(r.Context(), r, deps, labelID, confirmed); err != nil {
			renderLabelsPage(w, r, deps.database, labelsPageState{
				DeleteLabelID: labelID,
				DeleteError:   labelMutationErrorMessage("gelöscht", err),
			}, http.StatusOK)
			return
		}

		renderLabelsPage(w, r, deps.database, labelsPageState{PageSuccess: "label wurde gelöscht"}, http.StatusOK)
	}
}

func parseLabelDeleteFormValues(w http.ResponseWriter, r *http.Request, database *db.Database, labelID string) (url.Values, bool) {
	formValues := r.URL.Query()
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/x-www-form-urlencoded") {
		return formValues, true
	}

	bodyBytes, err := io.ReadAll(io.LimitReader(r.Body, 8*1024))
	if err != nil {
		renderLabelsPage(w, r, database, labelsPageState{
			DeleteLabelID: labelID,
			DeleteError:   "ungültige eingabe",
		}, http.StatusOK)
		return nil, false
	}
	if len(bodyBytes) == 0 {
		return formValues, true
	}

	parsed, err := url.ParseQuery(string(bodyBytes))
	if err != nil {
		renderLabelsPage(w, r, database, labelsPageState{
			DeleteLabelID: labelID,
			DeleteError:   "ungültige eingabe",
		}, http.StatusOK)
		return nil, false
	}
	for key, values := range parsed {
		for _, value := range values {
			formValues.Add(key, value)
		}
	}
	return formValues, true
}

func renameLabel(ctx context.Context, r *http.Request, deps labelUpdateDependencies, labelID string, newName string) error {
	normalizedName, err := normalizeManagedLabelName(newName)
	if err != nil {
		return err
	}

	label, tasks, err := loadLabelMutationBase(ctx, deps.database, labelID)
	if err != nil {
		return err
	}
	if len(tasks) == 0 {
		return deps.database.RenameUnusedLabel(ctx, label.ID, normalizedName)
	}

	if err := writeLabelTaskUpdates(ctx, r, deps, tasks, func(categories []string) ([]string, error) {
		return renameCategoriesLabel(categories, label.Name, normalizedName)
	}); err != nil {
		return err
	}

	if strings.EqualFold(label.Name, normalizedName) {
		return deps.database.RenameLabelRow(ctx, label.ID, normalizedName)
	}
	return deps.database.DeleteUnusedLabel(ctx, label.ID)
}

func deleteLabel(ctx context.Context, r *http.Request, deps labelUpdateDependencies, labelID string, confirmed bool) error {
	label, tasks, err := loadLabelMutationBase(ctx, deps.database, labelID)
	if err != nil {
		return err
	}
	if !confirmed {
		return errLabelConfirmationFailed
	}
	if len(tasks) == 0 {
		return deps.database.DeleteUnusedLabel(ctx, label.ID)
	}

	if err := writeLabelTaskUpdates(ctx, r, deps, tasks, func(categories []string) ([]string, error) {
		return deleteCategoriesLabel(categories, label.Name)
	}); err != nil {
		return err
	}
	return deps.database.DeleteUnusedLabel(ctx, label.ID)
}

func loadLabelMutationBase(ctx context.Context, database *db.Database, labelID string) (db.LabelDetail, []db.LabelMutationTask, error) {
	if database == nil {
		return db.LabelDetail{}, nil, errors.New("label database unavailable")
	}
	label, err := database.LoadLabelDetail(ctx, labelID)
	if err != nil {
		return db.LabelDetail{}, nil, err
	}
	tasks, err := database.ListLabelMutationTasks(ctx, label.ID)
	if err != nil {
		return db.LabelDetail{}, nil, err
	}
	return label, tasks, nil
}

func writeLabelTaskUpdates(ctx context.Context, r *http.Request, deps labelUpdateDependencies, tasks []db.LabelMutationTask, transform func([]string) ([]string, error)) error {
	if deps.database == nil || deps.todos == nil {
		return errLabelCalDAVUnavailable
	}
	tabID := strings.TrimSpace(r.Header.Get("X-Tab-ID"))
	if tabID == "" {
		return errLabelTabIDRequired
	}
	sessionID := requestSessionID(r)

	credentials, err := deps.database.LoadCalDAVCredentials(ctx, deps.encryptionKey)
	if err != nil {
		return errLabelCalDAVUnavailable
	}
	todoCredentials := caldav.Credentials{URL: credentials.URL, Username: credentials.Username, Password: credentials.Password}

	for index, task := range tasks {
		rawVTODO, parsed, err := labelTaskPatchedVTODO(task.RawVTODO, transform)
		if err != nil {
			return err
		}

		input := db.TaskUpdateInput{
			TaskID:          task.ID,
			ExpectedVersion: task.ServerVersion,
			SessionID:       sessionID,
			TabID:           tabID,
			ProjectID:       task.ProjectID,
			ProjectName:     task.ProjectName,
			Href:            task.Href,
			ETag:            task.ETag,
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

		prepared, err := deps.database.PrepareTaskUpdate(ctx, input)
		if err != nil {
			return err
		}

		newETag, err := deps.todos.PutVTODOUpdate(ctx, todoCredentials, prepared.PreviousHref, rawVTODO, prepared.PreviousETag)
		if err != nil {
			persistCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), taskUpdatePersistTimeout)
			if errors.Is(err, caldav.ErrPreconditionFailed) {
				if markErr := markTaskUpdateConflict(persistCtx, taskUpdateDependencies{database: deps.database, todos: deps.todos}, todoCredentials, task.ID, prepared.PendingVersion, prepared.PreviousHref, task.RawVTODO, rawVTODO); markErr != nil {
					cancel()
					return markErr
				}
			} else if markErr := deps.database.MarkTaskUpdateError(persistCtx, task.ID, prepared.PendingVersion); markErr != nil {
				cancel()
				return markErr
			}
			cancel()
			return &labelTaskWriteError{completed: index, total: len(tasks), err: err}
		}

		persistCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), taskUpdatePersistTimeout)
		if err := deps.database.MarkTaskUpdateSynced(persistCtx, task.ID, prepared.PendingVersion, newETag); err != nil {
			cancel()
			return err
		}
		cancel()

		if deps.broker != nil {
			deps.broker.publish(appEvent{Type: "task", Resource: task.ID, Version: prepared.PendingVersion + 1, OriginConnection: tabID})
		}
	}

	return nil
}

func labelTaskPatchedVTODO(rawVTODO string, transform func([]string) ([]string, error)) (string, model.VTODOFields, error) {
	fields := model.ParseVTODOFields(rawVTODO)
	categories, err := transform(fields.Categories)
	if err != nil {
		return "", model.VTODOFields{}, err
	}
	rawVTODO = model.PatchVTODO(rawVTODO, model.VTODOPatch{Categories: categories})
	parsed := model.ParseVTODOFields(rawVTODO)
	return rawVTODO, parsed, nil
}

func renameCategoriesLabel(categories []string, oldName string, newName string) ([]string, error) {
	labels, favorite := model.CategoriesToLabelsAndFavorite(categories)
	renamed := make([]string, 0, len(labels))
	for _, label := range labels {
		if strings.EqualFold(label, oldName) {
			renamed = append(renamed, newName)
			continue
		}
		renamed = append(renamed, label)
	}
	return model.LabelsAndFavoriteToCategories(renamed, favorite)
}

func deleteCategoriesLabel(categories []string, oldName string) ([]string, error) {
	labels, favorite := model.CategoriesToLabelsAndFavorite(categories)
	filtered := make([]string, 0, len(labels))
	for _, label := range labels {
		if strings.EqualFold(label, oldName) {
			continue
		}
		filtered = append(filtered, label)
	}
	return model.LabelsAndFavoriteToCategories(filtered, favorite)
}

func normalizeManagedLabelName(name string) (string, error) {
	normalized := strings.TrimSpace(name)
	if normalized == "" {
		return "", errLabelNameRequired
	}
	if strings.EqualFold(normalized, model.ReservedFavoriteCategory) {
		return "", errLabelNameReserved
	}
	if strings.Contains(normalized, ",") {
		return "", errLabelNameInvalid
	}
	return normalized, nil
}

func labelMutationErrorMessage(action string, err error) string {
	switch {
	case errors.Is(err, errLabelNameRequired):
		return "labelname ist erforderlich"
	case errors.Is(err, errLabelNameInvalid):
		return "labelname darf kein komma enthalten"
	case errors.Is(err, errLabelNameReserved):
		return "dieser labelname ist reserviert"
	case errors.Is(err, errLabelConfirmationFailed):
		return "löschung muss bestätigt werden"
	case errors.Is(err, db.ErrLabelNotFound):
		return "label wurde nicht gefunden"
	case errors.Is(err, db.ErrTaskVersionMismatch):
		return "label konnte nicht vollständig " + action + " werden, weil eine Aufgabe inzwischen geändert wurde"
	case errors.Is(err, errLabelTabIDRequired):
		return "label konnte nicht " + action + " werden, weil die browser-sitzung fehlt"
	case errors.Is(err, errLabelCalDAVUnavailable):
		return "caldav-zugangsdaten sind nicht verfügbar"
	}

	var writeErr *labelTaskWriteError
	if errors.As(err, &writeErr) {
		if errors.Is(err, caldav.ErrPreconditionFailed) {
			return "label konnte nicht vollständig " + action + " werden; eine Aufgabe hat einen CalDAV-Konflikt"
		}
		return "label konnte nicht vollständig " + action + " werden (" + strconv.Itoa(writeErr.completed) + " von " + strconv.Itoa(writeErr.total) + " Aufgaben aktualisiert)"
	}

	return "label konnte nicht " + action + " werden"
}
