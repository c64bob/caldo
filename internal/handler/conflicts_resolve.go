package handler

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"caldo/internal/caldav"
	"caldo/internal/db"
	"caldo/internal/model"
	"github.com/go-chi/chi/v5"
)

// ResolveConflict resolves one unresolved conflict by writing the selected VTODO to CalDAV.
func ResolveConflict(deps taskUpdateDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form payload", http.StatusBadRequest)
			return
		}
		conflictID := chi.URLParam(r, "conflictID")
		resolution := strings.TrimSpace(r.FormValue("resolution"))
		loaded, err := deps.database.LoadConflictResolutionBase(r.Context(), conflictID, r.FormValue("project_id"))
		if err != nil {
			if errors.Is(err, db.ErrConflictNotFound) {
				http.NotFound(w, r)
				return
			}
			if errors.Is(err, db.ErrTaskProjectNotFound) {
				http.Error(w, "selected project does not exist", http.StatusBadRequest)
				return
			}
			http.Error(w, "failed to load conflict", http.StatusInternalServerError)
			return
		}
		resolved := loaded.RemoteVTODO
		switch resolution {
		case "local":
			if strings.TrimSpace(loaded.LocalVTODO) == "" {
				http.Error(w, "local version is unavailable", http.StatusBadRequest)
				return
			}
			resolved = loaded.LocalVTODO
		case "remote":
			if strings.TrimSpace(loaded.RemoteVTODO) == "" {
				http.Error(w, "remote version is unavailable", http.StatusBadRequest)
				return
			}
			resolved = loaded.RemoteVTODO
		case "manual":
			resolved, err = buildManualConflictVTODO(loaded, r.PostForm)
			if err != nil {
				http.Error(w, "invalid manual conflict resolution", http.StatusBadRequest)
				return
			}
		case "split":
			if strings.TrimSpace(loaded.RemoteVTODO) == "" {
				http.Error(w, "remote version is unavailable", http.StatusBadRequest)
				return
			}
			if !formBool(firstFormValue(r.PostForm, "confirm_split")) {
				http.Error(w, "split confirmation is required", http.StatusBadRequest)
				return
			}
			resolved = loaded.RemoteVTODO
		default:
			http.Error(w, "invalid resolution", http.StatusBadRequest)
			return
		}

		creds, err := deps.database.LoadCalDAVCredentials(r.Context(), deps.encryptionKey)
		if err != nil {
			http.Error(w, "caldav credentials unavailable", http.StatusFailedDependency)
			return
		}
		todoCredentials := caldav.Credentials{URL: creds.URL, Username: creds.Username, Password: creds.Password}
		persistCtx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), taskUpdatePersistTimeout)
		defer cancel()
		if resolution == "split" {
			splitUID := splitConflictUID(conflictID)
			splitVTODO, err := prepareSplitVTODO(loaded.RemoteVTODO, splitUID)
			if err != nil {
				http.Error(w, "failed to build split task", http.StatusInternalServerError)
				return
			}
			splitHref := joinCalendarTaskHref(loaded.PreviousHref, splitUID)
			newETag, err := deps.todos.PutVTODOCreate(r.Context(), todoCredentials, splitHref, splitVTODO)
			if err != nil {
				if errors.Is(err, caldav.ErrPreconditionFailed) {
					http.Error(w, "conflict changed on caldav server", http.StatusConflict)
					return
				}
				http.Error(w, "failed to resolve conflict on caldav server", http.StatusBadGateway)
				return
			}
			if err := deps.database.MarkConflictSplitResolved(persistCtx, db.ResolveConflictSplitInput{
				ConflictID:      conflictID,
				ResolvedVTODO:   splitVTODO,
				NewTaskUID:      splitUID,
				NewTaskHref:     splitHref,
				NewTaskETag:     newETag,
				ExpectedVersion: loaded.ServerVersion,
			}); err != nil {
				cleanupCtx, cleanupCancel := context.WithTimeout(context.WithoutCancel(r.Context()), taskUpdatePersistTimeout)
				defer cleanupCancel()
				_ = deps.todos.DeleteVTODO(cleanupCtx, todoCredentials, splitHref, newETag)
				http.Error(w, "failed to persist conflict resolution", http.StatusInternalServerError)
				return
			}
		} else {
			var newETag string
			if loaded.ProjectChanged || conflictRemoteWasDeleted(loaded) {
				newETag, err = deps.todos.PutVTODOCreate(r.Context(), todoCredentials, loaded.NextHref, resolved)
				if err == nil && loaded.ProjectChanged {
					err = deps.todos.DeleteVTODO(r.Context(), todoCredentials, loaded.PreviousHref, loaded.PreviousETag)
				}
			} else {
				newETag, err = deps.todos.PutVTODOUpdate(r.Context(), todoCredentials, loaded.PreviousHref, resolved, loaded.PreviousETag)
			}
			if err != nil {
				if errors.Is(err, caldav.ErrPreconditionFailed) {
					http.Error(w, "conflict changed on caldav server", http.StatusConflict)
					return
				}
				http.Error(w, "failed to resolve conflict on caldav server", http.StatusBadGateway)
				return
			}
			if err := deps.database.MarkConflictResolved(persistCtx, db.ResolveConflictInput{
				ConflictID:      conflictID,
				Resolution:      resolution,
				ResolvedVTODO:   resolved,
				NewETag:         newETag,
				ProjectID:       loaded.ProjectID,
				ProjectName:     loaded.ProjectName,
				Href:            loaded.NextHref,
				ExpectedVersion: loaded.ServerVersion,
			}); err != nil {
				http.Error(w, "failed to persist conflict resolution", http.StatusInternalServerError)
				return
			}
		}

		if deps.broker != nil && loaded.TaskID != "" {
			deps.broker.publish(appEvent{Type: "task", Resource: loaded.TaskID, Version: loaded.ServerVersion + 1, OriginConnection: r.Header.Get("X-Tab-ID")})
		}
		if strings.EqualFold(r.Header.Get("HX-Request"), "true") {
			w.Header().Set("HX-Redirect", "/conflicts")
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("conflict resolved"))
	}
}

type conflictVersionFields struct {
	raw     string
	fields  model.VTODOFields
	present bool
}

func buildManualConflictVTODO(loaded db.ConflictResolutionBase, form map[string][]string) (string, error) {
	base := conflictVersionFromRaw(loaded.BaseVTODO)
	local := conflictVersionFromRaw(loaded.LocalVTODO)
	remote := conflictVersionFromRaw(loaded.RemoteVTODO)
	target := remote
	if !target.present {
		target = local
	}
	if !target.present {
		target = base
	}
	if !target.present {
		return loaded.RemoteVTODO, nil
	}

	patch := model.VTODOPatch{}
	priorityTouched := false
	if conflictFieldSource(form, "title") == "manual" {
		title := strings.TrimSpace(firstFormValue(form, "title_manual"))
		patch.Summary = &title
	} else if selected, ok := selectedConflictVersion(form, "title", base, local, remote); ok {
		title := strings.TrimSpace(selected.fields.Title)
		patch.Summary = &title
	}
	if conflictFieldSource(form, "description") == "manual" {
		description := strings.TrimSpace(firstFormValue(form, "description_manual"))
		patch.Description = &description
	} else if selected, ok := selectedConflictVersion(form, "description", base, local, remote); ok {
		description := strings.TrimSpace(selected.fields.Description)
		patch.Description = &description
	}
	if conflictFieldSource(form, "status") == "manual" {
		status, ok := manualConflictStatus(firstFormValue(form, "status_manual"))
		if !ok {
			return "", fmt.Errorf("invalid manual status")
		}
		patch.Status = &status
		if status == "completed" && target.fields.CompletedAt != nil {
			patch.CompletedAt = target.fields.CompletedAt
		}
		if status != "completed" {
			patch.ClearCompleted = true
		}
	} else if selected, ok := selectedConflictVersion(form, "status", base, local, remote); ok {
		status := strings.ToLower(strings.TrimSpace(selected.fields.Status))
		if status == "" {
			status = "needs-action"
		}
		patch.Status = &status
		if status == "completed" && selected.fields.CompletedAt != nil {
			patch.CompletedAt = selected.fields.CompletedAt
		}
		if status != "completed" {
			patch.ClearCompleted = true
		}
	}
	if conflictFieldSource(form, "due") == "manual" {
		if err := applyManualConflictDuePatch(firstFormValue(form, "due_manual"), &patch); err != nil {
			return "", err
		}
	} else if selected, ok := selectedConflictVersion(form, "due", base, local, remote); ok {
		switch {
		case selected.fields.DueDate != nil:
			dueDate := strings.TrimSpace(*selected.fields.DueDate)
			patch.DueDate = &dueDate
		case selected.fields.DueAt != nil:
			patch.DueAt = selected.fields.DueAt
		default:
			patch.ClearDue = true
		}
	}
	if conflictFieldSource(form, "priority") == "manual" {
		if err := applyManualConflictPriorityPatch(firstFormValue(form, "priority_manual"), &patch); err != nil {
			return "", err
		}
		priorityTouched = true
	} else if selected, ok := selectedConflictVersion(form, "priority", base, local, remote); ok {
		if selected.fields.Priority != nil {
			patch.Priority = selected.fields.Priority
		} else {
			patch.ClearPriority = true
		}
		priorityTouched = true
	}
	if conflictFieldSource(form, "labels") == "manual" {
		if err := applyManualConflictLabelsPatch(firstFormValue(form, "labels_manual"), target.fields, &patch); err != nil {
			return "", err
		}
	} else if selected, ok := selectedConflictVersion(form, "labels", base, local, remote); ok {
		patch.Categories = append([]string(nil), selected.fields.Categories...)
	}
	if conflictFieldSource(form, "parent") == "manual" {
		parentUID := strings.TrimSpace(firstFormValue(form, "parent_manual"))
		if parentUID == "" {
			patch.ClearParent = true
		} else {
			patch.ParentUID = &parentUID
		}
	} else if selected, ok := selectedConflictVersion(form, "parent", base, local, remote); ok {
		parentUID := strings.TrimSpace(selected.fields.ParentUID)
		if parentUID == "" {
			patch.ClearParent = true
		} else {
			patch.ParentUID = &parentUID
		}
	}

	applyFavoritePriorityRule(target.fields, &patch, priorityTouched)
	return model.PatchVTODO(target.raw, patch), nil
}

func conflictVersionFromRaw(raw string) conflictVersionFields {
	if strings.TrimSpace(raw) == "" {
		return conflictVersionFields{}
	}
	return conflictVersionFields{
		raw:     raw,
		fields:  model.ParseVTODOFields(raw),
		present: true,
	}
}

func selectedConflictVersion(form map[string][]string, field string, base conflictVersionFields, local conflictVersionFields, remote conflictVersionFields) (conflictVersionFields, bool) {
	switch conflictFieldSource(form, field) {
	case "base":
		return base, base.present
	case "local":
		return local, local.present
	case "remote", "":
		return remote, remote.present
	default:
		return conflictVersionFields{}, false
	}
}

func conflictFieldSource(form map[string][]string, field string) string {
	switch strings.ToLower(strings.TrimSpace(firstFormValue(form, field+"_source"))) {
	case "base", "local", "remote", "manual":
		return strings.ToLower(strings.TrimSpace(firstFormValue(form, field+"_source")))
	default:
		return "remote"
	}
}

func manualConflictStatus(raw string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "needs-action":
		return "needs-action", true
	case "in-process":
		return "in-process", true
	case "completed":
		return "completed", true
	case "cancelled":
		return "cancelled", true
	default:
		return "", false
	}
}

func applyManualConflictDuePatch(raw string, patch *model.VTODOPatch) error {
	value := strings.TrimSpace(raw)
	if value == "" {
		patch.ClearDue = true
		return nil
	}
	if dueDate := parseOptionalDate(value); dueDate != nil {
		patch.DueDate = dueDate
		return nil
	}
	if dueAt := parseOptionalDateTime(value); dueAt != nil {
		patch.DueAt = dueAt
		return nil
	}
	return fmt.Errorf("invalid manual due")
}

func applyManualConflictPriorityPatch(raw string, patch *model.VTODOPatch) error {
	value := strings.TrimSpace(raw)
	if value == "" {
		patch.ClearPriority = true
		return nil
	}
	priority := parseOptionalInt(value)
	if priority == nil || *priority < 1 || *priority > 9 {
		return fmt.Errorf("invalid manual priority")
	}
	patch.Priority = priority
	return nil
}

func applyManualConflictLabelsPatch(raw string, targetFields model.VTODOFields, patch *model.VTODOPatch) error {
	_, isFavorite := model.CategoriesToLabelsAndFavorite(targetFields.Categories)
	categories, err := model.LabelsAndFavoriteToCategories(parseLabels(raw), isFavorite)
	if err != nil {
		return fmt.Errorf("invalid manual labels: %w", err)
	}
	patch.Categories = categories
	return nil
}

func conflictRemoteWasDeleted(loaded db.ConflictResolutionBase) bool {
	return strings.EqualFold(strings.TrimSpace(loaded.ConflictType), "edit_delete") &&
		strings.TrimSpace(loaded.RemoteVTODO) == "" &&
		strings.TrimSpace(loaded.LocalVTODO) != ""
}

func prepareSplitVTODO(raw string, uid string) (string, error) {
	rewrittenUID, err := replaceVTODOUID(raw, uid)
	if err != nil {
		return "", err
	}
	return removeParentRelatedTo(rewrittenUID), nil
}

func splitConflictUID(conflictID string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(conflictID)))
	return "split-" + hex.EncodeToString(sum[:16])
}

func removeParentRelatedTo(raw string) string {
	lines := strings.Split(strings.ReplaceAll(raw, "\r\n", "\n"), "\n")
	filtered := make([]string, 0, len(lines))
	for _, line := range lines {
		if isParentRelatedToLine(line) {
			continue
		}
		filtered = append(filtered, line)
	}
	return strings.Join(filtered, "\r\n")
}

func isParentRelatedToLine(line string) bool {
	rawName, _, ok := strings.Cut(line, ":")
	if !ok {
		return false
	}
	name, rawParams, hasParams := strings.Cut(strings.ToUpper(strings.TrimSpace(rawName)), ";")
	if name != "RELATED-TO" {
		return false
	}
	if !hasParams {
		return true
	}
	for _, param := range strings.Split(rawParams, ";") {
		key, value, hasValue := strings.Cut(strings.TrimSpace(param), "=")
		if !hasValue || key != "RELTYPE" {
			continue
		}
		return strings.EqualFold(strings.Trim(value, `"`), "PARENT")
	}
	return false
}

func replaceVTODOUID(raw string, uid string) (string, error) {
	lines := strings.Split(strings.ReplaceAll(raw, "\r\n", "\n"), "\n")
	for i, line := range lines {
		if strings.HasPrefix(strings.ToUpper(line), "UID:") {
			lines[i] = "UID:" + uid
			return strings.Join(lines, "\r\n"), nil
		}
	}
	return "", fmt.Errorf("uid property not found in vtodo")
}

func joinCalendarTaskHref(existingHref string, uid string) string {
	trimmedHref := strings.TrimSpace(existingHref)
	slash := strings.LastIndex(trimmedHref, "/")
	if slash < 0 {
		return "/" + uid + ".ics"
	}
	return trimmedHref[:slash+1] + uid + ".ics"
}

func optionalTrimmedFormPointer(r *http.Request, key string) *string {
	if _, ok := r.PostForm[key]; !ok {
		return nil
	}
	return stringPointer(strings.TrimSpace(r.PostFormValue(key)))
}

func optionalLowerTrimmedFormPointer(r *http.Request, key string) *string {
	if _, ok := r.PostForm[key]; !ok {
		return nil
	}
	return stringPointer(strings.ToLower(strings.TrimSpace(r.PostFormValue(key))))
}
