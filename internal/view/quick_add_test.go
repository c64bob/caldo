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

	if !strings.Contains(output, `hx-headers='{"X-CSRF-Token":"token-123"}'`) {
		t.Fatal("expected quick add save form to include csrf token in htmx headers")
	}
	if !strings.Contains(output, `name="labels"`) || !strings.Contains(output, `name="priority"`) {
		t.Fatal("expected quick add save form to include labels and priority fields")
	}
}
