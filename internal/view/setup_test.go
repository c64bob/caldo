package view

import (
	"bytes"
	"caldo/internal/caldav"
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

func TestSetupCalendarsContentIncludesSelectionAndDefaultControls(t *testing.T) {
	t.Parallel()

	ctx := WithCSRFToken(context.Background(), "token-123")
	component := SetupCalendarsContent([]caldav.Calendar{
		{Href: "/cal/work/", DisplayName: "Work"},
		{Href: "/cal/home/", DisplayName: "Home"},
	}, "", nil)

	var rendered bytes.Buffer
	if err := component.Render(ctx, &rendered); err != nil {
		t.Fatalf("render setup calendars content: %v", err)
	}

	output := rendered.String()
	for _, want := range []string{
		`hx-post="/setup/calendars"`,
		`name="calendar_href"`,
		`name="default_calendar_href"`,
		`name="new_default_project_name"`,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected setup calendars content to include %q", want)
		}
	}
}

func TestSetupImportContentIncludesControllerHooks(t *testing.T) {
	t.Parallel()

	component := SetupImportContent("")

	var rendered bytes.Buffer
	if err := component.Render(context.Background(), &rendered); err != nil {
		t.Fatalf("render setup import content: %v", err)
	}

	output := rendered.String()
	for _, want := range []string{
		`data-setup-import`,
		`data-setup-import-start-url="/setup/import"`,
		`data-setup-import-complete-url="/setup/complete"`,
		`data-setup-import-status`,
		`data-setup-import-progress-bar`,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected setup import content to include %q", want)
		}
	}
}
