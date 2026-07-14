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

func TestTaskRowRendersDescriptionURLsAsSafeLinks(t *testing.T) {
	t.Parallel()

	ctx := WithCSRFToken(context.Background(), "token-123")
	description := "Bitte https://example.com/browser prüfen."
	component := TaskRow(TaskRowView{
		ID:            "task-link",
		ProjectID:     "project-1",
		Title:         "Link Aufgabe",
		Description:   description,
		Status:        "needs-action",
		SyncStatus:    "synced",
		ServerVersion: 2,
	})

	var rendered bytes.Buffer
	if err := component.Render(ctx, &rendered); err != nil {
		t.Fatalf("render task row: %v", err)
	}

	output := rendered.String()
	for _, want := range []string{
		`Bitte `,
		`<a href="https://example.com/browser" target="_blank" rel="noopener noreferrer" class="caldo-task-description-link">https://example.com/browser</a>`,
		` prüfen.`,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected linked description to include %q in %s", want, output)
		}
	}
	if !strings.Contains(output, `>Bitte https://example.com/browser prüfen.</textarea>`) {
		t.Fatalf("expected detail edit controls to preserve raw description %q in %s", description, output)
	}
}

func TestTaskRowRendersDescriptionMarkdownReadOnlyWithoutChangingRawEditValue(t *testing.T) {
	t.Parallel()

	description := "Bitte **prüfen**\\nZeile 2 mit escaped\\, comma"
	component := TaskRow(TaskRowView{
		ID:            "task-markdown",
		ProjectID:     "project-1",
		Title:         "Markdown Aufgabe",
		Description:   description,
		Status:        "needs-action",
		SyncStatus:    "synced",
		ServerVersion: 2,
	})

	var rendered bytes.Buffer
	if err := component.Render(context.Background(), &rendered); err != nil {
		t.Fatalf("render task row: %v", err)
	}

	output := rendered.String()
	for _, want := range []string{
		`Bitte <strong>prüfen</strong><br>Zeile 2 mit escaped, comma`,
		`>Bitte **prüfen**\nZeile 2 mit escaped\, comma</textarea>`,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected markdown description to include %q in %s", want, output)
		}
	}
}

func TestTaskRowRendersDecodedReadOnlyTextAndRawEditValues(t *testing.T) {
	t.Parallel()

	component := TaskRow(TaskRowView{
		ID:            "task-escaped",
		ProjectID:     "project-1",
		Title:         `Plan\, phase 2\; team\\blue`,
		Description:   `First\nSecond\, item`,
		ParentTitle:   `Parent\, task`,
		IsSubtask:     true,
		Status:        "needs-action",
		SyncStatus:    "pending",
		ServerVersion: 2,
	})

	var rendered bytes.Buffer
	if err := component.Render(context.Background(), &rendered); err != nil {
		t.Fatalf("render task row: %v", err)
	}

	output := rendered.String()
	for _, want := range []string{
		`<p class="caldo-task-title">Plan, phase 2; team\blue</p>`,
		`<h2 id="task-detail-title-task-escaped" class="caldo-task-detail-title">Plan, phase 2; team\blue</h2>`,
		`Unteraufgabe von Parent, task`,
		`First<br>Second, item`,
		`value="Plan\, phase 2\; team\\blue"`,
		`>First\nSecond\, item</textarea>`,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected decoded display and raw edit value %q in %s", want, output)
		}
	}
}

func TestTaskRowHonorsTaskNoteDisplayPreference(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		mode      string
		wantClass string
		hidden    bool
	}{
		{name: "none", mode: TaskNoteDisplayNone, hidden: true},
		{name: "full", mode: TaskNoteDisplayFull, wantClass: "caldo-task-description-full"},
		{name: "one line", mode: TaskNoteDisplayOneLine, wantClass: "caldo-task-description-lines-1"},
		{name: "two lines", mode: TaskNoteDisplayTwoLines, wantClass: "caldo-task-description-lines-2"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := WithTaskNoteDisplay(context.Background(), tt.mode)
			component := TaskRow(TaskRowView{
				ID:            "task-note-" + strings.ReplaceAll(tt.name, " ", "-"),
				ProjectID:     "project-1",
				Title:         "Notiz Aufgabe",
				Description:   "Zeile 1\\nZeile 2\\nZeile 3",
				Status:        "needs-action",
				SyncStatus:    "synced",
				ServerVersion: 2,
			})

			var rendered bytes.Buffer
			if err := component.Render(ctx, &rendered); err != nil {
				t.Fatalf("render task row: %v", err)
			}

			output := rendered.String()
			hasDescription := strings.Contains(output, `caldo-task-description`)
			if tt.hidden {
				if hasDescription {
					t.Fatalf("expected task note to be hidden in %s", output)
				}
				if !strings.Contains(output, `>Zeile 1\nZeile 2\nZeile 3</textarea>`) {
					t.Fatalf("hidden list notes must not remove raw detail description in %s", output)
				}
				return
			}
			if !hasDescription || !strings.Contains(output, tt.wantClass) {
				t.Fatalf("expected task note class %q in %s", tt.wantClass, output)
			}
		})
	}
}

func TestTaskRowLeavesPlainDescriptionTextUnlinked(t *testing.T) {
	t.Parallel()

	component := TaskRow(TaskRowView{
		ID:            "task-plain",
		Title:         "Plain Aufgabe",
		Description:   "Nur Text ohne URL",
		Status:        "needs-action",
		SyncStatus:    "synced",
		ServerVersion: 1,
	})

	var rendered bytes.Buffer
	if err := component.Render(context.Background(), &rendered); err != nil {
		t.Fatalf("render task row: %v", err)
	}

	output := rendered.String()
	if !strings.Contains(output, `Nur Text ohne URL`) {
		t.Fatalf("expected plain description text in %s", output)
	}
	if strings.Contains(output, `caldo-task-description-link`) || strings.Contains(output, `<a href=`) {
		t.Fatalf("plain description must not render as link: %s", output)
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
		ParentVisible: true,
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
		`type="checkbox"`,
		`checked`,
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
		`data-task-project-id="project-1"`,
		`data-task-move-path="/tasks/task-1/move"`,
		`data-task-drag-move`,
		`draggable="true"`,
		`data-inline-task-edit-persistent`,
		`data-inline-task-edit-title`,
		`aria-label="Titel bearbeiten: Editierbare Aufgabe"`,
		`data-inline-task-edit-open`,
		`data-inline-task-edit-focus="title"`,
		`data-inline-task-edit-focus="project"`,
		`data-inline-task-edit-kind="project"`,
		`aria-controls="task-project-edit-task-1"`,
		`data-inline-task-edit-focus="priority"`,
		`data-inline-task-edit-kind="priority"`,
		`data-inline-task-priority-select`,
		`caldo-task-priority-select`,
		`data-inline-task-edit-focus="labels"`,
		`data-inline-task-edit-kind="labels"`,
		`aria-controls="task-labels-edit-task-1"`,
		`data-inline-task-edit-control`,
		`data-inline-task-edit-autosave`,
		`data-bulk-select-control`,
		`aria-label="Für Mehrfachbearbeitung auswählen: Editierbare Aufgabe"`,
		`caldo-date-dropdown`,
		`data-date-hidden-input`,
		`caldo-date-quick-action`,
		`data-task-favorite-form`,
		`aria-label="Favorit entfernen"`,
		`aria-pressed="true"`,
		`data-task-action-form`,
		`hx-disabled-elt="find input[data-task-completion-checkbox]"`,
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
		`hx-get="/tasks/task-1/subtasks"`,
		`hx-trigger="caldo:task-detail-open"`,
		`data-task-detail-subtask-list`,
		`Unteraufgabe hinzufügen`,
		`Unteraufgaben`,
		`Unteraufgabentitel`,
		`hidden`,
		`hx-patch="/tasks/task-1"`,
		`name="expected_version" value="3"`,
		`name="status" value="needs-action"`,
		`name="title"`,
		`value="Editierbare Aufgabe"`,
		`Alter Text`,
		`name="project_id"`,
		`<option value="4" selected>P1 Hoch</option>`,
		`name="labels" value="Büro, urgent"`,
		`name="due_date" value="2026-06-09"`,
		`data-inline-task-edit-error`,
		`X-CSRF-Token`,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected inline edit form to include %q in %s", want, output)
		}
	}
	for _, notWant := range []string{
		`>Bearbeiten<`,
		`class="caldo-task-edit-open"`,
	} {
		if strings.Contains(output, notWant) {
			t.Fatalf("inline edit trigger must not render old edit button %q in %s", notWant, output)
		}
	}
	if strings.Contains(output, `name="labels" value="Büro, urgent, STARRED"`) {
		t.Fatalf("reserved favorite category must not render in the label editor: %s", output)
	}
	if strings.Contains(output, `>Aufgabe erledigen<`) {
		t.Fatalf("completion must be available only through the checkbox control: %s", output)
	}
	if strings.Contains(output, `data-inline-task-edit-open data-inline-task-edit-focus="title"`) {
		t.Fatalf("expected task title to use the persistent native input in %s", output)
	}
}

func TestTaskPriorityInlineEditorRendersPersistentNativeSelect(t *testing.T) {
	t.Parallel()

	component := TaskPriorityInlineEditor(TaskRowView{
		ID:            "task-priority",
		Priority:      4,
		HasPriority:   true,
		ServerVersion: 7,
	})

	var rendered bytes.Buffer
	if err := component.Render(WithCSRFToken(context.Background(), "token-123"), &rendered); err != nil {
		t.Fatalf("render priority editor: %v", err)
	}

	output := rendered.String()
	for _, want := range []string{
		`data-inline-task-edit-kind="priority"`,
		`data-inline-task-edit-persistent`,
		`data-inline-task-priority-select`,
		`name="priority"`,
		`autocomplete="off"`,
		`caldo-task-priority-select`,
		`caldo-task-priority-select-caret`,
		`caldo-task-priority-p1`,
		`aria-label="Priorität bearbeiten: P1 Hoch"`,
		`<option value="4" selected>P1 Hoch</option>`,
		`<option value="5">P2 Mittel</option>`,
		`<option value="9">P3 Niedrig</option>`,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected one-click priority editor to include %q in %s", want, output)
		}
	}
	for _, unwanted := range []string{
		`data-inline-task-edit-open`,
		`aria-controls=`,
		`<option value="1">P1 Hoch</option>`,
		` hidden`,
	} {
		if strings.Contains(output, unwanted) {
			t.Fatalf("one-click priority editor must not include %q in %s", unwanted, output)
		}
	}
}

func TestTaskRowAssociatesInteractiveFormErrors(t *testing.T) {
	t.Parallel()

	ctx := WithCSRFToken(context.Background(), "token-123")
	component := TaskRow(TaskRowView{
		ID:               "task-a11y",
		ProjectID:        "project-1",
		Title:            "A11y Aufgabe",
		ProjectName:      "Inbox",
		Status:           "needs-action",
		SyncStatus:       "synced",
		ServerVersion:    8,
		SubtaskCount:     1,
		OpenSubtaskCount: 1,
		ProjectOptions: []TaskProjectOption{
			{ID: "project-1", Name: "Inbox"},
		},
	})

	var rendered bytes.Buffer
	if err := component.Render(ctx, &rendered); err != nil {
		t.Fatalf("render task row: %v", err)
	}

	output := rendered.String()
	for _, want := range []string{
		`id="task-action-error-task-a11y"`,
		`id="task-edit-error-task-a11y"`,
		`aria-describedby="task-edit-error-task-a11y"`,
		`id="subtask-create-error-task-a11y"`,
		`aria-describedby="subtask-create-error-task-a11y"`,
		`id="task-detail-error-task-a11y"`,
		`aria-describedby="task-detail-error-task-a11y"`,
		`id="task-complete-error-task-a11y"`,
		`aria-describedby="task-complete-error-task-a11y"`,
		`id="task-delete-error-task-a11y"`,
		`aria-describedby="task-delete-error-task-a11y"`,
		`role="alert"`,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected associated task error markup %q in %s", want, output)
		}
	}
}

func TestTaskRowDoesNotEnableDragMoveForUnsyncedTask(t *testing.T) {
	t.Parallel()

	component := TaskRow(TaskRowView{
		ID:            "task-pending",
		ProjectID:     "project-1",
		Title:         "Pending",
		Status:        "needs-action",
		SyncStatus:    "pending",
		ServerVersion: 3,
	})

	var rendered bytes.Buffer
	if err := component.Render(context.Background(), &rendered); err != nil {
		t.Fatalf("render task row: %v", err)
	}

	output := rendered.String()
	if strings.Contains(output, `data-task-drag-move`) {
		t.Fatalf("pending task must not be draggable between projects: %s", output)
	}
	if !strings.Contains(output, `draggable="false"`) {
		t.Fatalf("expected pending task to render draggable=false in %s", output)
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
		ParentVisible: true,
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
		`data-task-parent-visible`,
		`caldo-task-row-subtask`,
		`caldo-task-chip-subtask`,
		`Unteraufgabe von Hauptaufgabe`,
		`data-task-parent-open`,
		`data-parent-task-id="parent-1"`,
		`aria-label="Elternaufgabe öffnen: Hauptaufgabe"`,
		`hx-post="/tasks/child-1/complete"`,
		`name="expected_version" value="7"`,
		`type="checkbox"`,
		`aria-label="Aufgabe erledigen"`,
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
		`caldo-task-row-completed`,
		`caldo-task-chip-subtask`,
		`>Unteraufgabe</button>`,
		`data-task-parent-open`,
		`aria-label="Elternaufgabe öffnen"`,
		`hx-post="/tasks/child-missing-parent/reopen"`,
		`type="checkbox"`,
		`checked`,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected faulty subtask row to include %q in %s", want, output)
		}
	}
	if strings.Contains(output, `caldo-task-row-subtask`) || strings.Contains(output, `data-task-parent-visible`) {
		t.Fatalf("subtask without a visible parent must not be indented: %s", output)
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
		{name: "overdue", due: "2026-06-08", want: "08.06.2026", class: "caldo-task-chip-due-overdue"},
		{name: "today", due: "2026-06-09", want: "Heute", class: "caldo-task-chip-due-today"},
		{name: "future", due: "2026-06-10", want: "Fällig 10.06.2026", class: "caldo-task-chip-due-future"},
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

func TestTaskRowRendersCompletedCheckboxAndEditOnlyDeleteAction(t *testing.T) {
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
		`type="checkbox"`,
		`aria-label="Aufgabe wieder öffnen"`,
		`checked`,
		`Speichern ...`,
		`data-task-action-error`,
		`data-task-delete-open`,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected completed synced task row to include %q in %s", want, output)
		}
	}
	if strings.Contains(output, `>Aufgabe wieder öffnen<`) {
		t.Fatalf("reopen must be available only through the checkbox control: %s", output)
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
		`data-task-labels-input`,
		`name="repeat_update" value="1"`,
		`data-task-recurrence-update`,
		`name="repeat_freq"`,
		`value="WEEKLY" selected`,
		`name="repeat_interval" value="2"`,
		`data-task-recurrence-control`,
		`name="repeat_end"`,
		`value="count" selected`,
		`name="repeat_count" value="5"`,
		`2 Unteraufgaben`,
		`https://example.com/file.pdf`,
		`rel="noopener noreferrer"`,
		`Anhang vorhanden (inline/binary)`,
		`data-task-detail-error`,
		`data-task-detail-complete`,
		`>Erledigt</button>`,
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

func TestTaskDetailCompleteActionOnlyRendersForSyncedOpenTasks(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		status     string
		syncStatus string
		want       bool
	}{
		{name: "open synced", status: "needs-action", syncStatus: "synced", want: true},
		{name: "completed synced", status: "completed", syncStatus: "synced", want: false},
		{name: "open pending", status: "needs-action", syncStatus: "pending", want: false},
		{name: "open conflict", status: "needs-action", syncStatus: "conflict", want: false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var rendered bytes.Buffer
			err := TaskRow(TaskRowView{
				ID:            "task-detail-complete",
				ProjectID:     "project-1",
				Title:         "Detail completion",
				Status:        tt.status,
				SyncStatus:    tt.syncStatus,
				ServerVersion: 2,
			}).Render(context.Background(), &rendered)
			if err != nil {
				t.Fatalf("render task row: %v", err)
			}
			if got := strings.Contains(rendered.String(), "data-task-detail-complete"); got != tt.want {
				t.Fatalf("detail completion action present = %t, want %t in %s", got, tt.want, rendered.String())
			}
		})
	}
}

func TestTaskRecurrenceSupportsMVPEditablePatterns(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		rule      string
		frequency string
		interval  string
		byDay     string
		end       string
		until     string
		count     string
	}{
		{
			name:      "daily interval without end",
			rule:      "FREQ=DAILY;INTERVAL=3",
			frequency: "DAILY",
			interval:  "3",
			byDay:     "MO",
			end:       "never",
		},
		{
			name:      "weekly interval without end",
			rule:      "FREQ=WEEKLY;INTERVAL=2",
			frequency: "WEEKLY",
			interval:  "2",
			byDay:     "MO",
			end:       "never",
		},
		{
			name:      "weekdays",
			rule:      "FREQ=WEEKLY;BYDAY=MO,TU,WE,TH,FR",
			frequency: "WEEKDAYS",
			interval:  "1",
			byDay:     "MO",
			end:       "never",
		},
		{
			name:      "specific weekday",
			rule:      "FREQ=WEEKLY;BYDAY=WE",
			frequency: "BYDAY",
			interval:  "1",
			byDay:     "WE",
			end:       "never",
		},
		{
			name:      "monthly interval until date",
			rule:      "FREQ=MONTHLY;INTERVAL=2;UNTIL=20260710T235959Z",
			frequency: "MONTHLY",
			interval:  "2",
			byDay:     "MO",
			end:       "until",
			until:     "2026-07-10",
		},
		{
			name:      "yearly count",
			rule:      "FREQ=YEARLY;COUNT=4",
			frequency: "YEARLY",
			interval:  "1",
			byDay:     "MO",
			end:       "count",
			count:     "4",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := taskRecurrence(TaskRowView{RRule: tt.rule})
			if !got.Editable {
				t.Fatalf("expected editable recurrence for %q", tt.rule)
			}
			if got.Frequency != tt.frequency || got.Interval != tt.interval || got.ByDay != tt.byDay || got.End != tt.end || got.Until != tt.until || got.Count != tt.count {
				t.Fatalf("unexpected recurrence editor for %q: %#v", tt.rule, got)
			}
		})
	}
}

func TestTaskRecurrenceSectionIsCollapsedWithCurrentSummary(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		rrule       string
		wantSummary string
		wantComplex bool
	}{
		{name: "none", wantSummary: "Nicht eingerichtet"},
		{name: "editable", rrule: "FREQ=WEEKLY;INTERVAL=2", wantSummary: "Wöchentlich · alle 2"},
		{name: "complex", rrule: "FREQ=MONTHLY;BYDAY=MO,TU,WE,TH,FR;BYSETPOS=1", wantSummary: "Komplexe Wiederholung", wantComplex: true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var rendered bytes.Buffer
			err := TaskRow(TaskRowView{
				ID:            "task-recurrence-" + tt.name,
				ProjectID:     "project-1",
				Title:         "Recurrence " + tt.name,
				Status:        "needs-action",
				SyncStatus:    "synced",
				ServerVersion: 2,
				RRule:         tt.rrule,
			}).Render(context.Background(), &rendered)
			if err != nil {
				t.Fatalf("render task row: %v", err)
			}
			output := rendered.String()
			start := strings.Index(output, `data-task-recurrence-section`)
			if start < 0 {
				t.Fatalf("expected recurrence disclosure in %s", output)
			}
			tagStart := strings.LastIndex(output[:start], "<details")
			tagEnd := strings.Index(output[start:], ">")
			if tagStart < 0 || tagEnd < 0 {
				t.Fatalf("expected complete recurrence details tag in %s", output)
			}
			detailsTag := output[tagStart : start+tagEnd+1]
			if strings.Contains(detailsTag, " open") {
				t.Fatalf("recurrence disclosure must be collapsed by default: %s", detailsTag)
			}
			if !strings.Contains(output, tt.wantSummary) {
				t.Fatalf("expected recurrence summary %q in %s", tt.wantSummary, output)
			}
			if got := strings.Contains(output, "Komplexe Wiederholung wird unverändert erhalten."); got != tt.wantComplex {
				t.Fatalf("complex recurrence explanation present = %t, want %t in %s", got, tt.wantComplex, output)
			}
		})
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

func TestTaskRowRendersConflictLinkWhenConflictIDIsKnown(t *testing.T) {
	t.Parallel()

	component := TaskRow(TaskRowView{
		ID:            "task-conflict",
		Title:         "Konflikt Aufgabe",
		Status:        "needs-action",
		SyncStatus:    "conflict",
		ServerVersion: 5,
		ConflictID:    "conflict-1",
	})

	var rendered bytes.Buffer
	if err := component.Render(context.Background(), &rendered); err != nil {
		t.Fatalf("render task row: %v", err)
	}

	output := rendered.String()
	for _, want := range []string{
		`data-task-conflict-link`,
		`href="/conflicts/conflict-1"`,
		`Konfliktlösung`,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected conflict task link to include %q in %s", want, output)
		}
	}
	if strings.Contains(output, `data-task-detail-open`) || strings.Contains(output, `data-task-detail-dialog`) {
		t.Fatalf("conflict task with conflict id must link directly instead of opening detail: %s", output)
	}
}
