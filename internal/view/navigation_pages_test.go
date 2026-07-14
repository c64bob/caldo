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
	}}, ProjectFeedback{}).Render(ctx, &output)
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
	}}, ProjectFeedback{}).Render(ctx, &output)
	if err != nil {
		t.Fatalf("render projects page: %v", err)
	}

	body := output.String()
	for _, want := range []string{
		`data-project-drop-target`,
		`data-project-id="project-1"`,
		`data-project-name="Work"`,
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
	}}, ProjectFeedback{}).Render(context.Background(), &output)
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

func TestProjectsOverviewPageRendersProjectSuccessMessages(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	err := ProjectsOverviewPage([]NavigationOverviewItem{{
		ID:            "project-1",
		Name:          "Work",
		Href:          "/projects/project-1",
		Count:         2,
		HasCount:      true,
		Meta:          "Offene Aufgaben",
		ServerVersion: 3,
		RenameSuccess: "projekt wurde umbenannt",
	}}, ProjectFeedback{
		PageSuccess:   "projekt wurde gelöscht",
		CreateSuccess: "projekt wurde angelegt",
	}).Render(context.Background(), &output)
	if err != nil {
		t.Fatalf("render projects page: %v", err)
	}

	body := output.String()
	for _, want := range []string{
		`data-project-page-success`,
		`projekt wurde gelöscht`,
		`data-project-create-success`,
		`projekt wurde angelegt`,
		`data-project-rename-success`,
		`projekt wurde umbenannt`,
		`caldo-alert-success`,
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
	}}, ProjectFeedback{}).Render(ctx, &output)
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
		`Dieses Projekt wird auf dem CalDAV-Server gelöscht. 5 Aufgaben werden lokal entfernt. Zum Löschen Work eingeben.`,
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
	}}, ProjectFeedback{}).Render(context.Background(), &output)
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
	err := ProjectsOverviewPage(nil, ProjectFeedback{CreateError: "projekt konnte nicht angelegt werden", CreateValue: "New Project"}).Render(context.Background(), &output)
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

func TestNavigationOverviewPageRendersLabelLinksAndCounters(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	err := NavigationOverviewPage("Labels", "Keine Labels", []NavigationOverviewItem{{
		ID:       "label-long",
		Name:     "ein-sehr-langes-label-ohne-trennzeichen-1234567890",
		Href:     "/labels/label-long",
		Count:    12,
		HasCount: true,
		Meta:     "30 Aufgaben gesamt",
	}}).Render(context.Background(), &output)
	if err != nil {
		t.Fatalf("render labels page: %v", err)
	}

	body := output.String()
	for _, want := range []string{
		`href="/labels/label-long"`,
		`caldo-list-link`,
		`ein-sehr-langes-label-ohne-trennzeichen-1234567890`,
		`30 Aufgaben gesamt`,
		`12 offen`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("labels page missing %q in %s", want, body)
		}
	}
}

func TestLabelsOverviewPageRendersManagementForms(t *testing.T) {
	t.Parallel()

	ctx := WithCSRFToken(context.Background(), "token-123")
	var output bytes.Buffer
	err := LabelsOverviewPage([]NavigationOverviewItem{{
		ID:              "label-1",
		Name:            "Büro",
		Href:            "/labels/label-1",
		Count:           1,
		HasCount:        true,
		Meta:            "2 Aufgaben gesamt",
		DeleteTaskCount: 2,
	}}, LabelFeedback{}).Render(ctx, &output)
	if err != nil {
		t.Fatalf("render labels page: %v", err)
	}

	body := output.String()
	for _, want := range []string{
		`data-label-management-list`,
		`data-label-rename-form`,
		`action="/labels/label-1"`,
		`hx-patch="/labels/label-1"`,
		`name="name" value="Büro"`,
		`Label umbenennen`,
		`data-label-delete-open`,
		`aria-haspopup="dialog"`,
		`data-label-delete-dialog`,
		`data-label-delete-form`,
		`hx-delete="/labels/label-1"`,
		`hx-sync="this:drop"`,
		`name="confirmed" value="true"`,
		`Möchtest du das Label „Büro“ wirklich löschen? Es wird aus 2 Aufgaben entfernt.`,
		`>Nein</button>`,
		`>Ja, löschen</button>`,
		`X-CSRF-Token&#34;:&#34;token-123`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("labels page missing %q in %s", want, body)
		}
	}
	if strings.Contains(body, `name="confirmation_name"`) || strings.Contains(body, `data-label-delete-confirmation`) {
		t.Fatalf("labels page still renders typed-name confirmation in %s", body)
	}
}

func TestLabelDeleteConfirmationTextDescribesAffectedTasks(t *testing.T) {
	t.Parallel()

	tests := []struct {
		count int
		want  string
	}{
		{count: 0, want: `Möchtest du das Label „Büro“ wirklich löschen? Es wird aus keiner Aufgabe entfernt.`},
		{count: 1, want: `Möchtest du das Label „Büro“ wirklich löschen? Es wird aus 1 Aufgabe entfernt.`},
		{count: 3, want: `Möchtest du das Label „Büro“ wirklich löschen? Es wird aus 3 Aufgaben entfernt.`},
	}

	for _, test := range tests {
		item := NavigationOverviewItem{Name: "Büro", DeleteTaskCount: test.count}
		if got := labelDeleteConfirmationText(item); got != test.want {
			t.Errorf("count %d: got %q, want %q", test.count, got, test.want)
		}
	}
}

func TestLabelsOverviewPageRendersFeedback(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	err := LabelsOverviewPage([]NavigationOverviewItem{{
		ID:              "label-1",
		Name:            "Büro",
		Href:            "/labels/label-1",
		Meta:            "1 Aufgabe gesamt",
		DeleteTaskCount: 1,
		RenameError:     "label konnte nicht umbenannt werden",
		RenameValue:     "Buro",
		DeleteError:     "bestätigung stimmt nicht überein",
	}}, LabelFeedback{PageSuccess: "label wurde gelöscht"}).Render(context.Background(), &output)
	if err != nil {
		t.Fatalf("render labels page: %v", err)
	}

	body := output.String()
	for _, want := range []string{
		`data-label-page-success`,
		`label wurde gelöscht`,
		`data-label-rename-error`,
		`label konnte nicht umbenannt werden`,
		`value="Buro"`,
		`data-label-delete-error`,
		`bestätigung stimmt nicht überein`,
		`data-label-delete-reopen`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("labels page missing %q in %s", want, body)
		}
	}
}
