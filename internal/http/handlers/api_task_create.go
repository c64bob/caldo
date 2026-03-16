package handlers

import (
	"errors"
	"net/http"
	"net/url"
	"strings"

	"caldo/internal/caldav"
	"caldo/internal/http/middleware"
	"caldo/internal/service"
)

func (h *TasksHandler) APITaskCreate(w http.ResponseWriter, r *http.Request) {
	h.mutateTask(w, r, func(principal string) error {
		_, err := h.Service.CreateTask(r.Context(), principal, service.TaskMutationInput{
			ListID:   strings.TrimSpace(r.FormValue("list_id")),
			UID:      strings.TrimSpace(r.FormValue("uid")),
			Summary:  strings.TrimSpace(r.FormValue("summary")),
			Status:   strings.TrimSpace(r.FormValue("status")),
			Priority: service.ParsePriority(r.FormValue("priority")),
		})
		return err
	})
}

func (h *TasksHandler) APITaskUpdate(w http.ResponseWriter, r *http.Request) {
	h.mutateTask(w, r, func(principal string) error {
		_, err := h.Service.UpdateTask(r.Context(), principal, service.TaskMutationInput{
			ListID:   strings.TrimSpace(r.FormValue("list_id")),
			UID:      strings.TrimSpace(r.FormValue("uid")),
			Href:     strings.TrimSpace(r.FormValue("href")),
			ETag:     strings.TrimSpace(r.FormValue("etag")),
			Summary:  strings.TrimSpace(r.FormValue("summary")),
			Status:   strings.TrimSpace(r.FormValue("status")),
			Priority: service.ParsePriority(r.FormValue("priority")),
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
		if errors.Is(err, caldav.ErrPreconditionFailed) {
			http.Error(w, "Konflikt erkannt: Aufgabe wurde auf dem Server geändert. Bitte neu laden.", http.StatusConflict)
			return
		}
		if errors.Is(err, caldav.ErrMissingETag) || errors.Is(err, caldav.ErrInvalidTaskHref) {
			http.Error(w, "Ungültige Task-Parameter", http.StatusBadRequest)
			return
		}
		http.Error(w, "Task-Änderung fehlgeschlagen", http.StatusBadGateway)
		return
	}
	listID := strings.TrimSpace(r.FormValue("list_id"))
	redirectTarget := "/tasks"
	if listID != "" {
		redirectTarget += "?list=" + url.QueryEscape(listID)
	}
	w.Header().Set("HX-Redirect", redirectTarget)
	w.WriteHeader(http.StatusOK)
}
