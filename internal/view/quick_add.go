package view

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
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

func quickAddDueChipLabel(draft parser.QuickAddDraft) string {
	due := strings.TrimSpace(draft.Due)
	if due == "" {
		return ""
	}
	source := strings.TrimSpace(draft.DueSource)
	if source == "" {
		return due
	}
	return source + " -> " + due
}

func quickAddPriorityLabel(ctx context.Context, priority string) string {
	switch strings.ToLower(strings.TrimSpace(priority)) {
	case "high":
		return "P1 " + Text(ctx).PriorityHigh
	case "medium":
		return "P2 " + Text(ctx).PriorityMedium
	case "low":
		return "P3 " + Text(ctx).PriorityLow
	default:
		return Text(ctx).None
	}
}

func quickAddPriorityChipClass(priority string) string {
	switch strings.ToLower(strings.TrimSpace(priority)) {
	case "high":
		return "caldo-quick-add-chip caldo-quick-add-priority-p1"
	case "medium":
		return "caldo-quick-add-chip caldo-quick-add-priority-p2"
	case "low":
		return "caldo-quick-add-chip caldo-quick-add-priority-p3"
	default:
		return "caldo-quick-add-chip"
	}
}

type quickAddPriorityOption struct {
	Value string
	Label string
}

func quickAddPriorityOptions(ctx context.Context) []quickAddPriorityOption {
	return []quickAddPriorityOption{
		{Value: "high", Label: quickAddPriorityLabel(ctx, "high")},
		{Value: "medium", Label: quickAddPriorityLabel(ctx, "medium")},
		{Value: "low", Label: quickAddPriorityLabel(ctx, "low")},
	}
}

func quickAddRecurrenceDisplayValue(ctx context.Context, recurrence string) string {
	if label := quickAddRecurrenceLabel(ctx, recurrence); label != "" {
		return label
	}
	return Text(ctx).None
}

func quickAddRecurrenceLabel(ctx context.Context, recurrence string) string {
	trimmed := strings.TrimSpace(recurrence)
	if trimmed == "" {
		return ""
	}
	if label, ok := quickAddSimpleRecurrenceLabel(ctx, trimmed); ok {
		return label
	}
	return "RRULE: " + trimmed
}

func quickAddSimpleRecurrenceLabel(ctx context.Context, recurrence string) (string, bool) {
	parts, ok := quickAddRRuleParts(recurrence)
	if !ok {
		return "", false
	}

	freq := parts["FREQ"]
	if freq == "" {
		return "", false
	}

	interval := 1
	if rawInterval := parts["INTERVAL"]; rawInterval != "" {
		parsed, err := strconv.Atoi(rawInterval)
		if err != nil || parsed <= 0 {
			return "", false
		}
		interval = parsed
	}

	allowedSimpleKeys := map[string]struct{}{"FREQ": {}, "INTERVAL": {}}
	if byday := parts["BYDAY"]; byday != "" {
		allowedSimpleKeys["BYDAY"] = struct{}{}
		if freq != "WEEKLY" || interval != 1 {
			return "", false
		}
		if byday == "MO,TU,WE,TH,FR" {
			return localizedQuickAddText(ctx, "Werktags", "Weekdays"), quickAddOnlyKeys(parts, allowedSimpleKeys)
		}
		if !strings.Contains(byday, ",") {
			if weekday, ok := quickAddWeekdayName(ctx, byday); ok {
				if UILanguage(ctx) == "en" {
					return "Every " + weekday, quickAddOnlyKeys(parts, allowedSimpleKeys)
				}
				return "Jeden " + weekday, quickAddOnlyKeys(parts, allowedSimpleKeys)
			}
		}
		return "", false
	}

	if !quickAddOnlyKeys(parts, allowedSimpleKeys) {
		return "", false
	}
	return quickAddIntervalRecurrenceLabel(ctx, freq, interval)
}

func quickAddRRuleParts(recurrence string) (map[string]string, bool) {
	parts := strings.Split(recurrence, ";")
	result := make(map[string]string, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			return nil, false
		}
		key, value, ok := strings.Cut(part, "=")
		if !ok {
			return nil, false
		}
		key = strings.ToUpper(strings.TrimSpace(key))
		value = strings.ToUpper(strings.TrimSpace(value))
		if key == "" || value == "" {
			return nil, false
		}
		if _, exists := result[key]; exists {
			return nil, false
		}
		result[key] = value
	}
	return result, true
}

func quickAddOnlyKeys(parts map[string]string, allowed map[string]struct{}) bool {
	for key := range parts {
		if _, ok := allowed[key]; !ok {
			return false
		}
	}
	return true
}

func quickAddIntervalRecurrenceLabel(ctx context.Context, freq string, interval int) (string, bool) {
	if interval <= 1 {
		switch freq {
		case "DAILY":
			return localizedQuickAddText(ctx, "Täglich", "Daily"), true
		case "WEEKLY":
			return localizedQuickAddText(ctx, "Wöchentlich", "Weekly"), true
		case "MONTHLY":
			return localizedQuickAddText(ctx, "Monatlich", "Monthly"), true
		case "YEARLY":
			return localizedQuickAddText(ctx, "Jährlich", "Yearly"), true
		default:
			return "", false
		}
	}

	if UILanguage(ctx) == "en" {
		switch freq {
		case "DAILY":
			return fmt.Sprintf("Every %d days", interval), true
		case "WEEKLY":
			return fmt.Sprintf("Every %d weeks", interval), true
		case "MONTHLY":
			return fmt.Sprintf("Every %d months", interval), true
		case "YEARLY":
			return fmt.Sprintf("Every %d years", interval), true
		default:
			return "", false
		}
	}

	switch freq {
	case "DAILY":
		return fmt.Sprintf("Alle %d Tage", interval), true
	case "WEEKLY":
		return fmt.Sprintf("Alle %d Wochen", interval), true
	case "MONTHLY":
		return fmt.Sprintf("Alle %d Monate", interval), true
	case "YEARLY":
		return fmt.Sprintf("Alle %d Jahre", interval), true
	default:
		return "", false
	}
}

func quickAddWeekdayName(ctx context.Context, day string) (string, bool) {
	if UILanguage(ctx) == "en" {
		switch day {
		case "MO":
			return "Monday", true
		case "TU":
			return "Tuesday", true
		case "WE":
			return "Wednesday", true
		case "TH":
			return "Thursday", true
		case "FR":
			return "Friday", true
		case "SA":
			return "Saturday", true
		case "SU":
			return "Sunday", true
		default:
			return "", false
		}
	}

	switch day {
	case "MO":
		return "Montag", true
	case "TU":
		return "Dienstag", true
	case "WE":
		return "Mittwoch", true
	case "TH":
		return "Donnerstag", true
	case "FR":
		return "Freitag", true
	case "SA":
		return "Samstag", true
	case "SU":
		return "Sonntag", true
	default:
		return "", false
	}
}

func localizedQuickAddText(ctx context.Context, german string, english string) string {
	if UILanguage(ctx) == "en" {
		return english
	}
	return german
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
