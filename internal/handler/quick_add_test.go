package handler

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

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
	if !strings.Contains(body, "Unbekannt") || !strings.Contains(body, "wird beim Speichern ignoriert") {
		t.Fatalf("expected unknown project warning in preview, got body: %s", body)
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
