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

func TestQuickAddPreviewRendersUnknownProjectCreateOption(t *testing.T) {
	t.Parallel()

	component := QuickAddPreview(parser.QuickAddDraft{Title: "Test", Project: "Work", ProjectNew: true}, "")

	var rendered bytes.Buffer
	if err := component.Render(context.Background(), &rendered); err != nil {
		t.Fatalf("render quick add preview: %v", err)
	}

	output := rendered.String()
	for _, want := range []string{`Neu anlegen`, `name="project_new_name" value="Work"`, `name="create_project" value="1"`, `checked`} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected quick add project create option %q in %s", want, output)
		}
	}
}
