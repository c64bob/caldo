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
		loaded, err := deps.database.LoadConflictResolutionBase(r.Context(), conflictID)
		if err != nil {
			if errors.Is(err, db.ErrConflictNotFound) {
				http.NotFound(w, r)
				return
			}
			http.Error(w, "failed to load conflict", http.StatusInternalServerError)
			return
		}
		resolved := loaded.RemoteVTODO
		switch resolution {
		case "local":
			resolved = loaded.LocalVTODO
		case "remote":
			resolved = loaded.RemoteVTODO
		case "manual":
			patch := model.VTODOPatch{
				Summary:     optionalTrimmedFormPointer(r, "title"),
				Description: optionalTrimmedFormPointer(r, "description"),
				Status:      optionalLowerTrimmedFormPointer(r, "status"),
				DueDate:     parseOptionalDate(r.FormValue("due_date")),
				Priority:    parseOptionalInt(r.FormValue("priority")),
				Categories:  parseOptionalLabels(r.FormValue("labels")),
			}
			resolved = model.PatchVTODO(loaded.RemoteVTODO, patch)
		case "split":
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
		persistCtx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), taskUpdatePersistTimeout)
		defer cancel()
		if resolution == "split" {
			splitUID := splitConflictUID(conflictID)
			splitVTODO, err := prepareSplitVTODO(loaded.RemoteVTODO, splitUID)
			if err != nil {
				http.Error(w, "failed to build split task", http.StatusInternalServerError)
				return
			}
			splitHref := joinCalendarTaskHref(loaded.Href, splitUID)
			newETag, err := deps.todos.PutVTODOCreate(r.Context(), caldav.Credentials{URL: creds.URL, Username: creds.Username, Password: creds.Password}, splitHref, splitVTODO)
			if err != nil {
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
				http.Error(w, "failed to persist conflict resolution", http.StatusInternalServerError)
				return
			}
		} else {
			newETag, err := deps.todos.PutVTODOUpdate(r.Context(), caldav.Credentials{URL: creds.URL, Username: creds.Username, Password: creds.Password}, loaded.Href, resolved, loaded.ETag)
			if err != nil {
				http.Error(w, "failed to resolve conflict on caldav server", http.StatusBadGateway)
				return
			}
			if err := deps.database.MarkConflictResolved(persistCtx, db.ResolveConflictInput{ConflictID: conflictID, Resolution: resolution, ResolvedVTODO: resolved, NewETag: newETag, ExpectedVersion: loaded.ServerVersion}); err != nil {
				http.Error(w, "failed to persist conflict resolution", http.StatusInternalServerError)
				return
			}
		}

		if deps.broker != nil && loaded.TaskID != "" {
			deps.broker.publish(appEvent{Type: "task", Resource: loaded.TaskID, Version: loaded.ServerVersion + 1, OriginConnection: r.Header.Get("X-Tab-ID")})
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("conflict resolved"))
	}
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
		upper := strings.ToUpper(line)
		if strings.HasPrefix(upper, "RELATED-TO") && strings.Contains(upper, "RELTYPE=PARENT") {
			continue
		}
		filtered = append(filtered, line)
	}
	return strings.Join(filtered, "\r\n")
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
