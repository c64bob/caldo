package view

import (
	"context"
	"encoding/json"
	"strings"

	"caldo/internal/parser"
)

func quickAddCSRFHeaders(ctx context.Context) string {
	encoded, err := json.Marshal(map[string]string{"X-CSRF-Token": CSRFToken(ctx)})
	if err != nil {
		return "{}"
	}
	return string(encoded)
}

func quickAddProjectLabel(draft parser.QuickAddDraft) string {
	if project := strings.TrimSpace(draft.Project); project != "" {
		return project
	}
	return "Default-Projekt"
}

func quickAddProjectStatus(draft parser.QuickAddDraft) string {
	switch {
	case draft.ProjectNew:
		return "Neu anlegen"
	case strings.TrimSpace(draft.ProjectID) != "":
		return "Gefunden"
	default:
		return "Default"
	}
}

func quickAddLabelsValue(draft parser.QuickAddDraft) string {
	return strings.Join(draft.Labels, ", ")
}

func quickAddDisplayValue(value string) string {
	if trimmed := strings.TrimSpace(value); trimmed != "" {
		return trimmed
	}
	return "Keine"
}

func quickAddPriorityLabel(priority string) string {
	switch strings.ToLower(strings.TrimSpace(priority)) {
	case "high":
		return "high"
	case "medium":
		return "medium"
	case "low":
		return "low"
	default:
		return "Keine"
	}
}
