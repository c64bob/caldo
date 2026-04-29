package model

import (
	"fmt"
	"strings"
	"time"
)

// VTODOPatch describes explicit VTODO field changes.
type VTODOPatch struct {
	Summary        *string
	Description    *string
	Status         *string
	Priority       *int
	RRule          *string
	DueDate        *string
	DueAt          *time.Time
	Categories     []string
	CompletedAt    *time.Time
	ClearDue       bool
	ClearPriority  bool
	ClearCompleted bool
}

// PatchVTODO applies explicit known-field changes to a raw VTODO payload.
func PatchVTODO(raw string, patch VTODOPatch) string {
	if !hasPatchChanges(patch) {
		return raw
	}

	lineBreak := "\n"
	if strings.Contains(raw, "\r\n") {
		lineBreak = "\r\n"
	}

	lines := unfoldICalendarLines(raw)
	start, end := findFirstVTODOBounds(lines)
	if start < 0 || end <= start {
		return raw
	}

	prefix := append([]string(nil), lines[:start+1]...)
	body := lines[start+1 : end]
	suffix := append([]string(nil), lines[end:]...)

	filtered := filterTopLevelKnownFields(body, patch)
	patchedFields := buildPatchedFieldLines(patch)
	propertyLines, nestedAndTrailing := splitBeforeFirstTopLevelNestedComponent(filtered)
	patched := append(prefix, propertyLines...)
	patched = append(patched, patchedFields...)
	patched = append(patched, nestedAndTrailing...)
	patched = append(patched, suffix...)
	return strings.Join(patched, lineBreak)
}

func hasPatchChanges(patch VTODOPatch) bool {
	return patch.Summary != nil ||
		patch.Description != nil ||
		patch.Status != nil ||
		patch.Priority != nil ||
		patch.RRule != nil ||
		patch.DueDate != nil ||
		patch.DueAt != nil ||
		patch.ClearDue ||
		patch.Categories != nil ||
		patch.CompletedAt != nil ||
		patch.ClearPriority ||
		patch.ClearCompleted
}

func findFirstVTODOBounds(lines []string) (int, int) {
	start := -1
	for i, line := range lines {
		name, value, _, ok := splitPropertyLine(line)
		if !ok || name != "BEGIN" {
			continue
		}
		if strings.EqualFold(value, "VTODO") {
			start = i
			break
		}
	}
	if start < 0 {
		return -1, -1
	}

	depth := 0
	for i := start; i < len(lines); i++ {
		name, value, _, ok := splitPropertyLine(lines[i])
		if !ok {
			continue
		}
		if name == "BEGIN" {
			depth++
			continue
		}
		if name == "END" {
			depth--
			if depth == 0 && strings.EqualFold(value, "VTODO") {
				return start, i
			}
		}
	}
	return -1, -1
}

func filterTopLevelKnownFields(body []string, patch VTODOPatch) []string {
	filtered := make([]string, 0, len(body))
	depth := 0

	for _, line := range body {
		name, _, _, ok := splitPropertyLine(line)
		if ok {
			if name == "BEGIN" {
				depth++
				filtered = append(filtered, line)
				continue
			}
			if name == "END" {
				depth--
				filtered = append(filtered, line)
				continue
			}

			if depth == 0 && shouldReplaceProperty(name, patch) {
				continue
			}
		}
		filtered = append(filtered, line)
	}

	return filtered
}

func shouldReplaceProperty(name string, patch VTODOPatch) bool {
	switch name {
	case "SUMMARY":
		return patch.Summary != nil
	case "DESCRIPTION":
		return patch.Description != nil
	case "STATUS":
		return patch.Status != nil
	case "PRIORITY":
		return patch.Priority != nil || patch.ClearPriority
	case "RRULE":
		return patch.RRule != nil
	case "DUE":
		return patch.DueDate != nil || patch.DueAt != nil || patch.ClearDue
	case "CATEGORIES":
		return patch.Categories != nil
	case "COMPLETED":
		return patch.CompletedAt != nil || patch.ClearCompleted
	default:
		return false
	}
}

func buildPatchedFieldLines(patch VTODOPatch) []string {
	lines := make([]string, 0, 9)
	if patch.Summary != nil && strings.TrimSpace(*patch.Summary) != "" {
		lines = append(lines, "SUMMARY:"+strings.TrimSpace(*patch.Summary))
	}
	if patch.Description != nil && strings.TrimSpace(*patch.Description) != "" {
		lines = append(lines, "DESCRIPTION:"+strings.TrimSpace(*patch.Description))
	}
	if patch.Status != nil && strings.TrimSpace(*patch.Status) != "" {
		lines = append(lines, "STATUS:"+strings.ToUpper(strings.TrimSpace(*patch.Status)))
	}
	if patch.CompletedAt != nil {
		lines = append(lines, "COMPLETED:"+patch.CompletedAt.UTC().Format("20060102T150405Z"))
	}
	if patch.DueDate != nil && strings.TrimSpace(*patch.DueDate) != "" {
		if date, err := time.Parse("2006-01-02", strings.TrimSpace(*patch.DueDate)); err == nil {
			lines = append(lines, "DUE;VALUE=DATE:"+date.UTC().Format("20060102"))
		}
	}
	if patch.DueAt != nil {
		lines = append(lines, "DUE:"+patch.DueAt.UTC().Format("20060102T150405Z"))
	}
	if patch.Priority != nil {
		lines = append(lines, "PRIORITY:"+intToString(*patch.Priority))
	}
	if patch.RRule != nil && strings.TrimSpace(*patch.RRule) != "" {
		lines = append(lines, "RRULE:"+strings.TrimSpace(*patch.RRule))
	}
	if patch.Categories != nil {
		labels := make([]string, 0, len(patch.Categories))
		for _, category := range patch.Categories {
			category = strings.TrimSpace(category)
			if category != "" {
				labels = append(labels, category)
			}
		}
		if len(labels) > 0 {
			lines = append(lines, fmt.Sprintf("CATEGORIES:%s", strings.Join(labels, ",")))
		}
	}
	return lines
}

func splitBeforeFirstTopLevelNestedComponent(lines []string) ([]string, []string) {
	depth := 0
	for i, line := range lines {
		name, _, _, ok := splitPropertyLine(line)
		if !ok {
			continue
		}
		if name == "BEGIN" && depth == 0 {
			return lines[:i], lines[i:]
		}
		if name == "BEGIN" {
			depth++
			continue
		}
		if name == "END" && depth > 0 {
			depth--
		}
	}
	return lines, nil
}

func intToString(value int) string {
	if value == 0 {
		return "0"
	}
	negative := value < 0
	if negative {
		value = -value
	}

	buf := make([]byte, 0, 12)
	for value > 0 {
		buf = append(buf, byte('0'+(value%10)))
		value /= 10
	}
	if negative {
		buf = append(buf, '-')
	}

	for i, j := 0, len(buf)-1; i < j; i, j = i+1, j-1 {
		buf[i], buf[j] = buf[j], buf[i]
	}
	return string(buf)
}
