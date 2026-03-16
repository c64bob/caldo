package handlers

import (
	"html/template"
	"net/http"

	"caldo/internal/http/middleware"
	"caldo/internal/service"
)

type SettingsHandler struct {
	Service          *service.SettingsService
	DefaultServerURL string
}

type settingsPageData struct {
	DefaultServerURL string
	Message          string
	Error            string
}

var settingsPageTpl = template.Must(template.New("settings").Parse(`<!doctype html>
<html lang="de"><head><meta charset="utf-8"><title>Caldo Settings</title></head>
<body>
<h1>CalDAV Verbindung</h1>
{{if .Message}}<p style="color: green">{{.Message}}</p>{{end}}
{{if .Error}}<p style="color: red">{{.Error}}</p>{{end}}
<form method="post" action="/settings/dav-account">
  <label>Server URL <input name="server_url" value="{{.DefaultServerURL}}" required></label><br>
  <label>Benutzername <input name="username" required></label><br>
  <label>Passwort <input type="password" name="password" required></label><br>
  <button type="submit">Speichern & Testen</button>
</form>
</body></html>`))

func (h *SettingsHandler) Page(w http.ResponseWriter, _ *http.Request) {
	_ = settingsPageTpl.Execute(w, settingsPageData{DefaultServerURL: h.DefaultServerURL})
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

	data := settingsPageData{DefaultServerURL: h.DefaultServerURL}
	if err != nil {
		data.Error = err.Error()
		_ = settingsPageTpl.Execute(w, data)
		return
	}

	data.Message = "CalDAV-Zugang erfolgreich gespeichert und validiert."
	_ = settingsPageTpl.Execute(w, data)
}
