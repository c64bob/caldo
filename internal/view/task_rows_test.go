package view

import (
	"bytes"
	"context"
	"strings"
	"testing"
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
