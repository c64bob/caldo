package view

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestSetupCalDAVContentIncludesCSRFHeaderForHTMXSubmit(t *testing.T) {
	t.Parallel()

	ctx := WithCSRFToken(context.Background(), "token-123")
	component := SetupCalDAVContent("")

	var rendered bytes.Buffer
	if err := component.Render(ctx, &rendered); err != nil {
		t.Fatalf("render setup caldav content: %v", err)
	}

	output := rendered.String()
	if !strings.Contains(output, `hx-post="/setup/caldav"`) {
		t.Fatal("expected setup form to use htmx post")
	}

	if !strings.Contains(output, `hx-headers='{"X-CSRF-Token":"token-123"}'`) {
		t.Fatal("expected setup form to include csrf token in htmx headers")
	}
}
