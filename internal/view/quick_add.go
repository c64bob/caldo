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
