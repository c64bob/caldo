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
