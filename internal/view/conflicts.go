package view

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"
	"time"

	"caldo/internal/db"
	"caldo/internal/model"
)

type conflictComparisonRow struct {
	Field  string
	Label  string
	Base   conflictComparisonCell
	Local  conflictComparisonCell
	Remote conflictComparisonCell
}

type conflictComparisonCell struct {
	Key     string
	Value   string
	Present bool
	Changed bool
}

type conflictRawVersion struct {
	Label string
	Raw   string
}

type conflictManualField struct {
	Name    string
	Label   string
	Options []conflictManualOption
	Manual  conflictManualInput
}

type conflictManualOption struct {
	Value        string
	Label        string
	DisplayValue string
	Present      bool
	Changed      bool
	Selected     bool
}

type conflictManualInput struct {
	Kind         string
	Value        string
	DisplayValue string
	EmptyLabel   string
	Options      []conflictManualInputOption
}

type conflictManualInputOption struct {
	Value    string
	Label    string
	Selected bool
}

type conflictManualPreviewRow struct {
	Name  string
	Label string
	Value string
}

type conflictSplitPreview struct {
	Role        string
	Label       string
	Description string
	Title       string
	UID         string
	Project     string
	Status      string
	Due         string
	Labels      string
	Parent      string
}

func conflictTypeLabel(conflictType string) string {
	switch strings.ToLower(strings.TrimSpace(conflictType)) {
	case "field_conflict":
		return "Feldkonflikt"
	case "edit_delete":
		return "Lokal geändert, remote gelöscht"
	case "delete_edit":
		return "Lokal gelöscht, remote geändert"
	default:
		if trimmed := strings.TrimSpace(conflictType); trimmed != "" {
			return trimmed
		}
		return "Konflikt"
	}
}

func conflictTypeBadgeClass(conflictType string) string {
	base := "caldo-badge caldo-conflict-type-badge"
	switch strings.ToLower(strings.TrimSpace(conflictType)) {
	case "edit_delete", "delete_edit":
		return base + " caldo-conflict-type-badge-danger"
	case "field_conflict":
		return base + " caldo-conflict-type-badge-warning"
	default:
		return base
	}
}

func conflictListCountLabel(count int) string {
	if count == 1 {
		return "1 offener Konflikt"
	}
	return strconv.Itoa(count) + " offene Konflikte"
}

func conflictCreatedAtLabel(createdAt time.Time) string {
	if createdAt.IsZero() {
		return "Erkannt unbekannt"
	}
	return "Erkannt " + createdAt.UTC().Format("2006-01-02 15:04 UTC")
}

func conflictNextActionLabel(conflictType string) string {
	switch strings.ToLower(strings.TrimSpace(conflictType)) {
	case "field_conflict":
		return "Felder vergleichen und Zielversion wählen"
	case "edit_delete":
		return "Lokale Änderung prüfen oder Remote-Löschung übernehmen"
	case "delete_edit":
		return "Remote-Änderung prüfen oder lokale Löschung bestätigen"
	default:
		return "Konflikt prüfen und Auflösung wählen"
	}
}

func conflictProjectLabel(projectName string) string {
	if trimmed := strings.TrimSpace(projectName); trimmed != "" {
		return trimmed
	}
	return "Ohne Projekt"
}

func conflictTaskTitle(title string) string {
	if trimmed := strings.TrimSpace(title); trimmed != "" {
		return trimmed
	}
	return "(gelöschte Aufgabe)"
}

func conflictHasBase(conflict db.ConflictDetail) bool {
	return conflict.BaseVTODO.Valid && strings.TrimSpace(conflict.BaseVTODO.String) != ""
}

func conflictComparisonRows(conflict db.ConflictDetail) []conflictComparisonRow {
	baseFields, basePresent := conflictParsedFields(conflict.BaseVTODO)
	localFields, localPresent := conflictParsedFields(conflict.LocalVTODO)
	remoteFields, remotePresent := conflictParsedFields(conflict.RemoteVTODO)

	specs := []struct {
		key   string
		label string
		value func(model.VTODOFields) string
	}{
		{key: "title", label: "Titel", value: conflictTitleValue},
		{key: "description", label: "Beschreibung", value: conflictDescriptionValue},
		{key: "status", label: "Status", value: conflictStatusValue},
		{key: "due", label: "Fälligkeit", value: conflictDueValue},
		{key: "priority", label: "Priorität", value: conflictPriorityValue},
		{key: "labels", label: "Labels", value: conflictLabelsValue},
		{key: "favorite", label: "Favorit", value: conflictFavoriteValue},
		{key: "recurrence", label: "Wiederholung", value: conflictRecurrenceValue},
		{key: "attachments", label: "Anhänge", value: conflictAttachmentsValue},
		{key: "parent", label: "Unteraufgabe", value: conflictParentValue},
	}

	rows := make([]conflictComparisonRow, 0, len(specs))
	for _, spec := range specs {
		baseValue := spec.value(baseFields)
		localValue := spec.value(localFields)
		remoteValue := spec.value(remoteFields)
		localChanged, remoteChanged := conflictComparisonChanged(basePresent, localPresent, remotePresent, baseValue, localValue, remoteValue)
		base := conflictCell("base-"+spec.key, basePresent, baseValue, false)
		rows = append(rows, conflictComparisonRow{
			Field:  spec.key,
			Label:  spec.label,
			Base:   base,
			Local:  conflictCell("local-"+spec.key, localPresent, localValue, localChanged),
			Remote: conflictCell("remote-"+spec.key, remotePresent, remoteValue, remoteChanged),
		})
	}
	rows = append([]conflictComparisonRow{conflictProjectComparisonRow(conflict, basePresent, localPresent, remotePresent)}, rows...)
	return rows
}

func conflictProjectComparisonRow(conflict db.ConflictDetail, basePresent bool, localPresent bool, remotePresent bool) conflictComparisonRow {
	project := conflictProjectLabel(conflict.ProjectName)
	return conflictComparisonRow{
		Field:  "project",
		Label:  "Projekt",
		Base:   conflictCell("base-project", basePresent, project, false),
		Local:  conflictCell("local-project", localPresent, project, false),
		Remote: conflictCell("remote-project", remotePresent, project, false),
	}
}

func conflictComparisonChanged(basePresent bool, localPresent bool, remotePresent bool, baseValue string, localValue string, remoteValue string) (bool, bool) {
	if basePresent {
		return localPresent && localValue != baseValue, remotePresent && remoteValue != baseValue
	}
	if localPresent && remotePresent && localValue != remoteValue {
		return true, true
	}
	return false, false
}

func conflictComparisonRowChanged(row conflictComparisonRow) bool {
	return row.Base.Changed || row.Local.Changed || row.Remote.Changed
}

func conflictComparisonRowClass(row conflictComparisonRow) string {
	if conflictComparisonRowChanged(row) {
		return "caldo-conflict-row-changed"
	}
	return "caldo-conflict-row-unchanged"
}

func conflictComparisonRowState(row conflictComparisonRow) string {
	if conflictComparisonRowChanged(row) {
		return "changed"
	}
	return "unchanged"
}

func conflictRawVersions(conflict db.ConflictDetail) []conflictRawVersion {
	versions := make([]conflictRawVersion, 0, 3)
	if conflict.BaseVTODO.Valid && strings.TrimSpace(conflict.BaseVTODO.String) != "" {
		versions = append(versions, conflictRawVersion{Label: "Base", Raw: conflict.BaseVTODO.String})
	}
	if conflict.LocalVTODO.Valid && strings.TrimSpace(conflict.LocalVTODO.String) != "" {
		versions = append(versions, conflictRawVersion{Label: "Lokal", Raw: conflict.LocalVTODO.String})
	}
	if conflict.RemoteVTODO.Valid && strings.TrimSpace(conflict.RemoteVTODO.String) != "" {
		versions = append(versions, conflictRawVersion{Label: "Remote", Raw: conflict.RemoteVTODO.String})
	}
	return versions
}

func conflictResolvePath(conflict db.ConflictDetail) string {
	return "/conflicts/" + url.PathEscape(conflict.ID) + "/resolve"
}

func conflictResolveCSRFHeaders(ctx context.Context) string {
	encoded, err := json.Marshal(map[string]string{"X-CSRF-Token": CSRFToken(ctx)})
	if err != nil {
		return "{}"
	}
	return string(encoded)
}

func conflictCanResolveExistingTask(conflict db.ConflictDetail) bool {
	return conflict.TaskID.Valid && strings.TrimSpace(conflict.TaskID.String) != ""
}

func conflictHasLocalVersion(conflict db.ConflictDetail) bool {
	return conflict.LocalVTODO.Valid && strings.TrimSpace(conflict.LocalVTODO.String) != ""
}

func conflictHasRemoteVersion(conflict db.ConflictDetail) bool {
	return conflict.RemoteVTODO.Valid && strings.TrimSpace(conflict.RemoteVTODO.String) != ""
}

func conflictCanSplit(conflict db.ConflictDetail) bool {
	return conflictCanResolveExistingTask(conflict) && conflictHasLocalVersion(conflict) && conflictHasRemoteVersion(conflict)
}

func conflictManualFields(conflict db.ConflictDetail) []conflictManualField {
	baseFields, basePresent := conflictParsedFields(conflict.BaseVTODO)
	localFields, localPresent := conflictParsedFields(conflict.LocalVTODO)
	remoteFields, remotePresent := conflictParsedFields(conflict.RemoteVTODO)
	rowsByField := make(map[string]conflictComparisonRow)
	for _, row := range conflictComparisonRows(conflict) {
		rowsByField[row.Field] = row
	}
	manualDefaults := conflictManualDefaultFields(baseFields, basePresent, localFields, localPresent, remoteFields, remotePresent)
	fields := []conflictManualField{}
	for _, field := range []struct {
		name  string
		label string
	}{
		{name: "title", label: "Titel"},
		{name: "description", label: "Beschreibung"},
		{name: "due", label: "Fälligkeit"},
		{name: "priority", label: "Priorität"},
		{name: "labels", label: "Labels"},
		{name: "status", label: "Status"},
		{name: "parent", label: "Unteraufgaben"},
	} {
		row, ok := rowsByField[field.name]
		if !ok {
			continue
		}
		options := conflictManualOptions(row, basePresent, localPresent, remotePresent)
		if len(options) == 0 {
			continue
		}
		fields = append(fields, conflictManualField{
			Name:    field.name,
			Label:   field.label,
			Options: options,
			Manual:  conflictManualInputForField(field.name, manualDefaults),
		})
	}
	return fields
}

func conflictManualDefaultFields(baseFields model.VTODOFields, basePresent bool, localFields model.VTODOFields, localPresent bool, remoteFields model.VTODOFields, remotePresent bool) model.VTODOFields {
	if remotePresent {
		return remoteFields
	}
	if localPresent {
		return localFields
	}
	if basePresent {
		return baseFields
	}
	return model.VTODOFields{}
}

func conflictManualInputForField(field string, fields model.VTODOFields) conflictManualInput {
	input := conflictManualInput{
		Kind:         "text",
		Value:        conflictManualInputValue(field, fields),
		DisplayValue: conflictManualDisplayValue(field, fields),
		EmptyLabel:   conflictManualEmptyLabel(field),
	}
	switch field {
	case "description":
		input.Kind = "textarea"
	case "status":
		input.Kind = "select"
		input.Options = conflictManualStatusOptions(input.Value)
	case "priority":
		input.Kind = "select"
		input.Options = conflictManualPriorityOptions(input.Value)
	}
	return input
}

func conflictManualInputValue(field string, fields model.VTODOFields) string {
	switch field {
	case "title":
		return strings.TrimSpace(fields.Title)
	case "description":
		return strings.TrimSpace(fields.Description)
	case "due":
		if fields.DueDate != nil {
			return strings.TrimSpace(*fields.DueDate)
		}
		if fields.DueAt != nil && !fields.DueAt.IsZero() {
			return fields.DueAt.UTC().Format(time.RFC3339)
		}
	case "priority":
		if fields.Priority != nil && *fields.Priority > 0 {
			return strconv.Itoa(*fields.Priority)
		}
	case "labels":
		return strings.Join(conflictUserLabels(fields.Categories), ", ")
	case "status":
		status := strings.ToLower(strings.TrimSpace(fields.Status))
		if status == "" {
			return "needs-action"
		}
		return status
	case "parent":
		return strings.TrimSpace(fields.ParentUID)
	}
	return ""
}

func conflictManualDisplayValue(field string, fields model.VTODOFields) string {
	switch field {
	case "title":
		return conflictTitleValue(fields)
	case "description":
		return conflictDescriptionValue(fields)
	case "due":
		return conflictDueValue(fields)
	case "priority":
		return conflictPriorityValue(fields)
	case "labels":
		return conflictLabelsValue(fields)
	case "status":
		return conflictStatusValue(fields)
	case "parent":
		return conflictParentValue(fields)
	default:
		return ""
	}
}

func conflictManualEmptyLabel(field string) string {
	switch field {
	case "title":
		return "Ohne Titel"
	case "description":
		return "Keine Beschreibung"
	case "due":
		return "Keine Fälligkeit"
	case "priority":
		return "Keine Priorität"
	case "labels":
		return "Keine Labels"
	case "parent":
		return "Keine Unteraufgaben-Beziehung"
	default:
		return ""
	}
}

func conflictManualStatusOptions(selected string) []conflictManualInputOption {
	selected = strings.ToLower(strings.TrimSpace(selected))
	if selected == "" {
		selected = "needs-action"
	}
	options := []conflictManualInputOption{
		{Value: "needs-action", Label: "Offen"},
		{Value: "in-process", Label: "In Arbeit"},
		{Value: "completed", Label: "Erledigt"},
		{Value: "cancelled", Label: "Abgebrochen"},
	}
	for i := range options {
		options[i].Selected = options[i].Value == selected
	}
	return options
}

func conflictManualPriorityOptions(selected string) []conflictManualInputOption {
	options := make([]conflictManualInputOption, 0, len(taskPriorityOptions()))
	for _, option := range taskPriorityOptions() {
		options = append(options, conflictManualInputOption{
			Value:    option.Value,
			Label:    option.Label,
			Selected: option.Value == selected,
		})
	}
	return options
}

func conflictManualPreviewRows(fields []conflictManualField) []conflictManualPreviewRow {
	rows := make([]conflictManualPreviewRow, 0, len(fields))
	for _, field := range fields {
		value := field.Manual.DisplayValue
		for _, option := range field.Options {
			if option.Selected {
				value = option.DisplayValue
				break
			}
		}
		rows = append(rows, conflictManualPreviewRow{Name: field.Name, Label: field.Label, Value: value})
	}
	return rows
}

func conflictProjectPreviewValue(conflict db.ConflictDetail, projectOptions []TaskProjectOption) string {
	for _, project := range projectOptions {
		if conflictProjectSelected(conflict, project) {
			return project.Name
		}
	}
	return conflictProjectLabel(conflict.ProjectName)
}

func conflictManualOptions(row conflictComparisonRow, basePresent bool, localPresent bool, remotePresent bool) []conflictManualOption {
	options := []conflictManualOption{}
	defaultValue := "remote"
	if !remotePresent && localPresent {
		defaultValue = "local"
	}
	if !remotePresent && !localPresent && basePresent {
		defaultValue = "base"
	}
	if basePresent {
		options = append(options, conflictManualOptionFromCell("base", "Base", row.Base, defaultValue == "base"))
	}
	if localPresent {
		options = append(options, conflictManualOptionFromCell("local", "Lokal", row.Local, defaultValue == "local"))
	}
	if remotePresent {
		options = append(options, conflictManualOptionFromCell("remote", "Remote", row.Remote, defaultValue == "remote"))
	}
	return options
}

func conflictManualOptionFromCell(value string, label string, cell conflictComparisonCell, selected bool) conflictManualOption {
	return conflictManualOption{
		Value:        value,
		Label:        label,
		DisplayValue: cell.Value,
		Present:      cell.Present,
		Changed:      cell.Changed,
		Selected:     selected,
	}
}

func conflictProjectSelected(conflict db.ConflictDetail, option TaskProjectOption) bool {
	if !conflict.ProjectID.Valid {
		return false
	}
	return strings.TrimSpace(conflict.ProjectID.String) == strings.TrimSpace(option.ID)
}

func conflictSplitLocalPreview(conflict db.ConflictDetail) conflictSplitPreview {
	fields, _ := conflictParsedFields(conflict.LocalVTODO)
	return conflictSplitPreview{
		Role:        "local",
		Label:       "Aufgabe 1: Lokale Variante bleibt",
		Description: "Die lokale Aufgabe bleibt als bestehende Aufgabe im aktuellen Projekt erhalten.",
		Title:       conflictTitleValue(fields),
		UID:         conflictSplitUIDLabel(fields.UID, "UID bleibt erhalten"),
		Project:     conflictProjectLabel(conflict.ProjectName),
		Status:      conflictStatusValue(fields),
		Due:         conflictDueValue(fields),
		Labels:      conflictLabelsValue(fields),
		Parent:      conflictSplitLocalParentLabel(fields),
	}
}

func conflictSplitRemotePreview(conflict db.ConflictDetail) conflictSplitPreview {
	fields, _ := conflictParsedFields(conflict.RemoteVTODO)
	return conflictSplitPreview{
		Role:        "remote",
		Label:       "Aufgabe 2: Remote-Variante wird neu angelegt",
		Description: "Die entfernte Variante wird als zweite Aufgabe im selben Projekt gespeichert.",
		Title:       conflictTitleValue(fields),
		UID:         "Neue UID beim Speichern",
		Project:     conflictProjectLabel(conflict.ProjectName),
		Status:      conflictStatusValue(fields),
		Due:         conflictDueValue(fields),
		Labels:      conflictLabelsValue(fields),
		Parent:      conflictSplitRemoteParentLabel(fields),
	}
}

func conflictSplitUIDLabel(uid string, fallback string) string {
	if trimmed := strings.TrimSpace(uid); trimmed != "" {
		return "UID " + trimmed
	}
	return fallback
}

func conflictSplitLocalParentLabel(fields model.VTODOFields) string {
	if parent := strings.TrimSpace(fields.ParentUID); parent != "" {
		return "Unteraufgabe bleibt: " + parent
	}
	return "Keine Unteraufgaben-Beziehung"
}

func conflictSplitRemoteParentLabel(fields model.VTODOFields) string {
	if strings.TrimSpace(fields.ParentUID) != "" {
		return "Parent-Beziehung wird entfernt"
	}
	return "Keine Unteraufgaben-Beziehung"
}

func conflictParsedFields(raw sql.NullString) (model.VTODOFields, bool) {
	if !raw.Valid || strings.TrimSpace(raw.String) == "" {
		return model.VTODOFields{}, false
	}
	return model.ParseVTODOFields(raw.String), true
}

func conflictCell(key string, present bool, value string, changed bool) conflictComparisonCell {
	if !present {
		return conflictComparisonCell{
			Key:     key,
			Value:   "Nicht vorhanden (gelöscht)",
			Present: false,
			Changed: changed,
		}
	}
	return conflictComparisonCell{
		Key:     key,
		Value:   value,
		Present: true,
		Changed: changed,
	}
}

func conflictCellClass(cell conflictComparisonCell) string {
	classes := []string{"caldo-conflict-value"}
	if !cell.Present {
		classes = append(classes, "caldo-conflict-value-missing")
	}
	if cell.Changed {
		classes = append(classes, "caldo-conflict-value-changed")
	}
	return strings.Join(classes, " ")
}

func conflictManualOptionClass(option conflictManualOption) string {
	classes := []string{"caldo-conflict-manual-option"}
	if !option.Present {
		classes = append(classes, "caldo-conflict-manual-option-missing")
	}
	if option.Changed {
		classes = append(classes, "caldo-conflict-manual-option-changed")
	}
	return strings.Join(classes, " ")
}

func conflictTitleValue(fields model.VTODOFields) string {
	if trimmed := strings.TrimSpace(fields.Title); trimmed != "" {
		return trimmed
	}
	return "Ohne Titel"
}

func conflictDescriptionValue(fields model.VTODOFields) string {
	if trimmed := strings.TrimSpace(fields.Description); trimmed != "" {
		return trimmed
	}
	return "Keine Beschreibung"
}

func conflictStatusValue(fields model.VTODOFields) string {
	switch strings.ToLower(strings.TrimSpace(fields.Status)) {
	case "completed":
		return "Erledigt"
	case "cancelled":
		return "Abgebrochen"
	case "in-process":
		return "In Arbeit"
	case "needs-action", "":
		return "Offen"
	default:
		return strings.TrimSpace(fields.Status)
	}
}

func conflictDueValue(fields model.VTODOFields) string {
	if fields.DueDate != nil && strings.TrimSpace(*fields.DueDate) != "" {
		return strings.TrimSpace(*fields.DueDate)
	}
	if fields.DueAt != nil && !fields.DueAt.IsZero() {
		return fields.DueAt.UTC().Format("2006-01-02 15:04 UTC")
	}
	return "Keine Fälligkeit"
}

func conflictPriorityValue(fields model.VTODOFields) string {
	if fields.Priority == nil || *fields.Priority <= 0 {
		return "Keine Priorität"
	}
	switch {
	case *fields.Priority <= 4:
		return "P1 Hoch (" + intString(*fields.Priority) + ")"
	case *fields.Priority <= 6:
		return "P2 Mittel (" + intString(*fields.Priority) + ")"
	default:
		return "P3 Niedrig (" + intString(*fields.Priority) + ")"
	}
}

func conflictLabelsValue(fields model.VTODOFields) string {
	labels := conflictUserLabels(fields.Categories)
	if len(labels) == 0 {
		return "Keine Labels"
	}
	return strings.Join(labels, ", ")
}

func conflictFavoriteValue(fields model.VTODOFields) string {
	for _, category := range fields.Categories {
		if strings.EqualFold(strings.TrimSpace(category), model.ReservedFavoriteCategory) {
			return "Ja"
		}
	}
	return "Nein"
}

func conflictRecurrenceValue(fields model.VTODOFields) string {
	rule := strings.TrimSpace(fields.RRule)
	if rule == "" {
		return "Keine Wiederholung"
	}
	parts, ok := parseRRuleParts(rule)
	if ok && !model.IsComplexRRule(rule) {
		return taskRRuleLabel(parts)
	}
	return "RRULE: " + rule
}

func conflictAttachmentsValue(fields model.VTODOFields) string {
	if len(fields.Attachments) == 0 {
		return "Keine Anhänge"
	}
	values := make([]string, 0, len(fields.Attachments))
	for _, attachment := range fields.Attachments {
		if attachment.IsExternalURL {
			values = append(values, attachment.Value)
			continue
		}
		values = append(values, "Anhang vorhanden (inline/binary)")
	}
	return strings.Join(values, ", ")
}

func conflictParentValue(fields model.VTODOFields) string {
	if parent := strings.TrimSpace(fields.ParentUID); parent != "" {
		return "Unteraufgabe von " + parent
	}
	return "Keine Unteraufgaben-Beziehung"
}

func conflictUserLabels(categories []string) []string {
	labels := make([]string, 0, len(categories))
	seen := make(map[string]struct{}, len(categories))
	for _, category := range categories {
		category = strings.TrimSpace(category)
		if category == "" || strings.EqualFold(category, model.ReservedFavoriteCategory) {
			continue
		}
		key := strings.ToLower(category)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		labels = append(labels, category)
	}
	return labels
}

func intString(value int) string {
	return strconv.Itoa(value)
}
