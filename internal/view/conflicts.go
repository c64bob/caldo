package view

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"

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
}

type conflictManualOption struct {
	Value        string
	Label        string
	DisplayValue string
	Present      bool
	Changed      bool
	Selected     bool
}

type conflictSplitPreview struct {
	Label   string
	Title   string
	UID     string
	Project string
	Status  string
	Due     string
	Parent  string
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
		base := conflictCell("base-"+spec.key, basePresent, spec.value(baseFields), false)
		rows = append(rows, conflictComparisonRow{
			Field:  spec.key,
			Label:  spec.label,
			Base:   base,
			Local:  conflictCell("local-"+spec.key, localPresent, spec.value(localFields), basePresent && localPresent && spec.value(localFields) != base.Value),
			Remote: conflictCell("remote-"+spec.key, remotePresent, spec.value(remoteFields), basePresent && remotePresent && spec.value(remoteFields) != base.Value),
		})
	}
	return rows
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
	basePresent := conflictHasBase(conflict)
	localPresent := conflictHasLocalVersion(conflict)
	remotePresent := conflictHasRemoteVersion(conflict)
	rowsByField := make(map[string]conflictComparisonRow)
	for _, row := range conflictComparisonRows(conflict) {
		rowsByField[row.Field] = row
	}
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
		fields = append(fields, conflictManualField{Name: field.name, Label: field.label, Options: options})
	}
	return fields
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
		Label:   "Lokale Aufgabe bleibt",
		Title:   conflictTitleValue(fields),
		UID:     conflictSplitUIDLabel(fields.UID, "UID bleibt erhalten"),
		Project: conflictProjectLabel(conflict.ProjectName),
		Status:  conflictStatusValue(fields),
		Due:     conflictDueValue(fields),
		Parent:  conflictSplitLocalParentLabel(fields),
	}
}

func conflictSplitRemotePreview(conflict db.ConflictDetail) conflictSplitPreview {
	fields, _ := conflictParsedFields(conflict.RemoteVTODO)
	return conflictSplitPreview{
		Label:   "Remote-Version wird neue Aufgabe",
		Title:   conflictTitleValue(fields),
		UID:     "Neue UID beim Speichern",
		Project: conflictProjectLabel(conflict.ProjectName),
		Status:  conflictStatusValue(fields),
		Due:     conflictDueValue(fields),
		Parent:  conflictSplitRemoteParentLabel(fields),
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
