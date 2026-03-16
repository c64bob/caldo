package handlers

import (
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"caldo/internal/caldav"
	"caldo/internal/http/middleware"
	"caldo/internal/service"
)

func (h *TasksHandler) APITaskCreate(w http.ResponseWriter, r *http.Request) {
	h.mutateTask(w, r, func(principal string) error {
		_, err := h.Service.CreateTask(r.Context(), principal, service.TaskMutationInput{
			ListID:      strings.TrimSpace(r.FormValue("list_id")),
			UID:         strings.TrimSpace(r.FormValue("uid")),
			Summary:     strings.TrimSpace(r.FormValue("summary")),
			Status:      strings.TrimSpace(r.FormValue("status")),
			Priority:    service.ParsePriority(r.FormValue("priority")),
			Description: strings.TrimSpace(r.FormValue("description")),
			Categories:  service.ParseCategories(r.FormValue("categories")),
			Due:         parseDueOrNil(r.FormValue("due")),
			DueKind:     parseDueKind(r.FormValue("due")),
		})
		return err
	})
}

func (h *TasksHandler) APITaskUpdate(w http.ResponseWriter, r *http.Request) {
	h.mutateTask(w, r, func(principal string) error {
		_, err := h.Service.UpdateTask(r.Context(), principal, service.TaskMutationInput{
			ListID:      strings.TrimSpace(r.FormValue("list_id")),
			UID:         strings.TrimSpace(r.FormValue("uid")),
			Href:        strings.TrimSpace(r.FormValue("href")),
			ETag:        strings.TrimSpace(r.FormValue("etag")),
			Summary:     strings.TrimSpace(r.FormValue("summary")),
			Status:      strings.TrimSpace(r.FormValue("status")),
			Priority:    service.ParsePriority(r.FormValue("priority")),
			Description: strings.TrimSpace(r.FormValue("description")),
			Categories:  service.ParseCategories(r.FormValue("categories")),
			Due:         parseDueOrNil(r.FormValue("due")),
			DueKind:     parseDueKind(r.FormValue("due")),
		})
		return err
	})
}

func (h *TasksHandler) APITaskDelete(w http.ResponseWriter, r *http.Request) {
	h.mutateTask(w, r, func(principal string) error {
		return h.Service.DeleteTask(r.Context(), principal, service.TaskMutationInput{
			ListID: strings.TrimSpace(r.FormValue("list_id")),
			Href:   strings.TrimSpace(r.FormValue("href")),
			ETag:   strings.TrimSpace(r.FormValue("etag")),
		})
	})
}

func (h *TasksHandler) mutateTask(w http.ResponseWriter, r *http.Request, fn func(principal string) error) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Ungültiges Formular", http.StatusBadRequest)
		return
	}
	principal, ok := middleware.PrincipalFromContext(r.Context())
	if !ok {
		http.Error(w, "Authentifizierungsheader fehlt", http.StatusUnauthorized)
		return
	}
	if err := fn(principal); err != nil {
		message, status := taskMutationError(err)
		http.Error(w, message, status)
		return
	}
	listID := strings.TrimSpace(r.FormValue("list_id"))
	redirectTarget := "/tasks"
	if listID != "" {
		redirectTarget += "?list=" + url.QueryEscape(listID)
	}
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", redirectTarget)
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, redirectTarget, http.StatusSeeOther)
}

func parseDueOrNil(raw string) *time.Time {
	due, _ := service.ParseDue(raw)
	return due
}

func parseDueKind(raw string) string {
	_, dueKind := service.ParseDue(raw)
	return dueKind
}

func taskMutationError(err error) (string, int) {
	if errors.Is(err, caldav.ErrPreconditionFailed) {
		return "Konflikt (412): Aufgabe wurde am Server geändert. Bitte neu laden und erneut speichern.", http.StatusConflict
	}
	if errors.Is(err, caldav.ErrMissingETag) || errors.Is(err, caldav.ErrInvalidTaskHref) {
		return "Ungültige Task-Parameter", http.StatusBadRequest
	}
	message := strings.ToLower(err.Error())
	if strings.Contains(message, "nicht erreichbar") {
		return "Server nicht erreichbar. Bitte URL/Netzwerk prüfen.", http.StatusBadGateway
	}
	if strings.Contains(message, "tls") || strings.Contains(message, "x509") {
		return "TLS-Fehler bei der CalDAV-Verbindung. Zertifikat/Truststore prüfen.", http.StatusBadGateway
	}
	if strings.Contains(message, "unauthorized") || strings.Contains(message, "forbidden") || strings.Contains(message, "anmeldung") {
		return "Authentifizierung fehlgeschlagen. Bitte Zugangsdaten prüfen.", http.StatusBadGateway
	}
	return "Task-Änderung fehlgeschlagen", http.StatusBadGateway
}
