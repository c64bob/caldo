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
		TodayISODate:  "2026-06-09",
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
		`data-server-version="3"`,
		`caldo-task-checkbox`,
		`hx-post="/tasks/task-1/complete"`,
		`name="expected_version" value="3"`,
		`X-CSRF-Token`,
		`Sehr lange Aufgabe mit Metadaten`,
		`Beschreibung vorhanden`,
		`Heute`,
		`Work`,
		`Büro`,
		`urgent`,
		`P1 Hoch`,
		`data-task-favorite-form`,
		`hx-post="/tasks/task-1/favorite"`,
		`name="favorite" value="false"`,
		`aria-label="Favorit entfernen"`,
		`aria-pressed="true"`,
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
		LabelNames:    "urgent,Büro,STARRED",
		DueISODate:    "2026-06-09",
		TodayISODate:  "2026-06-09",
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
		`data-task-favorite-form`,
		`aria-label="Favorit entfernen"`,
		`aria-pressed="true"`,
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
		`data-subtask-create`,
		`data-subtask-create-form`,
		`hx-post="/tasks/task-1/subtasks"`,
		`Unteraufgabe hinzufügen`,
		`Unteraufgabentitel`,
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

func TestTaskRowRendersParentCompletionDecisionDialog(t *testing.T) {
	t.Parallel()

	ctx := WithCSRFToken(context.Background(), "token-123")
	component := TaskRow(TaskRowView{
		ID:               "task-parent",
		Title:            "Elternaufgabe",
		Status:           "needs-action",
		SyncStatus:       "synced",
		ServerVersion:    5,
		SubtaskCount:     3,
		OpenSubtaskCount: 2,
	})

	var rendered bytes.Buffer
	if err := component.Render(ctx, &rendered); err != nil {
		t.Fatalf("render task row: %v", err)
	}

	output := rendered.String()
	for _, want := range []string{
		`data-task-complete-open`,
		`aria-haspopup="dialog"`,
		`aria-controls="task-complete-task-parent"`,
		`data-task-complete-dialog`,
		`id="task-complete-task-parent"`,
		`data-task-complete-form`,
		`hx-post="/tasks/task-parent/complete"`,
		`name="expected_version" value="5"`,
		`name="subtasks_action" value="parent_only"`,
		`name="subtasks_action" value="complete_open"`,
		`Nur Elternaufgabe erledigen`,
		`Aufgabe und 2 offene Unteraufgaben erledigen`,
		`data-task-complete-error`,
		`data-task-complete-cancel`,
		`X-CSRF-Token`,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected parent completion dialog to include %q in %s", want, output)
		}
	}
}

func TestTaskRowRendersParentDeleteDialogWithSubtaskCount(t *testing.T) {
	t.Parallel()

	ctx := WithCSRFToken(context.Background(), "token-123")
	component := TaskRow(TaskRowView{
		ID:            "task-parent",
		Title:         "Elternaufgabe",
		Status:        "needs-action",
		SyncStatus:    "synced",
		ServerVersion: 5,
		SubtaskCount:  2,
	})

	var rendered bytes.Buffer
	if err := component.Render(ctx, &rendered); err != nil {
		t.Fatalf("render task row: %v", err)
	}

	output := rendered.String()
	for _, want := range []string{
		`data-task-delete-dialog`,
		`Diese Aufgabe hat 2 Unteraufgaben.`,
		`name="subtasks_action" value="delete_all"`,
		`Aufgabe und Unteraufgaben löschen`,
		`hx-delete="/tasks/task-parent"`,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected parent delete dialog to include %q in %s", want, output)
		}
	}
}

func TestTaskRowRendersSubtaskRelationshipWithoutCreateAction(t *testing.T) {
	t.Parallel()

	ctx := WithCSRFToken(context.Background(), "token-123")
	component := TaskRow(TaskRowView{
		ID:            "child-1",
		Title:         "Kindaufgabe",
		ParentID:      "parent-1",
		ParentTitle:   "Hauptaufgabe",
		Status:        "needs-action",
		SyncStatus:    "synced",
		ServerVersion: 7,
		IsSubtask:     true,
	})

	var rendered bytes.Buffer
	if err := component.Render(ctx, &rendered); err != nil {
		t.Fatalf("render task row: %v", err)
	}

	output := rendered.String()
	for _, want := range []string{
		`data-task-id="child-1"`,
		`data-parent-task-id="parent-1"`,
		`caldo-task-row-subtask`,
		`caldo-task-chip-subtask`,
		`Unteraufgabe von Hauptaufgabe`,
		`hx-post="/tasks/child-1/complete"`,
		`name="expected_version" value="7"`,
		`>Aufgabe erledigen<`,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected subtask row to include %q in %s", want, output)
		}
	}
	for _, unwanted := range []string{
		`data-subtask-create`,
		`/tasks/child-1/subtasks`,
		`Unteraufgabe hinzufügen`,
	} {
		if strings.Contains(output, unwanted) {
			t.Fatalf("subtask row must not include %q in %s", unwanted, output)
		}
	}
}

func TestTaskRowRendersFaultySubtaskRelationshipWithoutParentTitle(t *testing.T) {
	t.Parallel()

	component := TaskRow(TaskRowView{
		ID:            "child-missing-parent",
		Title:         "Verwaiste Unteraufgabe",
		ParentID:      "missing-parent",
		Status:        "completed",
		SyncStatus:    "synced",
		ServerVersion: 4,
		IsSubtask:     true,
	})

	var rendered bytes.Buffer
	if err := component.Render(context.Background(), &rendered); err != nil {
		t.Fatalf("render task row: %v", err)
	}

	output := rendered.String()
	for _, want := range []string{
		`caldo-task-row-subtask`,
		`caldo-task-row-completed`,
		`caldo-task-chip-subtask`,
		`>Unteraufgabe<`,
		`hx-post="/tasks/child-missing-parent/reopen"`,
		`aria-pressed="true"`,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected faulty subtask row to include %q in %s", want, output)
		}
	}
	if strings.Contains(output, `Unteraufgabe von`) {
		t.Fatalf("missing parent title must render a generic subtask chip: %s", output)
	}
}

func TestTaskRowRendersDueStateChips(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		due   string
		want  string
		class string
	}{
		{name: "none", due: "", want: "Ohne Datum", class: "caldo-task-chip-due-none"},
		{name: "overdue", due: "2026-06-08", want: "Überfällig 2026-06-08", class: "caldo-task-chip-due-overdue"},
		{name: "today", due: "2026-06-09", want: "Heute", class: "caldo-task-chip-due-today"},
		{name: "future", due: "2026-06-10", want: "Fällig 2026-06-10", class: "caldo-task-chip-due-future"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			component := TaskRow(TaskRowView{
				ID:            "task-" + tt.name,
				Title:         "Due " + tt.name,
				DueISODate:    tt.due,
				TodayISODate:  "2026-06-09",
				Status:        "needs-action",
				SyncStatus:    "synced",
				ServerVersion: 1,
			})

			var rendered bytes.Buffer
			if err := component.Render(context.Background(), &rendered); err != nil {
				t.Fatalf("render task row: %v", err)
			}

			output := rendered.String()
			if !strings.Contains(output, tt.want) {
				t.Fatalf("expected due chip %q in %s", tt.want, output)
			}
			if !strings.Contains(output, tt.class) {
				t.Fatalf("expected due chip class %q in %s", tt.class, output)
			}
		})
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
		TodayISODate:  "2026-06-09",
		Status:        "needs-action",
		SyncStatus:    "synced",
		Priority:      4,
		HasPriority:   true,
		ServerVersion: 3,
		SubtaskCount:  2,
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
		`2 Unteraufgaben`,
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
	if !strings.Contains(output, `data-task-favorite-form`) || !strings.Contains(output, `disabled`) {
		t.Fatalf("conflict task must render disabled favorite state: %s", output)
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
