package handler

import (
	"context"
	"net/http"
	"strings"

	"caldo/internal/caldav"
	"caldo/internal/db"
	"caldo/internal/view"
)

// CalDAVConnectionTester defines CalDAV connection tests for setup.
type CalDAVConnectionTester interface {
	TestConnection(ctx context.Context, credentials caldav.Credentials) (caldav.ServerCapabilities, error)
}

type setupDependencies struct {
	database      *db.Database
	encryptionKey []byte
	tester        CalDAVConnectionTester
}

// SetupPage renders the setup step for CalDAV credential capture.
func SetupPage(w http.ResponseWriter, r *http.Request) {
	renderSetupCalDAVPage(w, r, "")
}

// SetupCalDAV stores encrypted credentials, executes a live CalDAV connection test, and advances the setup step on success.
func SetupCalDAV(deps setupDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.database == nil || deps.tester == nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		if err := r.ParseForm(); err != nil {
			renderSetupCalDAVPage(w, r, "ungültige eingabe")
			return
		}

		credentials := db.CalDAVCredentials{
			URL:      strings.TrimSpace(r.FormValue("caldav_url")),
			Username: strings.TrimSpace(r.FormValue("caldav_username")),
			Password: r.FormValue("caldav_password"),
		}

		if err := deps.database.SaveCalDAVCredentials(r.Context(), deps.encryptionKey, credentials); err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		capabilities, err := deps.tester.TestConnection(r.Context(), caldav.Credentials{
			URL:      credentials.URL,
			Username: credentials.Username,
			Password: credentials.Password,
		})
		if err != nil {
			renderSetupCalDAVPage(w, r, "verbindungstest fehlgeschlagen")
			return
		}

		if err := deps.database.SaveCalDAVServerCapabilities(r.Context(), db.CalDAVServerCapabilities{
			WebDAVSync: capabilities.WebDAVSync,
			CTag:       capabilities.CTag,
			ETag:       capabilities.ETag,
			FullScan:   capabilities.FullScan,
		}); err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		if err := deps.database.SaveSetupStep(r.Context(), "calendars"); err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/setup/calendars", http.StatusFound)
	}
}

func renderSetupCalDAVPage(w http.ResponseWriter, r *http.Request, errorMessage string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := view.BaseLayout("Caldo Setup", view.SetupCalDAVContent(errorMessage)).Render(r.Context(), w); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func SetupCalendarsPage(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

func SetupCalendars(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

func SetupImport(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

func SetupImportEvents(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

func SetupComplete(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}
