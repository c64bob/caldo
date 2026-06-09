package view

import (
	"context"
	"encoding/json"
	"net/url"
	"strings"

	"caldo/internal/model"
)

// TaskRowView contains all fields rendered by the shared task-list row.
type TaskRowView struct {
	ID            string
	Title         string
	Description   string
	ProjectName   string
	LabelNames    string
	DueISODate    string
	Status        string
	SyncStatus    string
	Priority      int
	HasPriority   bool
	ServerVersion int
	IsSubtask     bool
	Attachments   []model.Attachment
}

type taskRowChip struct {
	Label string
	Class string
}

func taskRowTitle(task TaskRowView) string {
	title := strings.TrimSpace(task.Title)
	if title == "" {
		return "Ohne Titel"
	}
	return title
}

func taskRowDescription(task TaskRowView) string {
	return strings.TrimSpace(task.Description)
}

func taskIsCompleted(task TaskRowView) bool {
	return strings.EqualFold(strings.TrimSpace(task.Status), "completed")
}

func taskCompletionPressed(task TaskRowView) string {
	if taskIsCompleted(task) {
		return "true"
	}
	return "false"
}

func taskCompletionLabel(task TaskRowView) string {
	if taskIsCompleted(task) {
		return "Aufgabe wieder öffnen"
	}
	return "Aufgabe erledigen"
}

func taskCompletionPath(task TaskRowView) string {
	action := "complete"
	if taskIsCompleted(task) {
		action = "reopen"
	}
	return "/tasks/" + url.PathEscape(strings.TrimSpace(task.ID)) + "/" + action
}

func taskCSRFHeaders(ctx context.Context) string {
	encoded, err := json.Marshal(map[string]string{"X-CSRF-Token": CSRFToken(ctx)})
	if err != nil {
		return "{}"
	}
	return string(encoded)
}

func taskCanToggleCompletion(task TaskRowView) bool {
	if strings.TrimSpace(task.ID) == "" || task.ServerVersion <= 0 {
		return false
	}
	status := strings.ToLower(strings.TrimSpace(task.SyncStatus))
	return status == "" || status == "synced"
}

func taskMetaChips(task TaskRowView) []taskRowChip {
	chips := make([]taskRowChip, 0, 4)
	if due := strings.TrimSpace(task.DueISODate); due != "" {
		chips = append(chips, taskRowChip{Label: "Fällig " + due, Class: "caldo-task-chip caldo-task-chip-due"})
	}
	if project := strings.TrimSpace(task.ProjectName); project != "" {
		chips = append(chips, taskRowChip{Label: project, Class: "caldo-task-chip"})
	}
	if priority := taskPriorityLabel(task); priority != "" {
		chips = append(chips, taskRowChip{Label: priority, Class: "caldo-task-chip " + taskPriorityClass(task)})
	}
	return chips
}

func taskLabelChips(task TaskRowView) []taskRowChip {
	labels := splitTaskLabels(task.LabelNames)
	chips := make([]taskRowChip, 0, len(labels))
	for _, label := range labels {
		chips = append(chips, taskRowChip{Label: label, Class: "caldo-task-chip caldo-task-chip-label"})
	}
	return chips
}

func taskStateChips(task TaskRowView) []taskRowChip {
	chips := make([]taskRowChip, 0, 2)
	switch strings.ToLower(strings.TrimSpace(task.SyncStatus)) {
	case "pending":
		chips = append(chips, taskRowChip{Label: "Speichert", Class: "caldo-task-state caldo-task-state-pending"})
	case "error":
		chips = append(chips, taskRowChip{Label: "Fehler", Class: "caldo-task-state caldo-task-state-error"})
	case "conflict":
		chips = append(chips, taskRowChip{Label: "Konflikt", Class: "caldo-task-state caldo-task-state-conflict"})
	}
	if taskIsCompleted(task) {
		chips = append(chips, taskRowChip{Label: "Erledigt", Class: "caldo-task-state"})
	}
	return chips
}

func taskPriorityLabel(task TaskRowView) string {
	if !task.HasPriority || task.Priority <= 0 {
		return ""
	}
	switch {
	case task.Priority <= 4:
		return "P1"
	case task.Priority <= 6:
		return "P2"
	default:
		return "P3"
	}
}

func taskPriorityClass(task TaskRowView) string {
	if !task.HasPriority || task.Priority <= 0 {
		return ""
	}
	switch {
	case task.Priority <= 4:
		return "caldo-task-priority-p1"
	case task.Priority <= 6:
		return "caldo-task-priority-p2"
	default:
		return "caldo-task-priority-p3"
	}
}

func splitTaskLabels(raw string) []string {
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ';' || r == '\n' || r == '\t'
	})
	labels := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		label := strings.TrimSpace(part)
		if label == "" || strings.EqualFold(label, model.ReservedFavoriteCategory) {
			continue
		}
		key := strings.ToLower(label)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		labels = append(labels, label)
	}
	return labels
}
