package view

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestProjectsOverviewPageRendersCreateForm(t *testing.T) {
	t.Parallel()

	ctx := WithCSRFToken(context.Background(), "token-123")
	var output bytes.Buffer
	err := ProjectsOverviewPage([]NavigationOverviewItem{{
		Name:     "Work",
		Href:     "/search?q=%23Work",
		Count:    0,
		HasCount: true,
		Meta:     "Offene Aufgaben",
	}}, "", "").Render(ctx, &output)
	if err != nil {
		t.Fatalf("render projects page: %v", err)
	}

	body := output.String()
	for _, want := range []string{
		`data-project-create-form`,
		`method="post"`,
		`action="/projects"`,
		`hx-post="/projects"`,
		`hx-target="body"`,
		`hx-swap="outerHTML"`,
		`X-CSRF-Token&#34;:&#34;token-123`,
		`name="display_name"`,
		`Projekt anlegen`,
		`data-navigation-overview`,
		`Work`,
		`0 offen`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("projects page missing %q in %s", want, body)
		}
	}
}

func TestProjectsOverviewPageRendersCreateError(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	err := ProjectsOverviewPage(nil, "projekt konnte nicht angelegt werden", "New Project").Render(context.Background(), &output)
	if err != nil {
		t.Fatalf("render projects page: %v", err)
	}

	body := output.String()
	for _, want := range []string{
		`data-project-create-error`,
		`projekt konnte nicht angelegt werden`,
		`value="New Project"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("projects page missing %q in %s", want, body)
		}
	}
}
