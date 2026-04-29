package handler

import (
	"context"
	"errors"
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
		default:
			http.Error(w, "invalid resolution", http.StatusBadRequest)
			return
		}

		creds, err := deps.database.LoadCalDAVCredentials(r.Context(), deps.encryptionKey)
		if err != nil {
			http.Error(w, "caldav credentials unavailable", http.StatusFailedDependency)
			return
		}
		newETag, err := deps.todos.PutVTODOUpdate(r.Context(), caldav.Credentials{URL: creds.URL, Username: creds.Username, Password: creds.Password}, loaded.Href, resolved, loaded.ETag)
		if err != nil {
			http.Error(w, "failed to resolve conflict on caldav server", http.StatusBadGateway)
			return
		}

		persistCtx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), taskUpdatePersistTimeout)
		defer cancel()
		if err := deps.database.MarkConflictResolved(persistCtx, db.ResolveConflictInput{ConflictID: conflictID, Resolution: resolution, ResolvedVTODO: resolved, NewETag: newETag, ExpectedVersion: loaded.ServerVersion}); err != nil {
			http.Error(w, "failed to persist conflict resolution", http.StatusInternalServerError)
			return
		}

		if deps.broker != nil && loaded.TaskID != "" {
			deps.broker.publish(appEvent{Type: "task", Resource: loaded.TaskID, Version: loaded.ServerVersion + 1, OriginConnection: r.Header.Get("X-Tab-ID")})
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("conflict resolved"))
	}
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
