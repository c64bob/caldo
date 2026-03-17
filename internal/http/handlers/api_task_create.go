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
			ListID:          strings.TrimSpace(r.FormValue("list_id")),
			UID:             strings.TrimSpace(r.FormValue("uid")),
			Summary:         strings.TrimSpace(r.FormValue("summary")),
			Status:          strings.TrimSpace(r.FormValue("status")),
			Priority:        service.ParsePriority(r.FormValue("priority")),
			Description:     strings.TrimSpace(r.FormValue("description")),
			Categories:      service.ParseCategories(r.FormValue("categories")),
			CategoriesSet:   formFieldProvided(r, "categories"),
			PercentComplete: service.ParsePercentComplete(r.FormValue("percent_complete")),
			ParentUID:       strings.TrimSpace(r.FormValue("parent_uid")),
			Goal:            strings.TrimSpace(r.FormValue("goal")),
			Due:             parseDueOrNil(r.FormValue("due")),
			DueKind:         parseDueKind(r.FormValue("due")),
		})
		return err
	})
}

func (h *TasksHandler) APITaskQuickAdd(w http.ResponseWriter, r *http.Request) {
	h.mutateTask(w, r, func(principal string) error {
		in, err := service.ParseSmartAdd(r.FormValue("smart_add"))
		if err != nil {
			return err
		}
		if in.ListID == "" {
			in.ListID = strings.TrimSpace(r.FormValue("list_id"))
		}
		_, err = h.Service.CreateTask(r.Context(), principal, in)
		return err
	})
}

func (h *TasksHandler) APITaskUpdate(w http.ResponseWriter, r *http.Request) {
	h.mutateTask(w, r, func(principal string) error {
		_, err := h.Service.UpdateTask(r.Context(), principal, service.TaskMutationInput{
			ListID:          strings.TrimSpace(r.FormValue("list_id")),
			UID:             strings.TrimSpace(r.FormValue("uid")),
			Href:            strings.TrimSpace(r.FormValue("href")),
			ETag:            strings.TrimSpace(r.FormValue("etag")),
			Summary:         strings.TrimSpace(r.FormValue("summary")),
			Status:          strings.TrimSpace(r.FormValue("status")),
			Priority:        service.ParsePriority(r.FormValue("priority")),
			Description:     strings.TrimSpace(r.FormValue("description")),
			Categories:      service.ParseCategories(r.FormValue("categories")),
			CategoriesSet:   formFieldProvided(r, "categories"),
			PercentComplete: service.ParsePercentComplete(r.FormValue("percent_complete")),
			ParentUID:       strings.TrimSpace(r.FormValue("parent_uid")),
			Goal:            strings.TrimSpace(r.FormValue("goal")),
			Due:             parseDueOrNil(r.FormValue("due")),
			DueKind:         parseDueKind(r.FormValue("due")),
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
	view := activeView(r.FormValue("view"))
	contextValue := strings.TrimSpace(r.FormValue("context"))
	goalValue := strings.TrimSpace(r.FormValue("goal"))
	query := strings.TrimSpace(r.FormValue("q"))
	refSource := strings.TrimSpace(r.Header.Get("HX-Current-URL"))
	if refSource == "" {
		refSource = strings.TrimSpace(r.Referer())
	}
	if ref := refSource; ref != "" {
		if u, err := url.Parse(ref); err == nil {
			if listID == "" {
				listID = strings.TrimSpace(u.Query().Get("list"))
			}
			if view == "main" && strings.TrimSpace(r.FormValue("view")) == "" {
				view = activeView(u.Query().Get("view"))
			}
			if contextValue == "" {
				contextValue = strings.TrimSpace(u.Query().Get("context"))
			}
			if goalValue == "" {
				goalValue = strings.TrimSpace(u.Query().Get("goal"))
			}
			if query == "" {
				query = strings.TrimSpace(u.Query().Get("q"))
			}
		}
	}
	redirectTarget := "/tasks"
	if listID != "" {
		redirectTarget += "?list=" + url.QueryEscape(listID)
	}
	queryParts := make([]string, 0, 4)
	if view != "" {
		queryParts = append(queryParts, "view="+url.QueryEscape(view))
	}
	if contextValue != "" {
		queryParts = append(queryParts, "context="+url.QueryEscape(contextValue))
	}
	if goalValue != "" {
		queryParts = append(queryParts, "goal="+url.QueryEscape(goalValue))
	}
	if query != "" {
		queryParts = append(queryParts, "q="+url.QueryEscape(query))
	}
	if len(queryParts) > 0 {
		sep := "?"
		if strings.Contains(redirectTarget, "?") {
			sep = "&"
		}
		redirectTarget += sep + strings.Join(queryParts, "&")
	}
	if r.Header.Get("HX-Request") == "true" {
		if h.Templates == nil || h.Service == nil {
			w.Header().Set("HX-Redirect", redirectTarget)
			w.WriteHeader(http.StatusOK)
			return
		}
		reqURL := *r.URL
		parts := strings.SplitN(redirectTarget, "?", 2)
		if len(parts) == 2 {
			reqURL.RawQuery = parts[1]
		} else {
			reqURL.RawQuery = ""
		}
		proxyReq := r.Clone(r.Context())
		proxyReq.URL = &reqURL
		vm, err := h.buildTaskVM(proxyReq, principal)
		if err != nil {
			message, status := taskLoadError(err)
			http.Error(w, message, status)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := h.Templates.RenderTasksList(w, vm); err != nil {
			http.Error(w, "Template-Rendering fehlgeschlagen", http.StatusInternalServerError)
			return
		}
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
	if strings.Contains(strings.ToLower(err.Error()), "quick add") {
		return err.Error(), http.StatusBadRequest
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

func formFieldProvided(r *http.Request, key string) bool {
	_, ok := r.PostForm[key]
	return ok
}
