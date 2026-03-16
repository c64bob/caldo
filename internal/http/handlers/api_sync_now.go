package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"caldo/internal/http/middleware"
)

func (h *TasksHandler) APISyncNow(w http.ResponseWriter, r *http.Request) {
	if h.SyncService == nil {
		http.Error(w, "Sync-Service nicht verfügbar", http.StatusServiceUnavailable)
		return
	}
	principal, ok := middleware.PrincipalFromContext(r.Context())
	if !ok {
		http.Error(w, "Authentifizierungsheader fehlt", http.StatusUnauthorized)
		return
	}
	if forced := strings.TrimSpace(r.FormValue("principal_id")); forced != "" {
		principal = forced
	}
	result, err := h.SyncService.SyncNow(r.Context(), principal)
	if err != nil {
		http.Error(w, "Synchronisierung fehlgeschlagen: "+err.Error(), http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok":                 true,
		"principal_id":       result.PrincipalID,
		"collections":        result.Collections,
		"synced_collections": result.SyncedCollections,
		"mode":               result.Mode,
		"synced_at":          result.SyncedAt,
	})
}
