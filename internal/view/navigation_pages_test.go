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

func TestProjectsOverviewPageRendersRenameForm(t *testing.T) {
	t.Parallel()

	ctx := WithCSRFToken(context.Background(), "token-123")
	var output bytes.Buffer
	err := ProjectsOverviewPage([]NavigationOverviewItem{{
		ID:            "project-1",
		Name:          "Work",
		Href:          "/search?q=%23Work",
		Count:         2,
		HasCount:      true,
		Meta:          "Offene Aufgaben",
		ServerVersion: 3,
	}}, "", "").Render(ctx, &output)
	if err != nil {
		t.Fatalf("render projects page: %v", err)
	}

	body := output.String()
	for _, want := range []string{
		`data-project-rename-form`,
		`action="/projects/project-1"`,
		`hx-patch="/projects/project-1"`,
		`name="expected_version" value="3"`,
		`name="display_name" value="Work"`,
		`Umbenennen`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("projects page missing %q in %s", want, body)
		}
	}
}

func TestProjectsOverviewPageRendersRenameError(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	err := ProjectsOverviewPage([]NavigationOverviewItem{{
		ID:            "project-1",
		Name:          "Work",
		Href:          "/search?q=%23Work",
		Count:         2,
		HasCount:      true,
		Meta:          "Offene Aufgaben",
		ServerVersion: 3,
		RenameError:   "projekt konnte nicht umbenannt werden",
		RenameValue:   "New Work",
	}}, "", "").Render(context.Background(), &output)
	if err != nil {
		t.Fatalf("render projects page: %v", err)
	}

	body := output.String()
	for _, want := range []string{
		`data-project-rename-error`,
		`projekt konnte nicht umbenannt werden`,
		`value="New Work"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("projects page missing %q in %s", want, body)
		}
	}
}

func TestProjectsOverviewPageRendersDeleteForm(t *testing.T) {
	t.Parallel()

	ctx := WithCSRFToken(context.Background(), "token-123")
	var output bytes.Buffer
	err := ProjectsOverviewPage([]NavigationOverviewItem{{
		ID:              "project-1",
		Name:            "Work",
		Href:            "/search?q=%23Work",
		Count:           2,
		HasCount:        true,
		Meta:            "Offene Aufgaben",
		ServerVersion:   3,
		DeleteTaskCount: 5,
	}}, "", "").Render(ctx, &output)
	if err != nil {
		t.Fatalf("render projects page: %v", err)
	}

	body := output.String()
	for _, want := range []string{
		`data-project-delete-form`,
		`action="/projects/project-1"`,
		`hx-delete="/projects/project-1"`,
		`name="expected_version" value="3"`,
		`name="confirmation_name"`,
		`Zum Löschen Work eingeben. 5 Aufgaben betroffen.`,
		`Endgültig löschen`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("projects page missing %q in %s", want, body)
		}
	}
}

func TestProjectsOverviewPageRendersDeleteError(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	err := ProjectsOverviewPage([]NavigationOverviewItem{{
		ID:              "project-1",
		Name:            "Work",
		Href:            "/search?q=%23Work",
		Count:           2,
		HasCount:        true,
		Meta:            "Offene Aufgaben",
		ServerVersion:   3,
		DeleteTaskCount: 2,
		DeleteError:     "projekt konnte nicht gelöscht werden",
		DeleteValue:     "Wrok",
	}}, "", "").Render(context.Background(), &output)
	if err != nil {
		t.Fatalf("render projects page: %v", err)
	}

	body := output.String()
	for _, want := range []string{
		`data-project-delete-error`,
		`projekt konnte nicht gelöscht werden`,
		`value="Wrok"`,
		`open`,
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
