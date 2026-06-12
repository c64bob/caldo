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
	if !strings.Contains(output, `name="labels"`) || !strings.Contains(output, `name="priority"`) || !strings.Contains(output, `name="recurrence"`) {
		t.Fatal("expected quick add save form to include labels, priority, and recurrence fields")
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
		`hx-target="#quick-add-overlay-preview"`,
		`name="surface" value="overlay"`,
		`data-quick-add-overlay-error hidden`,
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
		`name="project_selection" value="existing:project-work"`,
		`Work Inbox`,
		`name="project_selection" value="create"`,
		`Neu anlegen`,
		`checked`,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected quick add project create option %q in %s", want, output)
		}
	}
	if strings.Contains(output, `name="create_project"`) {
		t.Fatalf("expected project selection radios instead of legacy create checkbox: %s", output)
	}
}
