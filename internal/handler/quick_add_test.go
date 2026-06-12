package handler

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestQuickAddPageUsesBaseLayout(t *testing.T) {
	h := QuickAddPage(quickAddDependencies{})
	req := httptest.NewRequest(http.MethodGet, "/quick-add", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "caldo-app-shell") || !strings.Contains(body, "caldo-sidebar") {
		t.Fatalf("expected quick add page to render in app shell: %s", body)
	}
	if !strings.Contains(body, `href="/quick-add"`) || !strings.Contains(body, "Quick Add") {
		t.Fatalf("expected quick add page links and content: %s", body)
	}
}

func TestQuickAddPreviewUsesDefaultProject(t *testing.T) {
	database := openSQLiteForTaskCreateHandlerTest(t)
	seedTaskCreateHandlerProject(t, database)
	h := QuickAddPreview(quickAddDependencies{database: database})

	form := url.Values{}
	form.Set("text", "Neue Aufgabe")
	req := httptest.NewRequest(http.MethodPost, "/quick-add/preview", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "Neue Aufgabe") || !strings.Contains(body, "Inbox") {
		t.Fatalf("missing preview fields: %s", body)
	}
}

func TestQuickAddPreviewUsesOverlaySurface(t *testing.T) {
	database := openSQLiteForTaskCreateHandlerTest(t)
	seedTaskCreateHandlerProject(t, database)
	h := QuickAddPreview(quickAddDependencies{database: database})

	form := url.Values{}
	form.Set("text", "Overlay Aufgabe")
	form.Set("surface", "overlay")
	req := httptest.NewRequest(http.MethodPost, "/quick-add/preview", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", w.Code)
	}
	body := w.Body.String()
	for _, want := range []string{
		`id="quick-add-overlay-preview"`,
		`data-quick-add-overlay-save-form`,
		`name="title" value="Overlay Aufgabe"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected overlay preview to include %q in %s", want, body)
		}
	}
	if strings.Contains(body, `id="quick-add-preview"`) {
		t.Fatalf("expected overlay preview response to avoid page preview id: %s", body)
	}
}

func TestQuickAddPreviewUsesPersistedUILanguage(t *testing.T) {
	database := openSQLiteForTaskCreateHandlerTest(t)
	seedTaskCreateHandlerProject(t, database)
	if _, err := database.Conn.Exec(`UPDATE settings SET ui_language = 'en', dark_mode = 'light' WHERE id = 'default';`); err != nil {
		t.Fatalf("set ui language: %v", err)
	}
	h := QuickAddPreview(quickAddDependencies{database: database})

	form := url.Values{}
	form.Set("text", "Call tomorrow")
	req := httptest.NewRequest(http.MethodPost, "/quick-add/preview", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, `name="title" value="Call"`) {
		t.Fatalf("expected English natural date token to be removed from title: %s", body)
	}
	if strings.Contains(body, `name="due_date" value=""`) {
		t.Fatalf("expected English natural date token to set due date: %s", body)
	}
	if !strings.Contains(body, `Date`) || !strings.Contains(body, `Preview`) {
		t.Fatalf("expected preview labels to use English UI language: %s", body)
	}
}

func TestQuickAddPreviewUsesProjectTokenWhenProjectExists(t *testing.T) {
	database := openSQLiteForTaskCreateHandlerTest(t)
	seedTaskCreateHandlerProject(t, database)
	h := QuickAddPreview(quickAddDependencies{database: database})

	form := url.Values{}
	form.Set("text", "Neue Aufgabe #Inbox")
	req := httptest.NewRequest(http.MethodPost, "/quick-add/preview", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Inbox") {
		t.Fatalf("expected inbox project in preview, got body: %s", w.Body.String())
	}
}

func TestQuickAddPreviewMarksUnknownProjectTokenAsNew(t *testing.T) {
	database := openSQLiteForTaskCreateHandlerTest(t)
	seedTaskCreateHandlerProject(t, database)
	h := QuickAddPreview(quickAddDependencies{database: database})

	form := url.Values{}
	form.Set("text", "Neue Aufgabe #Unbekannt")
	req := httptest.NewRequest(http.MethodPost, "/quick-add/preview", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", w.Code)
	}
	body := w.Body.String()
	for _, want := range []string{
		"Unbekannt",
		"Neu anlegen",
		"Projektvorschläge",
		`data-quick-add-project-suggestions`,
		`data-quick-add-project-suggestion`,
		`name="project_new_name"`,
		`name="project_selection" value="existing:project-default"`,
		`name="project_selection" value="create"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected unknown project preview to include %q in %s", want, body)
		}
	}
	if strings.Contains(body, `name="create_project"`) {
		t.Fatalf("expected project selection radios instead of legacy create checkbox, got body: %s", body)
	}
}

func TestQuickAddPreviewShowsLabelsAndPriorityTokens(t *testing.T) {
	database := openSQLiteForTaskCreateHandlerTest(t)
	seedTaskCreateHandlerProject(t, database)
	h := QuickAddPreview(quickAddDependencies{database: database})

	form := url.Values{}
	form.Set("text", "Neue Aufgabe @urgent @backend !2")
	req := httptest.NewRequest(http.MethodPost, "/quick-add/preview", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", w.Code)
	}
	body := w.Body.String()
	for _, want := range []string{`urgent, backend`, `value="urgent, backend"`, `name="priority"`, `value="medium" selected`} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected quick add token preview to include %q in %s", want, body)
		}
	}
}

func TestQuickAddPreviewIncludesEditableProjectOptions(t *testing.T) {
	database := openSQLiteForTaskCreateHandlerTest(t)
	seedTaskCreateHandlerProject(t, database)
	h := QuickAddPreview(quickAddDependencies{database: database})

	form := url.Values{}
	form.Set("text", "Neue Aufgabe #Inbox")
	req := httptest.NewRequest(http.MethodPost, "/quick-add/preview", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", w.Code)
	}
	body := w.Body.String()
	for _, want := range []string{
		`name="project_selection"`,
		`value="existing:project-default" selected`,
		`data-quick-add-correction="project"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected editable project options to include %q in %s", want, body)
		}
	}
}

func TestQuickAddPreviewRequiresTitle(t *testing.T) {
	database := openSQLiteForTaskCreateHandlerTest(t)
	h := QuickAddPreview(quickAddDependencies{database: database})
	req := httptest.NewRequest(http.MethodPost, "/quick-add/preview", strings.NewReader("text=   "))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status: %d", w.Code)
	}
}
