package handlers

import (
	"html/template"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"caldo/internal/http/middleware"
	"caldo/internal/http/render"
	"caldo/internal/service"
)

type SettingsHandler struct {
	Service            *service.SettingsService
	PreferencesService *service.PreferencesService
	TaskService        *service.TaskService
	DefaultServerURL   string
}

type settingsPageData struct {
	DefaultServerURL string
	PrincipalID      string
	DAVUsername      string
	Message          string
	Error            string
	DefaultView      string
	SyncInterval     int
	VisibleColumns   map[string]bool
	AllColumns       []string
}

func loadSettingsTemplate() *template.Template {
	templateRoot, err := render.ResolveTemplateRoot()
	if err == nil {
		settingsPath := filepath.Join(templateRoot, "pages", "settings.gohtml")
		if tpl, parseErr := template.ParseFiles(settingsPath); parseErr == nil {
			return tpl
		}
	}
	return template.Must(template.New("settings_page").Parse(`<html><body><h1>Settings template missing</h1></body></html>`))
}

var settingsPageTpl = loadSettingsTemplate()

func (h *SettingsHandler) Page(w http.ResponseWriter, r *http.Request) {
	principal, _ := middleware.PrincipalFromContext(r.Context())
	data := h.loadPageData(r, principal)
	_ = settingsPageTpl.ExecuteTemplate(w, "settings_page", data)
}

func (h *SettingsHandler) SaveDAVAccount(w http.ResponseWriter, r *http.Request) {
	principal, ok := middleware.PrincipalFromContext(r.Context())
	if !ok {
		http.Error(w, "Authentifizierungsheader fehlt", http.StatusUnauthorized)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Ungültiges Formular", http.StatusBadRequest)
		return
	}

	err := h.Service.SaveDAVAccount(r.Context(), service.SaveDAVAccountInput{
		PrincipalID: principal,
		ServerURL:   r.FormValue("server_url"),
		Username:    r.FormValue("username"),
		Password:    r.FormValue("password"),
	})

	data := h.loadPageData(r, principal)
	if err != nil {
		data.Error = err.Error()
		_ = settingsPageTpl.ExecuteTemplate(w, "settings_page", data)
		return
	}

	data.Message = "CalDAV-Zugang erfolgreich gespeichert und validiert."
	_ = settingsPageTpl.ExecuteTemplate(w, "settings_page", data)
}

func (h *SettingsHandler) SavePreferences(w http.ResponseWriter, r *http.Request) {
	principal, ok := middleware.PrincipalFromContext(r.Context())
	if !ok {
		http.Error(w, "Authentifizierungsheader fehlt", http.StatusUnauthorized)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Ungültiges Formular", http.StatusBadRequest)
		return
	}
	interval, _ := strconv.Atoi(strings.TrimSpace(r.FormValue("sync_interval_seconds")))
	if err := h.PreferencesService.Save(r.Context(), service.PreferencesInput{
		PrincipalID:         principal,
		DefaultView:         strings.TrimSpace(r.FormValue("default_view")),
		SyncIntervalSeconds: interval,
		VisibleColumns:      r.Form["columns"],
	}); err != nil {
		http.Error(w, "Einstellungen konnten nicht gespeichert werden", http.StatusBadGateway)
		return
	}
	data := h.loadPageData(r, principal)
	data.Message = "Einstellungen gespeichert."
	_ = settingsPageTpl.ExecuteTemplate(w, "settings_page", data)
}

func (h *SettingsHandler) CreateCollection(w http.ResponseWriter, r *http.Request) {
	principal, ok := middleware.PrincipalFromContext(r.Context())
	if !ok {
		http.Error(w, "Authentifizierungsheader fehlt", http.StatusUnauthorized)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Ungültiges Formular", http.StatusBadRequest)
		return
	}
	name := strings.TrimSpace(r.FormValue("collection_name"))
	if err := h.TaskService.CreateCollection(r.Context(), principal, name, name); err != nil {
		data := h.loadPageData(r, principal)
		data.Error = err.Error()
		_ = settingsPageTpl.ExecuteTemplate(w, "settings_page", data)
		return
	}
	data := h.loadPageData(r, principal)
	data.Message = "Neue Liste wurde angelegt."
	_ = settingsPageTpl.ExecuteTemplate(w, "settings_page", data)
}

func (h *SettingsHandler) loadPageData(r *http.Request, principal string) settingsPageData {
	prefs := struct {
		DefaultView         string
		SyncIntervalSeconds int
		VisibleColumns      []string
	}{DefaultView: "main", SyncIntervalSeconds: 300, VisibleColumns: []string{"star", "check", "name", "folder", "context", "due", "priority"}}
	if h.PreferencesService != nil {
		if loaded, err := h.PreferencesService.GetOrDefault(r.Context(), principal); err == nil {
			prefs.DefaultView = loaded.DefaultView
			prefs.SyncIntervalSeconds = loaded.SyncIntervalSeconds
			prefs.VisibleColumns = loaded.VisibleColumns
		}
	}
	columns := map[string]bool{}
	for _, col := range prefs.VisibleColumns {
		columns[col] = true
	}
	data := settingsPageData{
		DefaultServerURL: h.DefaultServerURL,
		PrincipalID:      principal,
		DefaultView:      prefs.DefaultView,
		SyncInterval:     prefs.SyncIntervalSeconds,
		VisibleColumns:   columns,
		AllColumns:       []string{"star", "check", "name", "folder", "context", "due", "priority"},
	}
	if h.Service != nil {
		if account, ok, _ := h.Service.GetDAVAccount(r.Context(), principal); ok {
			data.DAVUsername = account.Username
			if data.DefaultServerURL == "" {
				data.DefaultServerURL = account.ServerURL
			}
		}
	}
	return data
}
