package handler

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"caldo/internal/caldav"
	"caldo/internal/db"
	"caldo/internal/model"
	"github.com/go-chi/chi/v5"
)

// TaskFavorite updates the favorite status of a task via STARRED category.
func TaskFavorite(deps taskUpdateDependencies) http.HandlerFunc {
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
		favorite, err := parseFavoriteValue(r.FormValue("favorite"))
		if err != nil {
			http.Error(w, "favorite must be true or false", http.StatusBadRequest)
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

		if err := performTaskFavoriteUpdate(r.Context(), deps, taskID, expectedVersion, sessionID, tabID, favorite); err != nil {
			writeTaskCompletionError(w, err)
			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("task favorite updated"))
	}
}

func parseFavoriteValue(value string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "true", "1", "on", "yes":
		return true, nil
	case "false", "0", "off", "no":
		return false, nil
	default:
		return false, errors.New("invalid favorite value")
	}
}

func performTaskFavoriteUpdate(ctx context.Context, deps taskUpdateDependencies, taskID string, expectedVersion int, sessionID string, tabID string, favorite bool) error {
	base, err := deps.database.LoadTaskUpdateBase(ctx, taskID, "")
	if err != nil {
		switch {
		case errors.Is(err, db.ErrTaskNotFound):
			return errTaskCompletionVersionConflict
		default:
			return errTaskCompletionPersistPending
		}
	}
	if expectedVersion != base.ExpectedVersion {
		return errTaskCompletionVersionConflict
	}

	labels, _ := model.CategoriesToLabelsAndFavorite(model.ParseVTODOFields(base.RawVTODO).Categories)
	categories, err := model.LabelsAndFavoriteToCategories(labels, favorite)
	if err != nil {
		return errTaskCompletionPersistPending
	}
	patch := model.VTODOPatch{Categories: categories}
	rawVTODO := model.PatchVTODO(base.RawVTODO, patch)
	parsed := model.ParseVTODOFields(rawVTODO)

	creds, err := deps.database.LoadCalDAVCredentials(ctx, deps.encryptionKey)
	if err != nil {
		return errTaskCompletionCredentialsUnavailable
	}
	input := db.TaskUpdateInput{TaskID: taskID, ExpectedVersion: expectedVersion, SessionID: sessionID, TabID: tabID, ProjectID: base.ProjectID, ProjectName: base.ProjectName, Href: base.Href, ETag: base.ETag, RawVTODO: rawVTODO, Title: parsed.Title, Description: parsed.Description, Status: parsed.Status, DueDate: nullableDate(parsed.DueDate), DueAt: nullableTime(parsed.DueAt), Priority: nullableInt(parsed.Priority), LabelNames: nullableCSV(parsed.Categories)}
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
