package view

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"

	"caldo/internal/model"
)

// TaskRowView contains all fields rendered by the shared task-list row.
type TaskRowView struct {
	ID             string
	ProjectID      string
	Title          string
	Description    string
	ProjectName    string
	LabelNames     string
	DueISODate     string
	Status         string
	SyncStatus     string
	Priority       int
	HasPriority    bool
	ServerVersion  int
	IsSubtask      bool
	Attachments    []model.Attachment
	ProjectOptions []TaskProjectOption
}

// TaskProjectOption contains one project selectable from inline task editing.
type TaskProjectOption struct {
	ID   string
	Name string
}

// InlineTaskCreateView contains context hints for the inline task creator.
type InlineTaskCreateView struct {
	Enabled     bool
	ProjectID   string
	ProjectName string
	DueDate     string
	Placeholder string
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

func taskEditPath(task TaskRowView) string {
	return "/tasks/" + url.PathEscape(strings.TrimSpace(task.ID))
}

func taskCSRFHeaders(ctx context.Context) string {
	encoded, err := json.Marshal(map[string]string{"X-CSRF-Token": CSRFToken(ctx)})
	if err != nil {
		return "{}"
	}
	return string(encoded)
}

func inlineTaskCreatePlaceholder(create InlineTaskCreateView) string {
	placeholder := strings.TrimSpace(create.Placeholder)
	if placeholder != "" {
		return placeholder
	}
	return "Aufgabe hinzufügen"
}

func inlineTaskCreateContextChips(create InlineTaskCreateView) []taskRowChip {
	chips := make([]taskRowChip, 0, 2)
	if project := strings.TrimSpace(create.ProjectName); project != "" {
		chips = append(chips, taskRowChip{Label: project, Class: "caldo-task-chip"})
	}
	if due := strings.TrimSpace(create.DueDate); due != "" {
		chips = append(chips, taskRowChip{Label: "Fällig " + due, Class: "caldo-task-chip caldo-task-chip-due"})
	}
	return chips
}

func taskCanToggleCompletion(task TaskRowView) bool {
	if strings.TrimSpace(task.ID) == "" || task.ServerVersion <= 0 {
		return false
	}
	status := strings.ToLower(strings.TrimSpace(task.SyncStatus))
	return status == "" || status == "synced"
}

func taskCanInlineEdit(task TaskRowView) bool {
	return taskCanToggleCompletion(task)
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

func taskEditableLabels(task TaskRowView) string {
	return strings.Join(splitTaskLabels(task.LabelNames), ", ")
}

func taskPriorityValue(task TaskRowView) string {
	if !task.HasPriority || task.Priority <= 0 {
		return ""
	}
	return strconv.Itoa(task.Priority)
}

type taskPriorityOption struct {
	Value string
	Label string
}

func taskPriorityOptions() []taskPriorityOption {
	return []taskPriorityOption{
		{Value: "", Label: "Keine"},
		{Value: "1", Label: "P1 - 1"},
		{Value: "2", Label: "P1 - 2"},
		{Value: "3", Label: "P1 - 3"},
		{Value: "4", Label: "P1 - 4"},
		{Value: "5", Label: "P2 - 5"},
		{Value: "6", Label: "P2 - 6"},
		{Value: "7", Label: "P3 - 7"},
		{Value: "8", Label: "P3 - 8"},
		{Value: "9", Label: "P3 - 9"},
	}
}

func taskProjectOptions(task TaskRowView) []TaskProjectOption {
	options := make([]TaskProjectOption, 0, len(task.ProjectOptions)+1)
	seenCurrent := strings.TrimSpace(task.ProjectID) == ""
	for _, option := range task.ProjectOptions {
		if strings.TrimSpace(option.ID) == "" {
			continue
		}
		if strings.TrimSpace(option.ID) == strings.TrimSpace(task.ProjectID) {
			seenCurrent = true
		}
		options = append(options, option)
	}
	if !seenCurrent {
		options = append([]TaskProjectOption{{ID: task.ProjectID, Name: task.ProjectName}}, options...)
	}
	return options
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
