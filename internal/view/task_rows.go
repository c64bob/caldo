package view

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode"

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
	RRule          string
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

type taskRecurrenceEditor struct {
	Editable  bool
	Label     string
	Frequency string
	Interval  string
	ByDay     string
	End       string
	Until     string
	Count     string
}

type taskSelectOption struct {
	Value string
	Label string
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

func taskDeletePath(task TaskRowView) string {
	return taskEditPath(task)
}

func taskDOMID(prefix string, task TaskRowView) string {
	id := strings.TrimSpace(task.ID)
	if id == "" {
		id = "task"
	}
	builder := strings.Builder{}
	builder.WriteString(prefix)
	builder.WriteString("-")
	for _, r := range id {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' {
			builder.WriteRune(r)
			continue
		}
		builder.WriteRune('-')
	}
	return builder.String()
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

func taskCanOpenDetail(task TaskRowView) bool {
	return strings.TrimSpace(task.ID) != "" && task.ServerVersion > 0
}

func taskCanDetailEdit(task TaskRowView) bool {
	return taskCanInlineEdit(task)
}

func taskCanDelete(task TaskRowView) bool {
	return taskCanToggleCompletion(task)
}

func taskDetailStatusMessage(task TaskRowView) string {
	switch strings.ToLower(strings.TrimSpace(task.SyncStatus)) {
	case "pending":
		return "Änderung wird noch gespeichert."
	case "error":
		return "Letzter Schreibversuch ist fehlgeschlagen. Die Aufgabe ist nicht als gespeichert markiert."
	case "conflict":
		return "Für diese Aufgabe liegt ein Konflikt vor. Änderungen sind bis zur Konfliktlösung gesperrt."
	default:
		return ""
	}
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

func taskRecurrence(task TaskRowView) taskRecurrenceEditor {
	rule := strings.TrimSpace(task.RRule)
	if rule == "" {
		return taskRecurrenceEditor{
			Editable:  true,
			Label:     "Keine",
			Frequency: "NONE",
			Interval:  "1",
			ByDay:     "MO",
			End:       "never",
		}
	}

	parts, ok := parseRRuleParts(rule)
	if !ok || model.IsComplexRRule(rule) {
		return taskReadOnlyRecurrence(rule)
	}

	frequency := strings.ToUpper(parts["FREQ"])
	if frequency == "" {
		return taskReadOnlyRecurrence(rule)
	}
	editor := taskRecurrenceEditor{
		Editable:  true,
		Label:     taskRRuleLabel(parts),
		Frequency: frequency,
		Interval:  parts["INTERVAL"],
		ByDay:     "MO",
		End:       "never",
	}
	if editor.Interval == "" {
		editor.Interval = "1"
	}

	if byDay := strings.ToUpper(strings.TrimSpace(parts["BYDAY"])); byDay != "" {
		switch {
		case frequency == "WEEKLY" && byDay == "MO,TU,WE,TH,FR":
			editor.Frequency = "WEEKDAYS"
		case frequency == "WEEKLY" && taskIsSingleSupportedByDay(byDay):
			editor.Frequency = "BYDAY"
			editor.ByDay = byDay
		default:
			return taskReadOnlyRecurrence(rule)
		}
	}

	if until := strings.TrimSpace(parts["UNTIL"]); until != "" {
		editor.End = "until"
		editor.Until = taskRRuleUntilDate(until)
		if editor.Until == "" {
			return taskReadOnlyRecurrence(rule)
		}
	}
	if count := strings.TrimSpace(parts["COUNT"]); count != "" {
		if editor.End == "until" {
			return taskReadOnlyRecurrence(rule)
		}
		editor.End = "count"
		editor.Count = count
	}

	return editor
}

func taskReadOnlyRecurrence(rule string) taskRecurrenceEditor {
	return taskRecurrenceEditor{
		Editable: false,
		Label:    "RRULE: " + strings.TrimSpace(rule),
	}
}

func parseRRuleParts(rule string) (map[string]string, bool) {
	parts := strings.Split(rule, ";")
	result := make(map[string]string, len(parts))
	for _, rawPart := range parts {
		part := strings.TrimSpace(rawPart)
		if part == "" {
			continue
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
		result[key] = value
	}
	return result, true
}

func taskRRuleLabel(parts map[string]string) string {
	frequency := strings.ToUpper(strings.TrimSpace(parts["FREQ"]))
	byDay := strings.ToUpper(strings.TrimSpace(parts["BYDAY"]))
	interval := strings.TrimSpace(parts["INTERVAL"])
	base := "Wiederholt"
	switch frequency {
	case "DAILY":
		base = "Täglich"
	case "WEEKLY":
		if byDay == "MO,TU,WE,TH,FR" {
			base = "Werktags"
		} else if taskIsSingleSupportedByDay(byDay) {
			base = "Wöchentlich " + taskByDayLabel(byDay)
		} else {
			base = "Wöchentlich"
		}
	case "MONTHLY":
		base = "Monatlich"
	case "YEARLY":
		base = "Jährlich"
	}
	if interval != "" && interval != "1" {
		base += " · alle " + interval
	}
	if count := strings.TrimSpace(parts["COUNT"]); count != "" {
		base += " · " + count + " mal"
	}
	if until := taskRRuleUntilDate(parts["UNTIL"]); until != "" {
		base += " · bis " + until
	}
	return base
}

func taskRRuleUntilDate(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	for _, layout := range []string{"20060102T150405Z", "20060102"} {
		parsed, err := time.Parse(layout, trimmed)
		if err == nil {
			return parsed.UTC().Format("2006-01-02")
		}
	}
	return ""
}

func taskIsSingleSupportedByDay(value string) bool {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "MO", "TU", "WE", "TH", "FR", "SA", "SU":
		return true
	default:
		return false
	}
}

func taskByDayOptions() []taskSelectOption {
	return []taskSelectOption{
		{Value: "MO", Label: "Montag"},
		{Value: "TU", Label: "Dienstag"},
		{Value: "WE", Label: "Mittwoch"},
		{Value: "TH", Label: "Donnerstag"},
		{Value: "FR", Label: "Freitag"},
		{Value: "SA", Label: "Samstag"},
		{Value: "SU", Label: "Sonntag"},
	}
}

func taskRecurrenceFrequencyOptions() []taskSelectOption {
	return []taskSelectOption{
		{Value: "NONE", Label: "Keine"},
		{Value: "DAILY", Label: "Täglich"},
		{Value: "WEEKLY", Label: "Wöchentlich"},
		{Value: "WEEKDAYS", Label: "Werktags"},
		{Value: "BYDAY", Label: "Wöchentlich am Wochentag"},
		{Value: "MONTHLY", Label: "Monatlich"},
		{Value: "YEARLY", Label: "Jährlich"},
	}
}

func taskRecurrenceEndOptions() []taskSelectOption {
	return []taskSelectOption{
		{Value: "never", Label: "Nie"},
		{Value: "until", Label: "Bis Datum"},
		{Value: "count", Label: "Nach Anzahl"},
	}
}

func taskByDayLabel(value string) string {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "MO":
		return "Montag"
	case "TU":
		return "Dienstag"
	case "WE":
		return "Mittwoch"
	case "TH":
		return "Donnerstag"
	case "FR":
		return "Freitag"
	case "SA":
		return "Samstag"
	case "SU":
		return "Sonntag"
	default:
		return ""
	}
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
