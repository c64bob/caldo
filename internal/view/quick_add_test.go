package view

import (
	"bytes"
	"caldo/internal/parser"
	"context"
	"strings"
	"testing"
)

func TestQuickAddPreviewIncludesCSRFHeaderForSaveForm(t *testing.T) {
	t.Parallel()

	ctx := WithCSRFToken(context.Background(), "token-123")
	component := QuickAddPreview(parser.QuickAddDraft{Title: "Test", ProjectID: "project-1"}, "")

	var rendered bytes.Buffer
	if err := component.Render(ctx, &rendered); err != nil {
		t.Fatalf("render quick add preview: %v", err)
	}

	output := rendered.String()
	if !strings.Contains(output, `hx-post="/tasks"`) {
		t.Fatal("expected quick add save form to use htmx post")
	}

	if !strings.Contains(output, `X-CSRF-Token`) || !strings.Contains(output, `token-123`) {
		t.Fatal("expected quick add save form to include csrf token in htmx headers")
	}
	for _, want := range []string{
		`data-quick-add-corrections`,
		`name="title" value="Test"`,
		`name="labels"`,
		`name="priority"`,
		`name="recurrence"`,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected quick add save form to include %q in %s", want, output)
		}
	}
}

func TestQuickAddOverlayUsesDistinctPreviewTarget(t *testing.T) {
	t.Parallel()

	component := QuickAddOverlay()

	var rendered bytes.Buffer
	if err := component.Render(context.Background(), &rendered); err != nil {
		t.Fatalf("render quick add overlay: %v", err)
	}

	output := rendered.String()
	for _, want := range []string{
		`id="quick-add-overlay"`,
		`data-quick-add-overlay`,
		`data-quick-add-overlay-form`,
		`id="quick-add-overlay-text"`,
		`role="combobox"`,
		`aria-autocomplete="list"`,
		`aria-expanded="false"`,
		`aria-controls="quick-add-token-suggestions"`,
		`hx-target="#quick-add-overlay-preview"`,
		`hx-trigger="input changed delay:200ms from:#quick-add-overlay-text, submit"`,
		`name="surface" value="overlay"`,
		`aria-describedby="quick-add-overlay-error"`,
		`id="quick-add-overlay-error"`,
		`data-quick-add-overlay-error`,
		`role="alert"`,
		`id="quick-add-overlay-preview"`,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected quick add overlay to include %q in %s", want, output)
		}
	}
}

func TestQuickAddOverlayPreviewUsesOverlayTargetAndSaveHook(t *testing.T) {
	t.Parallel()

	ctx := WithCSRFToken(context.Background(), "token-123")
	component := QuickAddOverlayPreview(parser.QuickAddDraft{Title: "Overlay Task", ProjectID: "project-1"}, "")

	var rendered bytes.Buffer
	if err := component.Render(ctx, &rendered); err != nil {
		t.Fatalf("render quick add overlay preview: %v", err)
	}

	output := rendered.String()
	for _, want := range []string{
		`id="quick-add-overlay-preview"`,
		`data-quick-add-overlay-save-form`,
		`hx-post="/tasks"`,
		`X-CSRF-Token`,
		`token-123`,
		`name="title" value="Overlay Task"`,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected quick add overlay preview to include %q in %s", want, output)
		}
	}
	if strings.Contains(output, `id="quick-add-preview"`) {
		t.Fatalf("expected overlay preview to avoid page preview id: %s", output)
	}
}

func TestQuickAddPreviewShowsRecognizedDateAndAmbiguityInVisibleChip(t *testing.T) {
	t.Parallel()

	draft := parser.QuickAddDraft{
		Title:        "Anrufen",
		Due:          "2026-07-15",
		DueSource:    "Mittwoch",
		DueAmbiguous: true,
	}
	var rendered bytes.Buffer
	if err := QuickAddOverlayPreview(draft, "Anrufen Mittwoch").Render(context.Background(), &rendered); err != nil {
		t.Fatalf("render quick add overlay preview: %v", err)
	}

	output := rendered.String()
	visibleChip := `data-quick-add-chips><button type="button" class="caldo-quick-add-chip" data-quick-add-clear="[name='due_date']" data-quick-add-date-chip`
	if !strings.Contains(output, visibleChip) {
		t.Fatalf("expected recognized date resolution in visible chip: %s", output)
	}
	for _, want := range []string{"Mittwoch -&gt; 2026-07-15", `data-quick-add-date-warning`, "Datum prüfen"} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected visible date chip to include %q in %s", want, output)
		}
	}
}

func TestQuickAddPreviewRendersCorrectionChipsAndEditableFields(t *testing.T) {
	t.Parallel()

	ctx := WithCSRFToken(context.Background(), "token-123")
	component := QuickAddPreview(parser.QuickAddDraft{
		Title:     "Review",
		ProjectID: "project-work",
		Project:   "Work",
		ProjectOptions: []parser.QuickAddProjectSuggestion{
			{ID: "project-default", Name: "Inbox"},
			{ID: "project-work", Name: "Work"},
		},
		LabelOptions: []parser.QuickAddLabelSuggestion{
			{Name: "backend"},
			{Name: "review"},
			{Name: "urgent"},
		},
		Labels:     []string{"urgent", "backend"},
		Due:        "2026-06-13",
		DueSource:  "morgen",
		Recurrence: "FREQ=WEEKLY",
		Priority:   "medium",
	}, "")

	var rendered bytes.Buffer
	if err := component.Render(ctx, &rendered); err != nil {
		t.Fatalf("render quick add preview: %v", err)
	}

	output := rendered.String()
	for _, want := range []string{
		`data-quick-add-chips`,
		`class="caldo-quick-add-chip"`,
		`data-quick-add-corrections`,
		`data-quick-add-correction="title"`,
		`name="title" value="Review"`,
		`name="project_selection"`,
		`value="existing:project-work" selected`,
		`name="labels" value="urgent, backend"`,
		`data-quick-add-remove-label="urgent"`,
		`aria-label="Labels urgent entfernen"`,
		`data-quick-add-label-suggestions`,
		`data-quick-add-append-label="review"`,
		`aria-label="Label review hinzufügen"`,
		`id="quick-add-preview-label-options"`,
		`data-quick-add-date-resolution`,
		`morgen`,
		`name="due_date" value="2026-06-13"`,
		`Wöchentlich`,
		`name="recurrence" value="FREQ=WEEKLY"`,
		`data-quick-add-recurrence-input`,
		`data-quick-add-recurrence-error`,
		`Wiederholung prüfen`,
		`P2 Mittel`,
		`caldo-quick-add-priority-p2`,
		`value="medium" selected`,
		`data-quick-add-clear="[name='labels']"`,
		`data-quick-add-clear="[name='due_date']"`,
		`data-quick-add-clear="[name='priority']"`,
		`aria-label="Projekt Work entfernen"`,
		`aria-label="Datum morgen -&gt; 2026-06-13 entfernen"`,
		`role="alert"`,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected correction preview to include %q in %s", want, output)
		}
	}
}

func TestQuickAddPageAssociatesServerError(t *testing.T) {
	t.Parallel()

	component := QuickAddPage(nil, "bad input", "Vorschau konnte nicht erstellt werden.")

	var rendered bytes.Buffer
	if err := component.Render(context.Background(), &rendered); err != nil {
		t.Fatalf("render quick add page: %v", err)
	}

	output := rendered.String()
	for _, want := range []string{
		`aria-describedby="quick-add-page-error"`,
		`id="quick-add-page-error"`,
		`role="alert"`,
		`aria-live="polite"`,
		`Vorschau konnte nicht erstellt werden.`,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected quick add page error to include %q in %s", want, output)
		}
	}
}

func TestQuickAddRecurrenceLabelHumanizesSimplePatterns(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tests := []struct {
		name string
		rule string
		want string
	}{
		{name: "daily", rule: "FREQ=DAILY", want: "Täglich"},
		{name: "weekly", rule: "FREQ=WEEKLY", want: "Wöchentlich"},
		{name: "monthly", rule: "FREQ=MONTHLY", want: "Monatlich"},
		{name: "yearly", rule: "FREQ=YEARLY", want: "Jährlich"},
		{name: "weekdays", rule: "FREQ=WEEKLY;BYDAY=MO,TU,WE,TH,FR", want: "Werktags"},
		{name: "weekday", rule: "FREQ=WEEKLY;BYDAY=MO", want: "Jeden Montag"},
		{name: "interval", rule: "FREQ=DAILY;INTERVAL=3", want: "Alle 3 Tage"},
		{name: "complex", rule: "FREQ=MONTHLY;BYDAY=MO,TU,WE,TH,FR;BYSETPOS=1", want: "RRULE: FREQ=MONTHLY;BYDAY=MO,TU,WE,TH,FR;BYSETPOS=1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := quickAddRecurrenceLabel(ctx, tt.rule); got != tt.want {
				t.Fatalf("recurrence label: got %q want %q", got, tt.want)
			}
		})
	}
}

func TestQuickAddRecurrenceLabelUsesEnglishLanguage(t *testing.T) {
	t.Parallel()

	ctx := WithUIPreferences(context.Background(), "en", "system")
	if got := quickAddRecurrenceLabel(ctx, "FREQ=WEEKLY;BYDAY=FR"); got != "Every Friday" {
		t.Fatalf("english weekday recurrence label: got %q", got)
	}
	if got := quickAddRecurrenceLabel(ctx, "FREQ=MONTHLY;INTERVAL=2"); got != "Every 2 months" {
		t.Fatalf("english interval recurrence label: got %q", got)
	}
}

func TestQuickAddPreviewRendersAmbiguousDateWarning(t *testing.T) {
	t.Parallel()

	component := QuickAddPreview(parser.QuickAddDraft{
		Title:        "Review",
		Due:          "2026-06-17",
		DueSource:    "Mittwoch",
		DueAmbiguous: true,
	}, "")

	var rendered bytes.Buffer
	if err := component.Render(context.Background(), &rendered); err != nil {
		t.Fatalf("render quick add preview: %v", err)
	}

	output := rendered.String()
	for _, want := range []string{
		`data-quick-add-date-resolution`,
		`data-quick-add-date-warning`,
		`Datum prüfen`,
		`Mittwoch`,
		`2026-06-17`,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected ambiguous date preview to include %q in %s", want, output)
		}
	}
}

func TestQuickAddPreviewRendersUnknownProjectSuggestionsAndCreateOption(t *testing.T) {
	t.Parallel()

	component := QuickAddPreview(parser.QuickAddDraft{
		Title:      "Test",
		Project:    "Work",
		ProjectNew: true,
		ProjectSuggestions: []parser.QuickAddProjectSuggestion{
			{ID: "project-work", Name: "Work Inbox"},
		},
	}, "")

	var rendered bytes.Buffer
	if err := component.Render(context.Background(), &rendered); err != nil {
		t.Fatalf("render quick add preview: %v", err)
	}

	output := rendered.String()
	for _, want := range []string{
		`Projektvorschläge`,
		`data-quick-add-project-suggestions`,
		`data-quick-add-project-suggestion`,
		`name="project_new_name" value="Work"`,
		`name="project_selection" value="existing:project-work" required`,
		`Work Inbox`,
		`name="project_selection" value="create" required`,
		`Neu anlegen`,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected quick add project create option %q in %s", want, output)
		}
	}
	if strings.Contains(output, `value="create" checked`) {
		t.Fatalf("expected project creation to require explicit selection: %s", output)
	}
	if strings.Contains(output, `name="create_project"`) {
		t.Fatalf("expected project selection radios instead of legacy create checkbox: %s", output)
	}
}
