package view

import (
	"context"
	"encoding/json"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"caldo/internal/model"
)

// TaskRowView contains all fields rendered by the shared task-list row.
type TaskRowView struct {
	ID               string
	ProjectID        string
	Title            string
	Description      string
	ProjectName      string
	LabelNames       string
	DueISODate       string
	TodayISODate     string
	ParentID         string
	ParentTitle      string
	Status           string
	SyncStatus       string
	Priority         int
	HasPriority      bool
	ServerVersion    int
	IsSubtask        bool
	SubtaskCount     int
	OpenSubtaskCount int
	ConflictID       string
	RRule            string
	Attachments      []model.Attachment
	ProjectOptions   []TaskProjectOption
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

type taskDueChip struct {
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

type taskDescriptionSegment struct {
	Text string
	URL  string
	Kind string
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

func taskRowDescriptionVisible(ctx context.Context, task TaskRowView) bool {
	return UITaskNoteDisplay(ctx) != TaskNoteDisplayNone && taskRowDescription(task) != ""
}

func taskRowDescriptionClass(ctx context.Context) string {
	switch UITaskNoteDisplay(ctx) {
	case TaskNoteDisplayFull:
		return "caldo-task-description-full"
	case TaskNoteDisplayOneLine:
		return "caldo-task-description-lines-1"
	default:
		return "caldo-task-description-lines-2"
	}
}

func taskDescriptionMarkdownSegments(description string) []taskDescriptionSegment {
	description = taskDescriptionDisplayText(description)
	if description == "" {
		return nil
	}

	return mergeTaskDescriptionSegments(parseTaskDescriptionInlineMarkdown(description))
}

func taskDescriptionDisplayText(description string) string {
	return strings.TrimSpace(unescapeTaskDescriptionText(description))
}

func unescapeTaskDescriptionText(value string) string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")
	var builder strings.Builder
	builder.Grow(len(value))
	for i := 0; i < len(value); i++ {
		if value[i] != '\\' || i+1 >= len(value) {
			builder.WriteByte(value[i])
			continue
		}
		next := value[i+1]
		switch next {
		case 'n', 'N':
			builder.WriteByte('\n')
		case ',', ';', '\\':
			builder.WriteByte(next)
		default:
			builder.WriteByte(value[i])
			builder.WriteByte(next)
		}
		i++
	}
	return builder.String()
}

func parseTaskDescriptionInlineMarkdown(description string) []taskDescriptionSegment {
	segments := make([]taskDescriptionSegment, 0, 6)
	for len(description) > 0 {
		if description[0] == '\n' {
			segments = append(segments, taskDescriptionSegment{Kind: "break"})
			description = description[1:]
			continue
		}
		if strings.HasPrefix(description, "**") {
			if end := strings.Index(description[2:], "**"); end >= 0 {
				text := description[2 : 2+end]
				if strings.TrimSpace(text) != "" {
					segments = append(segments, taskDescriptionSegment{Text: text, Kind: "strong"})
					description = description[2+end+2:]
					continue
				}
			}
		}
		if description[0] == '*' {
			if end := strings.Index(description[1:], "*"); end >= 0 {
				text := description[1 : 1+end]
				if strings.TrimSpace(text) != "" {
					segments = append(segments, taskDescriptionSegment{Text: text, Kind: "em"})
					description = description[1+end+1:]
					continue
				}
			}
		}
		if description[0] == '`' {
			if end := strings.Index(description[1:], "`"); end >= 0 {
				text := description[1 : 1+end]
				if text != "" {
					segments = append(segments, taskDescriptionSegment{Text: text, Kind: "code"})
					description = description[1+end+1:]
					continue
				}
			}
		}
		if strings.HasPrefix(description, "[") {
			if segment, rest, ok := parseMarkdownLink(description); ok {
				segments = append(segments, segment)
				description = rest
				continue
			}
		}
		if segment, rest, ok := parseBareDescriptionURL(description); ok {
			segments = append(segments, segment)
			description = rest
			continue
		}

		next := nextDescriptionMarkupIndex(description[1:])
		if next < 0 {
			segments = append(segments, taskDescriptionSegment{Text: description})
			break
		}
		next++
		segments = append(segments, taskDescriptionSegment{Text: description[:next]})
		description = description[next:]
	}
	return segments
}

func parseMarkdownLink(description string) (taskDescriptionSegment, string, bool) {
	labelEnd := strings.Index(description, "](")
	if labelEnd <= 1 {
		return taskDescriptionSegment{}, description, false
	}
	urlStart := labelEnd + 2
	urlEnd := strings.Index(description[urlStart:], ")")
	if urlEnd < 0 {
		return taskDescriptionSegment{}, description, false
	}
	urlEnd += urlStart
	text := description[1:labelEnd]
	link := strings.TrimSpace(description[urlStart:urlEnd])
	if text == "" || !descriptionURLIsSafe(link) {
		return taskDescriptionSegment{}, description, false
	}
	return taskDescriptionSegment{Text: text, URL: link, Kind: "link"}, description[urlEnd+1:], true
}

func parseBareDescriptionURL(description string) (taskDescriptionSegment, string, bool) {
	index, _ := nextDescriptionURLIndex(description)
	if index != 0 {
		return taskDescriptionSegment{}, description, false
	}
	candidateEnd := 0
	for candidateEnd < len(description) && !unicode.IsSpace(rune(description[candidateEnd])) {
		candidateEnd++
	}
	candidate := description[:candidateEnd]
	urlText, suffix := trimDescriptionURL(candidate)
	if !descriptionURLIsSafe(urlText) {
		return taskDescriptionSegment{}, description, false
	}
	rest := suffix + description[candidateEnd:]
	return taskDescriptionSegment{Text: urlText, URL: urlText, Kind: "link"}, rest, true
}

func nextDescriptionMarkupIndex(text string) int {
	next := -1
	for _, marker := range []string{"\n", "**", "*", "`", "[", "http://", "https://"} {
		if index := strings.Index(text, marker); index >= 0 && (next < 0 || index < next) {
			next = index
		}
	}
	return next
}

func nextDescriptionURLIndex(text string) (int, int) {
	httpIndex := strings.Index(text, "http://")
	httpsIndex := strings.Index(text, "https://")
	switch {
	case httpIndex < 0:
		return httpsIndex, len("https://")
	case httpsIndex < 0:
		return httpIndex, len("http://")
	case httpIndex < httpsIndex:
		return httpIndex, len("http://")
	default:
		return httpsIndex, len("https://")
	}
}

func trimDescriptionURL(candidate string) (string, string) {
	linkEnd := len(candidate)
	for linkEnd > 0 {
		switch candidate[linkEnd-1] {
		case '.', ',', ';', ':', '!', '?', ')', ']', '}':
			linkEnd--
		default:
			return candidate[:linkEnd], candidate[linkEnd:]
		}
	}
	return candidate, ""
}

func descriptionURLIsSafe(candidate string) bool {
	candidate = strings.TrimSpace(candidate)
	if candidate == "" {
		return false
	}
	parsed, err := url.Parse(candidate)
	if err != nil {
		return false
	}
	if parsed.Host == "" {
		return false
	}
	return parsed.Scheme == "http" || parsed.Scheme == "https"
}

func mergeTaskDescriptionSegments(segments []taskDescriptionSegment) []taskDescriptionSegment {
	merged := make([]taskDescriptionSegment, 0, len(segments))
	for _, segment := range segments {
		if segment.Text == "" && segment.Kind != "break" {
			continue
		}
		last := len(merged) - 1
		if segment.Kind == "" && segment.URL == "" && last >= 0 && merged[last].Kind == "" && merged[last].URL == "" {
			merged[last].Text += segment.Text
			continue
		}
		merged = append(merged, segment)
	}
	return merged
}

func taskIsCompleted(task TaskRowView) bool {
	return strings.EqualFold(strings.TrimSpace(task.Status), "completed")
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

func taskMovePath(task TaskRowView) string {
	return taskEditPath(task) + "/move"
}

func taskDeletePath(task TaskRowView) string {
	return taskEditPath(task)
}

func taskFavoritePath(task TaskRowView) string {
	return "/tasks/" + url.PathEscape(strings.TrimSpace(task.ID)) + "/favorite"
}

func taskSubtaskCreatePath(task TaskRowView) string {
	return "/tasks/" + url.PathEscape(strings.TrimSpace(task.ID)) + "/subtasks"
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
		chips = append(chips, taskRowChip{Label: "Fällig " + taskDisplayDate(due), Class: "caldo-task-chip caldo-task-chip-due"})
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

func taskCanDragMove(task TaskRowView) bool {
	return taskCanToggleCompletion(task) && strings.TrimSpace(task.ProjectID) != ""
}

func taskDragMoveDraggable(task TaskRowView) string {
	if taskCanDragMove(task) {
		return "true"
	}
	return "false"
}

func taskCanCreateSubtask(task TaskRowView) bool {
	return !task.IsSubtask && taskCanToggleCompletion(task)
}

func taskNeedsCompletionDecision(task TaskRowView) bool {
	return !taskIsCompleted(task) && !task.IsSubtask && taskCanToggleCompletion(task) && task.OpenSubtaskCount > 0
}

func taskCanShowFavorite(task TaskRowView) bool {
	return strings.TrimSpace(task.ID) != "" && task.ServerVersion > 0
}

func taskIsFavorite(task TaskRowView) bool {
	if taskPriorityIsHigh(task) {
		return true
	}
	for _, label := range rawTaskLabels(task.LabelNames) {
		if strings.EqualFold(label, model.ReservedFavoriteCategory) {
			return true
		}
	}
	return false
}

func taskPriorityIsHigh(task TaskRowView) bool {
	return task.HasPriority && task.Priority > 0 && task.Priority <= 4
}

func taskFavoriteLabel(task TaskRowView) string {
	if taskIsFavorite(task) {
		return "Favorit entfernen"
	}
	return "Favorit setzen"
}

func taskFavoritePressed(task TaskRowView) string {
	if taskIsFavorite(task) {
		return "true"
	}
	return "false"
}

func taskFavoriteNextValue(task TaskRowView) string {
	if taskIsFavorite(task) {
		return "false"
	}
	return "true"
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

func taskConflictPath(task TaskRowView) string {
	if !strings.EqualFold(strings.TrimSpace(task.SyncStatus), "conflict") {
		return ""
	}
	conflictID := strings.TrimSpace(task.ConflictID)
	if conflictID == "" {
		return ""
	}
	return "/conflicts/" + url.PathEscape(conflictID)
}

func taskUsesConflictLink(task TaskRowView) bool {
	return taskConflictPath(task) != ""
}

func taskMetaChips(task TaskRowView) []taskRowChip {
	chips := make([]taskRowChip, 0, 3)
	if project := strings.TrimSpace(task.ProjectName); project != "" {
		chips = append(chips, taskRowChip{Label: project, Class: "caldo-task-chip"})
	}
	if priority := taskPriorityLabel(task); priority != "" {
		chips = append(chips, taskRowChip{Label: priority, Class: "caldo-task-chip " + taskPriorityClass(task)})
	}
	return chips
}

func taskCheckboxPriorityClass(task TaskRowView) string {
	if taskIsFavorite(task) {
		return "caldo-task-checkbox-priority-high"
	}
	if !task.HasPriority || task.Priority <= 0 {
		return ""
	}
	switch {
	case task.Priority <= 4:
		return "caldo-task-checkbox-priority-high"
	case task.Priority <= 6:
		return "caldo-task-checkbox-priority-medium"
	default:
		return "caldo-task-checkbox-priority-low"
	}
}

func taskDateEditLabel(task TaskRowView) string {
	return "Fälligkeit bearbeiten: " + taskDueStateChip(task).Label
}

func taskRelationshipChips(task TaskRowView) []taskRowChip {
	chips := make([]taskRowChip, 0, 1)
	if task.IsSubtask {
		label := "Unteraufgabe"
		if parentTitle := strings.TrimSpace(task.ParentTitle); parentTitle != "" {
			label = "Unteraufgabe von " + parentTitle
		}
		chips = append(chips, taskRowChip{Label: label, Class: "caldo-task-chip caldo-task-chip-subtask"})
		return chips
	}
	if task.SubtaskCount > 0 {
		chips = append(chips, taskRowChip{Label: taskSubtaskCountLabel(task.SubtaskCount), Class: "caldo-task-chip caldo-task-chip-subtask"})
	}
	return chips
}

func taskSubtaskCountLabel(count int) string {
	if count == 1 {
		return "1 Unteraufgabe"
	}
	return strconv.Itoa(count) + " Unteraufgaben"
}

func taskSubtaskProgressLabel(task TaskRowView) string {
	if task.SubtaskCount == 0 {
		return ""
	}
	completed := task.SubtaskCount - task.OpenSubtaskCount
	return strconv.Itoa(completed) + "/" + strconv.Itoa(task.SubtaskCount) + " erledigt"
}

func taskSubtaskProgressPercent(task TaskRowView) int {
	if task.SubtaskCount == 0 {
		return 0
	}
	completed := task.SubtaskCount - task.OpenSubtaskCount
	return (completed * 100) / task.SubtaskCount
}

func taskOpenSubtaskCountLabel(count int) string {
	if count == 1 {
		return "1 offene Unteraufgabe"
	}
	return strconv.Itoa(count) + " offene Unteraufgaben"
}

func taskCompleteWithSubtasksLabel(task TaskRowView) string {
	return "Aufgabe und " + taskOpenSubtaskCountLabel(task.OpenSubtaskCount) + " erledigen"
}

func taskHasDirectSubtasks(task TaskRowView) bool {
	return !task.IsSubtask && task.SubtaskCount > 0
}

func taskDeleteSubtaskWarning(task TaskRowView) string {
	return "Diese Aufgabe hat " + taskSubtaskCountLabel(task.SubtaskCount) + ". Die Elternaufgabe und alle direkten Unteraufgaben werden einzeln gelöscht."
}

func taskDeleteSubmitLabel(task TaskRowView) string {
	if taskHasDirectSubtasks(task) {
		return "Aufgabe und Unteraufgaben löschen"
	}
	return "Endgültig löschen"
}

func taskDueStateChip(task TaskRowView) taskDueChip {
	due := strings.TrimSpace(task.DueISODate)
	if due == "" {
		return taskDueChip{Label: "Ohne Datum", Class: "caldo-task-chip caldo-task-chip-due caldo-task-chip-due-none"}
	}

	displayDue := taskDisplayDate(due)
	today := taskTodayISODate(task)
	switch {
	case due < today:
		return taskDueChip{Label: displayDue, Class: "caldo-task-chip caldo-task-chip-due caldo-task-chip-due-overdue"}
	case due == today:
		return taskDueChip{Label: "Heute", Class: "caldo-task-chip caldo-task-chip-due caldo-task-chip-due-today"}
	default:
		return taskDueChip{Label: "Fällig " + displayDue, Class: "caldo-task-chip caldo-task-chip-due caldo-task-chip-due-future"}
	}
}

func taskDisplayDate(isoDate string) string {
	trimmed := strings.TrimSpace(isoDate)
	parsed, err := time.Parse("2006-01-02", trimmed)
	if err != nil {
		return trimmed
	}
	return parsed.Format("02.01.2006")
}

func taskTodayISODate(task TaskRowView) string {
	today := strings.TrimSpace(task.TodayISODate)
	if today != "" {
		return today
	}
	return time.Now().UTC().Format("2006-01-02")
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

func taskDetailPriorityOptions(task TaskRowView) []taskPriorityOption {
	options := []taskPriorityOption{{Value: "", Label: "Keine"}}
	selected := taskPriorityValue(task)
	skipCanonical := ""
	if selected != "" {
		canonical := taskPriorityCanonicalValue(task.Priority)
		if selected != canonical {
			options = append(options, taskPriorityOption{Value: selected, Label: taskPriorityLabel(task)})
			skipCanonical = canonical
		}
	}
	for _, option := range []taskPriorityOption{
		{Value: "1", Label: "P1 Hoch"},
		{Value: "5", Label: "P2 Mittel"},
		{Value: "9", Label: "P3 Niedrig"},
	} {
		if option.Value == skipCanonical {
			continue
		}
		options = append(options, option)
	}
	return options
}

func taskPriorityCanonicalValue(priority int) string {
	switch {
	case priority <= 0:
		return ""
	case priority <= 4:
		return "1"
	case priority <= 6:
		return "5"
	default:
		return "9"
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
		return "P1 Hoch"
	case task.Priority <= 6:
		return "P2 Mittel"
	default:
		return "P3 Niedrig"
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
	labels := rawTaskLabels(raw)
	filtered := labels[:0]
	for _, label := range labels {
		if strings.EqualFold(label, model.ReservedFavoriteCategory) {
			continue
		}
		filtered = append(filtered, label)
	}
	sort.Slice(filtered, func(i, j int) bool {
		left := strings.ToLower(filtered[i])
		right := strings.ToLower(filtered[j])
		if left == right {
			return filtered[i] < filtered[j]
		}
		return left < right
	})
	return filtered
}

func rawTaskLabels(raw string) []string {
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ';' || r == '\n' || r == '\t'
	})
	labels := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		label := strings.TrimSpace(part)
		if label == "" {
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
