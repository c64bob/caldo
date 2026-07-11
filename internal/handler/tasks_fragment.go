package handler

import (
	"errors"
	"net/http"
	"strings"

	"caldo/internal/db"
	"caldo/internal/model"
	"caldo/internal/view"
	"github.com/go-chi/chi/v5"
)

// TaskFragment renders the current task row as a small HTML fragment.
func TaskFragment(deps dateViewDependencies) http.HandlerFunc {
	nowFn := withDefaultNow(deps.now)

	return func(w http.ResponseWriter, r *http.Request) {
		taskID := strings.TrimSpace(chi.URLParam(r, "taskID"))
		if taskID == "" {
			http.Error(w, "task id is required", http.StatusBadRequest)
			return
		}

		row, err := deps.database.LoadTaskView(r.Context(), taskID)
		if err != nil {
			if errors.Is(err, db.ErrTaskNotFound) {
				http.Error(w, "task not found", http.StatusNotFound)
				return
			}
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		projectOptions, err := taskEditProjectOptions(r.Context(), deps.database)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := view.TaskRow(taskRowFromDatedRow(row, projectOptions, nowFn().UTC().Format("2006-01-02"))).Render(r.Context(), w); err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
	}
}

func taskRowFromDatedRow(row db.DatedTaskViewRow, projectOptions []view.TaskProjectOption, todayISODate string) view.TaskRowView {
	fields := model.ParseVTODOFields(row.RawVTODO)
	return view.TaskRowView{
		ID:               row.ID,
		ProjectID:        row.ProjectID,
		Title:            row.Title,
		Description:      row.Description,
		ProjectName:      row.ProjectName,
		LabelNames:       row.LabelNames,
		DueISODate:       row.DueISODate,
		TodayISODate:     todayISODate,
		ParentID:         row.ParentID,
		ParentTitle:      row.ParentTitle,
		Status:           row.Status,
		SyncStatus:       row.SyncStatus,
		Priority:         row.Priority,
		HasPriority:      row.HasPriority,
		ServerVersion:    row.ServerVersion,
		IsSubtask:        row.IsSubtask,
		SubtaskCount:     row.SubtaskCount,
		OpenSubtaskCount: row.OpenSubtaskCount,
		ConflictID:       row.UnresolvedConflictID,
		RRule:            fields.RRule,
		Attachments:      fields.Attachments,
		ProjectOptions:   projectOptions,
		CreatedAt:        row.CreatedAt,
	}
}
