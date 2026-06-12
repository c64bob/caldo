package view

import (
	"context"
	"encoding/json"
	"strings"

	"caldo/internal/parser"
	"github.com/a-h/templ"
)

const (
	quickAddOverlaySurface       = "overlay"
	quickAddLabelSuggestionLimit = 8
)

func quickAddCSRFHeaders(ctx context.Context) string {
	encoded, err := json.Marshal(map[string]string{"X-CSRF-Token": CSRFToken(ctx)})
	if err != nil {
		return "{}"
	}
	return string(encoded)
}

func quickAddProjectLabel(ctx context.Context, draft parser.QuickAddDraft) string {
	if project := strings.TrimSpace(draft.Project); project != "" {
		return project
	}
	return Text(ctx).QuickAddDefaultProject
}

func quickAddProjectStatus(ctx context.Context, draft parser.QuickAddDraft) string {
	switch {
	case draft.ProjectNew:
		return Text(ctx).QuickAddCreateNew
	case strings.TrimSpace(draft.ProjectID) != "":
		return Text(ctx).QuickAddFound
	default:
		return Text(ctx).QuickAddDefault
	}
}

func quickAddLabelsValue(draft parser.QuickAddDraft) string {
	return strings.Join(draft.Labels, ", ")
}

func quickAddLabelKnown(draft parser.QuickAddDraft, label string) bool {
	key := strings.ToLower(strings.TrimSpace(label))
	if key == "" {
		return false
	}
	for _, option := range draft.LabelOptions {
		if strings.ToLower(strings.TrimSpace(option.Name)) == key {
			return true
		}
	}
	return false
}

func quickAddLabelStatus(ctx context.Context, draft parser.QuickAddDraft, label string) string {
	if quickAddLabelKnown(draft, label) {
		return Text(ctx).QuickAddFound
	}
	return Text(ctx).QuickAddNewLabel
}

func quickAddLabelDatalistID(previewID string) string {
	id := strings.TrimSpace(previewID)
	if id == "" {
		return "quick-add-label-options"
	}
	return id + "-label-options"
}

func quickAddLabelSuggestions(draft parser.QuickAddDraft) []parser.QuickAddLabelSuggestion {
	selected := make(map[string]struct{}, len(draft.Labels))
	for _, label := range draft.Labels {
		key := strings.ToLower(strings.TrimSpace(label))
		if key != "" {
			selected[key] = struct{}{}
		}
	}

	suggestions := make([]parser.QuickAddLabelSuggestion, 0, min(len(draft.LabelOptions), quickAddLabelSuggestionLimit))
	seen := make(map[string]struct{}, len(draft.LabelOptions))
	for _, option := range draft.LabelOptions {
		name := strings.TrimSpace(option.Name)
		if name == "" {
			continue
		}
		key := strings.ToLower(name)
		if _, ok := selected[key]; ok {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		suggestions = append(suggestions, parser.QuickAddLabelSuggestion{Name: name})
		if len(suggestions) >= quickAddLabelSuggestionLimit {
			break
		}
	}
	return suggestions
}

func quickAddHasLabelSuggestions(draft parser.QuickAddDraft) bool {
	return len(quickAddLabelSuggestions(draft)) > 0
}

func quickAddDisplayValue(ctx context.Context, value string) string {
	if trimmed := strings.TrimSpace(value); trimmed != "" {
		return trimmed
	}
	return Text(ctx).None
}

func quickAddPriorityLabel(ctx context.Context, priority string) string {
	switch strings.ToLower(strings.TrimSpace(priority)) {
	case "high":
		return Text(ctx).PriorityHigh
	case "medium":
		return Text(ctx).PriorityMedium
	case "low":
		return Text(ctx).PriorityLow
	default:
		return Text(ctx).None
	}
}

func quickAddHasRecognizedTokens(draft parser.QuickAddDraft) bool {
	return strings.TrimSpace(draft.Project) != "" ||
		len(draft.Labels) > 0 ||
		strings.TrimSpace(draft.Due) != "" ||
		strings.TrimSpace(draft.Recurrence) != "" ||
		strings.TrimSpace(draft.Priority) != ""
}

func quickAddProjectSelected(draft parser.QuickAddDraft, projectID string) bool {
	return strings.TrimSpace(projectID) != "" && strings.TrimSpace(projectID) == strings.TrimSpace(draft.ProjectID)
}

func quickAddSelectProjectValue(projectID string) string {
	id := strings.TrimSpace(projectID)
	if id == "" {
		return ""
	}
	return "existing:" + id
}

func quickAddOverlayAttributes(surface string) templ.Attributes {
	if surface == quickAddOverlaySurface {
		return templ.Attributes{"data-quick-add-overlay-save-form": ""}
	}

	return nil
}
