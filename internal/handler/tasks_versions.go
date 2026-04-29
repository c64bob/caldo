package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"caldo/internal/db"
)

type taskVersionsDependencies struct {
	database *db.Database
}

type taskVersionsResponseItem struct {
	TaskID        string `json:"task_id"`
	ServerVersion int    `json:"server_version"`
}

// TaskVersions returns current task server versions for known task ids.
func TaskVersions(deps taskVersionsDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		taskIDs := compactTaskIDs(r.URL.Query()["task_id"])
		if len(taskIDs) == 0 {
			http.Error(w, "task_id is required", http.StatusBadRequest)
			return
		}

		versions, err := deps.database.ListTaskVersions(r.Context(), taskIDs)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		resp := make([]taskVersionsResponseItem, 0, len(versions))
		for _, v := range versions {
			resp = append(resp, taskVersionsResponseItem{TaskID: v.TaskID, ServerVersion: v.ServerVersion})
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"tasks": resp})
	}
}

func compactTaskIDs(raw []string) []string {
	seen := make(map[string]struct{}, len(raw))
	results := make([]string, 0, len(raw))
	for _, taskID := range raw {
		taskID = strings.TrimSpace(taskID)
		if taskID == "" {
			continue
		}
		if _, ok := seen[taskID]; ok {
			continue
		}
		seen[taskID] = struct{}{}
		results = append(results, taskID)
	}
	return results
}
