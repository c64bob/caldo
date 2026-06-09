package view

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"caldo/internal/model"
)

func TestTaskRowRendersTodoistLikeMetadataAndCompletionControl(t *testing.T) {
	t.Parallel()

	ctx := WithCSRFToken(context.Background(), "token-123")
	component := TaskRow(TaskRowView{
		ID:            "task-1",
		Title:         "Sehr lange Aufgabe mit Metadaten",
		Description:   "Beschreibung vorhanden",
		ProjectName:   "Work",
		LabelNames:    "Büro,urgent,STARRED",
		DueISODate:    "2026-06-09",
		Status:        "needs-action",
		SyncStatus:    "pending",
		Priority:      1,
		HasPriority:   true,
		ServerVersion: 3,
	})

	var rendered bytes.Buffer
	if err := component.Render(ctx, &rendered); err != nil {
		t.Fatalf("render task row: %v", err)
	}

	output := rendered.String()
	for _, want := range []string{
		`data-task-id="task-1"`,
		`caldo-task-checkbox`,
		`hx-post="/tasks/task-1/complete"`,
		`name="expected_version" value="3"`,
		`X-CSRF-Token`,
		`Sehr lange Aufgabe mit Metadaten`,
		`Beschreibung vorhanden`,
		`Fällig 2026-06-09`,
		`Work`,
		`Büro`,
		`urgent`,
		`P1`,
		`Speichert`,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected task row to include %q in %s", want, output)
		}
	}
	if strings.Contains(output, `>STARRED<`) {
		t.Fatalf("reserved favorite category must not render as a normal label: %s", output)
	}
}

func TestTaskRowRendersCompletedReopenControlAndConflictState(t *testing.T) {
	t.Parallel()

	component := TaskRow(TaskRowView{
		ID:            "task-2",
		Title:         "Done",
		Status:        "completed",
		SyncStatus:    "conflict",
		ServerVersion: 4,
		IsSubtask:     true,
	})

	var rendered bytes.Buffer
	if err := component.Render(context.Background(), &rendered); err != nil {
		t.Fatalf("render task row: %v", err)
	}

	output := rendered.String()
	for _, want := range []string{
		`caldo-task-row-completed`,
		`caldo-task-row-subtask`,
		`hx-post="/tasks/task-2/reopen"`,
		`aria-pressed="true"`,
		`Konflikt`,
		`Erledigt`,
		`disabled`,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected completed task row to include %q in %s", want, output)
		}
	}
}

func TestTaskRowRendersInlineEditFormForSyncedTask(t *testing.T) {
	t.Parallel()

	ctx := WithCSRFToken(context.Background(), "token-123")
	component := TaskRow(TaskRowView{
		ID:            "task-1",
		ProjectID:     "project-1",
		Title:         "Editierbare Aufgabe",
		Description:   "Alter Text",
		ProjectName:   "Inbox",
		LabelNames:    "Büro,urgent,STARRED",
		DueISODate:    "2026-06-09",
		Status:        "needs-action",
		SyncStatus:    "synced",
		Priority:      4,
		HasPriority:   true,
		ServerVersion: 3,
		ProjectOptions: []TaskProjectOption{
			{ID: "project-1", Name: "Inbox"},
			{ID: "project-2", Name: "Work"},
		},
	})

	var rendered bytes.Buffer
	if err := component.Render(ctx, &rendered); err != nil {
		t.Fatalf("render task row: %v", err)
	}

	output := rendered.String()
	for _, want := range []string{
		`data-inline-task-display`,
		`data-inline-task-edit-open`,
		`data-inline-task-edit-form`,
		`data-task-actions`,
		`data-task-action-form`,
		`>Aufgabe erledigen<`,
		`data-task-action-error`,
		`data-task-delete-open`,
		`aria-controls="task-delete-task-1"`,
		`data-task-delete-dialog`,
		`data-task-delete-form`,
		`hx-delete="/tasks/task-1"`,
		`data-task-delete-error`,
		`Endgültig löschen`,
		`data-task-delete-cancel`,
		`hidden`,
		`hx-patch="/tasks/task-1"`,
		`name="expected_version" value="3"`,
		`name="status" value="needs-action"`,
		`name="title"`,
		`value="Editierbare Aufgabe"`,
		`Alter Text`,
		`name="project_id"`,
		`value="project-1" selected`,
		`value="project-2"`,
		`name="due_date" value="2026-06-09"`,
		`name="priority"`,
		`value="4" selected`,
		`name="labels" value="Büro, urgent"`,
		`data-inline-task-edit-cancel`,
		`data-inline-task-edit-error`,
		`X-CSRF-Token`,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected inline edit form to include %q in %s", want, output)
		}
	}
	if strings.Contains(output, `name="labels" value="Büro, urgent, STARRED"`) {
		t.Fatalf("reserved favorite category must not render in the label editor: %s", output)
	}
}

func TestTaskRowRendersVisibleReopenActionForCompletedSyncedTask(t *testing.T) {
	t.Parallel()

	ctx := WithCSRFToken(context.Background(), "token-123")
	component := TaskRow(TaskRowView{
		ID:            "task-done",
		Title:         "Fertig",
		Status:        "completed",
		SyncStatus:    "synced",
		ServerVersion: 7,
	})

	var rendered bytes.Buffer
	if err := component.Render(ctx, &rendered); err != nil {
		t.Fatalf("render task row: %v", err)
	}

	output := rendered.String()
	for _, want := range []string{
		`hx-post="/tasks/task-done/reopen"`,
		`>Aufgabe wieder öffnen<`,
		`Speichern ...`,
		`data-task-action-error`,
		`data-task-delete-open`,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected completed synced task row to include %q in %s", want, output)
		}
	}
}

func TestTaskRowRendersDetailPanelForSyncedTask(t *testing.T) {
	t.Parallel()

	ctx := WithCSRFToken(context.Background(), "token-123")
	component := TaskRow(TaskRowView{
		ID:            "task-1",
		ProjectID:     "project-1",
		Title:         "Detail Aufgabe",
		Description:   "Langer Detailtext",
		ProjectName:   "Inbox",
		LabelNames:    "Büro,urgent,STARRED",
		DueISODate:    "2026-06-09",
		Status:        "needs-action",
		SyncStatus:    "synced",
		Priority:      4,
		HasPriority:   true,
		ServerVersion: 3,
		RRule:         "FREQ=WEEKLY;INTERVAL=2;COUNT=5",
		Attachments: []model.Attachment{
			{Value: "https://example.com/file.pdf", IsExternalURL: true},
			{Value: "AAAA", IsExternalURL: false},
		},
		ProjectOptions: []TaskProjectOption{
			{ID: "project-1", Name: "Inbox"},
			{ID: "project-2", Name: "Work"},
		},
	})

	var rendered bytes.Buffer
	if err := component.Render(ctx, &rendered); err != nil {
		t.Fatalf("render task row: %v", err)
	}

	output := rendered.String()
	for _, want := range []string{
		`data-task-detail-open`,
		`aria-haspopup="dialog"`,
		`data-task-detail-dialog`,
		`data-task-detail-form`,
		`hx-patch="/tasks/task-1"`,
		`name="expected_version" value="3"`,
		`name="title"`,
		`value="Detail Aufgabe"`,
		`Langer Detailtext`,
		`name="project_id"`,
		`value="project-1" selected`,
		`name="due_date" value="2026-06-09"`,
		`name="priority"`,
		`value="4" selected`,
		`name="labels" value="Büro, urgent"`,
		`name="repeat_update" value="1"`,
		`name="repeat_freq"`,
		`value="WEEKLY" selected`,
		`name="repeat_interval" value="2"`,
		`name="repeat_end"`,
		`value="count" selected`,
		`name="repeat_count" value="5"`,
		`https://example.com/file.pdf`,
		`rel="noopener noreferrer"`,
		`Anhang vorhanden (inline/binary)`,
		`data-task-detail-error`,
		`X-CSRF-Token`,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected detail panel to include %q in %s", want, output)
		}
	}
	if strings.Contains(output, `name="labels" value="Büro, urgent, STARRED"`) {
		t.Fatalf("reserved favorite category must not render in the label editor: %s", output)
	}
}

func TestTaskRowRendersConflictDetailPanelReadOnly(t *testing.T) {
	t.Parallel()

	component := TaskRow(TaskRowView{
		ID:            "task-conflict",
		Title:         "Konflikt Aufgabe",
		Status:        "needs-action",
		SyncStatus:    "conflict",
		ServerVersion: 5,
		RRule:         "FREQ=MONTHLY;BYDAY=MO,TU,WE,TH,FR;BYSETPOS=1",
	})

	var rendered bytes.Buffer
	if err := component.Render(context.Background(), &rendered); err != nil {
		t.Fatalf("render task row: %v", err)
	}

	output := rendered.String()
	for _, want := range []string{
		`data-task-detail-open`,
		`data-task-detail-dialog`,
		`Konflikt`,
		`Konfliktlösung`,
		`RRULE: FREQ=MONTHLY;BYDAY=MO,TU,WE,TH,FR;BYSETPOS=1`,
		`Komplexe Wiederholung wird unverändert erhalten.`,
		`disabled`,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected conflict detail panel to include %q in %s", want, output)
		}
	}
	if strings.Contains(output, `data-inline-task-edit-form`) {
		t.Fatalf("conflict task must not render inline edit form: %s", output)
	}
	if strings.Contains(output, `data-task-delete-form`) {
		t.Fatalf("conflict task must not render delete form: %s", output)
	}
}

func TestInlineTaskCreateRendersContextAndControls(t *testing.T) {
	t.Parallel()

	ctx := WithCSRFToken(context.Background(), "token-123")
	component := InlineTaskCreate(InlineTaskCreateView{
		Enabled:     true,
		ProjectID:   "project-1",
		ProjectName: "Work",
		DueDate:     "2026-06-09",
		Placeholder: "Aufgabe in Work hinzufügen",
	})

	var rendered bytes.Buffer
	if err := component.Render(ctx, &rendered); err != nil {
		t.Fatalf("render inline create: %v", err)
	}

	output := rendered.String()
	for _, want := range []string{
		`data-inline-task-create`,
		`data-inline-task-create-trigger`,
		`aria-expanded="false"`,
		`data-inline-task-create-form`,
		`hidden`,
		`hx-post="/tasks/"`,
		`name="project_id" value="project-1"`,
		`name="due_date" value="2026-06-09"`,
		`name="title"`,
		`required`,
		`data-inline-task-create-cancel`,
		`data-inline-task-create-error`,
		`X-CSRF-Token`,
		`Work`,
		`Fällig 2026-06-09`,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected inline create to include %q in %s", want, output)
		}
	}
}
