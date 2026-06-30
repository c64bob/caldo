package view

import (
	"bytes"
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	"caldo/internal/db"
	"caldo/internal/model"
)

func TestConflictListPageRendersWorklistRowsWithoutRawVTODO(t *testing.T) {
	t.Parallel()

	component := ConflictListPage([]db.ConflictListRow{
		{
			ID:           "conflict-delete",
			ProjectName:  "Inbox",
			ConflictType: "edit_delete",
			CreatedAt:    time.Date(2026, 6, 11, 8, 30, 0, 0, time.UTC),
			TaskTitle:    "Local changed task",
		},
		{
			ID:           "conflict-field",
			ProjectName:  "Work",
			ConflictType: "field_conflict",
			CreatedAt:    time.Date(2026, 6, 11, 9, 30, 0, 0, time.UTC),
			TaskTitle:    "Field changed task",
		},
	})

	var rendered bytes.Buffer
	if err := component.Render(context.Background(), &rendered); err != nil {
		t.Fatalf("render conflict list: %v", err)
	}

	output := rendered.String()
	for _, want := range []string{
		`data-conflict-list-summary`,
		`2 offene Konflikte`,
		`data-conflict-list-row`,
		`data-conflict-type="edit_delete"`,
		`Lokal geändert, remote gelöscht`,
		`Lokale Änderung prüfen oder Remote-Löschung übernehmen`,
		`data-conflict-type="field_conflict"`,
		`Feldkonflikt`,
		`Felder vergleichen und Zielversion wählen`,
		`Erkannt`,
		`datetime="2026-06-11T08:30:00Z"`,
		`data-local-date-time`,
		`Projekt: Inbox`,
		`href="/conflicts/conflict-delete"`,
		`Konflikt lösen`,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected conflict list to include %q in %s", want, output)
		}
	}
	if strings.Contains(output, "BEGIN:VTODO") || strings.Contains(output, "BEGIN:VCALENDAR") {
		t.Fatalf("conflict list must not render raw vtodo data: %s", output)
	}
}

func TestConflictListPageRendersEmptyState(t *testing.T) {
	t.Parallel()

	var rendered bytes.Buffer
	if err := ConflictListPage(nil).Render(context.Background(), &rendered); err != nil {
		t.Fatalf("render conflict list: %v", err)
	}

	output := rendered.String()
	if !strings.Contains(output, "Keine ungelösten Konflikte") || strings.Contains(output, `data-conflict-list-row`) {
		t.Fatalf("expected clear empty conflict list state, got %s", output)
	}
}

func TestConflictDetailPageRendersReadableComparison(t *testing.T) {
	t.Parallel()

	component := ConflictDetailPage(db.ConflictDetail{
		ID:           "conflict-1",
		TaskID:       sql.NullString{Valid: true, String: "task-1"},
		ProjectID:    sql.NullString{Valid: true, String: "project-1"},
		ProjectName:  "Work",
		ConflictType: "field_conflict",
		CreatedAt:    time.Date(2026, 6, 11, 8, 30, 0, 0, time.UTC),
		BaseVTODO:    sql.NullString{Valid: true, String: "BEGIN:VTODO\r\nSUMMARY:Base title\r\nDESCRIPTION:Base desc\r\nSTATUS:NEEDS-ACTION\r\nDUE;VALUE=DATE:20260610\r\nPRIORITY:5\r\nCATEGORIES:shared,STARRED\r\nEND:VTODO\r\n"},
		LocalVTODO:   sql.NullString{Valid: true, String: "BEGIN:VTODO\r\nUID:uid-local\r\nSUMMARY:Local title\r\nDESCRIPTION:Local desc\r\nSTATUS:NEEDS-ACTION\r\nDUE;VALUE=DATE:20260611\r\nPRIORITY:1\r\nCATEGORIES:shared,local,STARRED\r\nRRULE:FREQ=WEEKLY\r\nEND:VTODO\r\n"},
		RemoteVTODO:  sql.NullString{Valid: true, String: "BEGIN:VTODO\r\nUID:uid-remote\r\nSUMMARY:Remote title\r\nDESCRIPTION:Remote desc\r\nSTATUS:COMPLETED\r\nDUE;VALUE=DATE:20260612\r\nPRIORITY:9\r\nCATEGORIES:shared,remote\r\nRELATED-TO;RELTYPE=PARENT:uid-parent\r\nATTACH:https://example.com/spec.pdf\r\nEND:VTODO\r\n"},
	}, []TaskProjectOption{{ID: "project-1", Name: "Work"}, {ID: "project-2", Name: "Personal"}})

	var rendered bytes.Buffer
	if err := component.Render(WithCSRFToken(context.Background(), "token-123"), &rendered); err != nil {
		t.Fatalf("render conflict detail: %v", err)
	}

	output := rendered.String()
	for _, want := range []string{
		`data-conflict-comparison`,
		`Feldkonflikt`,
		`aria-label="Feldvergleich lokaler und entfernter Konfliktversionen"`,
		`data-conflict-field="project"`,
		`data-conflict-value="local-project"`,
		`data-conflict-row-state="unchanged"`,
		`caldo-conflict-row-unchanged`,
		`data-conflict-field="title"`,
		`data-conflict-row-state="changed"`,
		`caldo-conflict-row-changed`,
		`Base title`,
		`Local title`,
		`Remote title`,
		`2026-06-11`,
		`2026-06-12`,
		`P1 Hoch (1)`,
		`P3 Niedrig (9)`,
		`shared, local`,
		`shared, remote`,
		`Ja`,
		`Nein`,
		`Wöchentlich`,
		`https://example.com/spec.pdf`,
		`data-conflict-value="local-title"`,
		`caldo-conflict-value-changed`,
		`data-conflict-resolution`,
		`data-conflict-resolve-form`,
		`name="resolution" value="local"`,
		`name="resolution" value="remote"`,
		`name="resolution" value="split"`,
		`data-conflict-split-preview`,
		`data-conflict-split-form`,
		`Nach dem Split existieren zwei Aufgaben`,
		`data-conflict-split-target="local"`,
		`data-conflict-split-target="remote"`,
		`Aufgabe 1: Lokale Variante bleibt`,
		`Aufgabe 2: Remote-Variante wird neu angelegt`,
		`Die lokale Aufgabe bleibt als bestehende Aufgabe`,
		`Die entfernte Variante wird als zweite Aufgabe`,
		`UID uid-local`,
		`Neue UID beim Speichern`,
		`Labels`,
		`Parent-Beziehung wird entfernt`,
		`name="confirm_split" value="1" required`,
		`data-conflict-split-confirm`,
		`data-conflict-split-submit`,
		`data-conflict-manual-form`,
		`data-conflict-field-source="title"`,
		`data-conflict-source-option="local"`,
		`data-conflict-source-option="remote"`,
		`data-conflict-source-option="manual"`,
		`data-conflict-option-display="Remote title"`,
		`type="radio" name="title_source" value="local"`,
		`type="radio" name="title_source" value="remote" checked`,
		`type="radio" name="title_source" value="manual"`,
		`caldo-conflict-manual-value`,
		`name="title_source"`,
		`name="title_manual"`,
		`name="description_source"`,
		`name="description_manual"`,
		`name="due_source"`,
		`name="due_manual"`,
		`name="priority_source"`,
		`name="priority_manual"`,
		`name="labels_source"`,
		`name="labels_manual"`,
		`name="status_source"`,
		`name="status_manual"`,
		`name="parent_source"`,
		`name="parent_manual"`,
		`name="project_id"`,
		`data-conflict-project-select`,
		`data-conflict-manual-preview`,
		`data-conflict-preview-value="title"`,
		`data-conflict-preview-value="project"`,
		`Personal`,
		`X-CSRF-Token`,
		`Technische VTODO-Daten`,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected readable conflict detail to include %q in %s", want, output)
		}
	}
}

func TestConflictDetailPageRendersDeletedSideAsMissing(t *testing.T) {
	t.Parallel()

	component := ConflictDetailPage(db.ConflictDetail{
		ID:           "conflict-1",
		TaskID:       sql.NullString{Valid: true, String: "task-1"},
		ProjectName:  "Work",
		ConflictType: "edit_delete",
		CreatedAt:    time.Date(2026, 6, 11, 8, 30, 0, 0, time.UTC),
		BaseVTODO:    sql.NullString{Valid: true, String: "BEGIN:VTODO\r\nSUMMARY:Base title\r\nEND:VTODO\r\n"},
		LocalVTODO:   sql.NullString{Valid: true, String: "BEGIN:VTODO\r\nSUMMARY:Local title\r\nEND:VTODO\r\n"},
		RemoteVTODO:  sql.NullString{},
	}, nil)

	var rendered bytes.Buffer
	if err := component.Render(context.Background(), &rendered); err != nil {
		t.Fatalf("render conflict detail: %v", err)
	}

	output := rendered.String()
	if !strings.Contains(output, "Lokal geändert, remote gelöscht") || !strings.Contains(output, "Nicht vorhanden (gelöscht)") {
		t.Fatalf("expected deletion conflict labels in %s", output)
	}
}

func TestConflictDueDateTimeUsesBrowserLocalMetadata(t *testing.T) {
	t.Parallel()

	dueAt := time.Date(2026, time.June, 11, 8, 30, 0, 0, time.UTC)
	fields := model.VTODOFields{DueAt: &dueAt}

	if got := conflictDueValue(fields); got != "11.06.2026 08:30" {
		t.Fatalf("unexpected due fallback label: %q", got)
	}
	dateTime := conflictDueDateTime(fields)
	if dateTime.ISO != "2026-06-11T08:30:00Z" || dateTime.Text != "11.06.2026 08:30" {
		t.Fatalf("unexpected due datetime metadata: %#v", dateTime)
	}

	dueDate := "20260611"
	dateOnly := model.VTODOFields{DueDate: &dueDate, DueAt: &dueAt}
	if got := conflictDueValue(dateOnly); got != dueDate {
		t.Fatalf("expected date-only due value to win, got %q", got)
	}
	if got := conflictDueDateTime(dateOnly); got.ISO != "" {
		t.Fatalf("expected date-only due value not to carry datetime metadata, got %#v", got)
	}
}
