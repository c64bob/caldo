package handler

import (
	"net/http"
	"strconv"
	"strings"

	"caldo/internal/caldav"
	"caldo/internal/db"
)

type settingsDependencies struct {
	database        *db.Database
	encryptionKey   []byte
	tester          CalDAVConnectionTester
	calendar        CalDAVCalendarClient
	proxyUserHeader string
}

type settingsPageState struct {
	CalDAVError    string
	CalDAVSuccess  string
	CalendarsError string
	Available      []caldav.Calendar
	SelectedHrefs  []string
	DefaultHref    string
	CalDAVURL      string
	CalDAVUsername string
	PreserveCalDAV bool
}

// SettingsCalDAVUpdate tests CalDAV settings and persists them after a successful save action.
func SettingsCalDAVUpdate(deps settingsDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.database == nil || deps.tester == nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		if err := r.ParseForm(); err != nil {
			renderSettingsPage(w, r, deps, settingsPageState{CalDAVError: "ungültige eingabe"}, http.StatusOK)
			return
		}

		credentials := db.CalDAVCredentials{
			URL:      strings.TrimSpace(r.FormValue("caldav_url")),
			Username: strings.TrimSpace(r.FormValue("caldav_username")),
			Password: r.FormValue("caldav_password"),
		}
		state := settingsPageState{
			CalDAVURL:      credentials.URL,
			CalDAVUsername: credentials.Username,
			PreserveCalDAV: true,
		}

		if credentials.Password == "" {
			current, err := deps.database.LoadCalDAVCredentials(r.Context(), deps.encryptionKey)
			if err != nil {
				state.CalDAVError = "passwort oder app-passwort ist erforderlich"
				renderSettingsPage(w, r, deps, state, http.StatusOK)
				return
			}
			credentials.Password = current.Password
		}

		capabilities, err := deps.tester.TestConnection(r.Context(), caldav.Credentials{
			URL:      credentials.URL,
			Username: credentials.Username,
			Password: credentials.Password,
		})
		if err != nil {
			state.CalDAVError = "verbindungstest fehlgeschlagen"
			renderSettingsPage(w, r, deps, state, http.StatusOK)
			return
		}

		if strings.EqualFold(strings.TrimSpace(r.FormValue("caldav_action")), "test") {
			state.CalDAVSuccess = "verbindungstest erfolgreich"
			renderSettingsPage(w, r, deps, state, http.StatusOK)
			return
		}

		if err := deps.database.SaveCalDAVCredentials(r.Context(), deps.encryptionKey, credentials); err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		if err := deps.database.SaveCalDAVServerCapabilities(r.Context(), db.CalDAVServerCapabilities{
			WebDAVSync:      capabilities.WebDAVSync,
			CTag:            capabilities.CTag,
			ETag:            capabilities.ETag,
			FullScan:        capabilities.FullScan,
			CalendarHomeSet: capabilities.CalendarHomeSet,
		}); err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/settings?caldav=saved", http.StatusSeeOther)
	}
}

// SettingsCalendarsUpdate persists selected calendars and the default project mapping.
func SettingsCalendarsUpdate(deps settingsDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.database == nil || deps.calendar == nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		if err := r.ParseForm(); err != nil {
			renderSettingsPage(w, r, deps, settingsPageState{CalendarsError: "ungültige eingabe"}, http.StatusOK)
			return
		}

		selectedHrefs := r.Form["calendar_href"]
		state := settingsPageState{
			CalendarsError: "",
			SelectedHrefs:  selectedHrefs,
			DefaultHref:    strings.TrimSpace(r.FormValue("default_calendar_href")),
		}
		defaultHref := state.DefaultHref

		calendars, err := loadSettingsCalendars(r.Context(), deps)
		if err != nil {
			state.CalendarsError = "kalender konnten nicht geladen werden"
			renderSettingsPage(w, r, deps, state, http.StatusOK)
			return
		}
		state.Available = calendars

		availableByHref := make(map[string]caldav.Calendar, len(calendars))
		for _, calendar := range calendars {
			availableByHref[calendar.Href] = calendar
		}

		selected := make([]db.SelectedCalendar, 0, len(selectedHrefs))
		seen := make(map[string]struct{}, len(selectedHrefs))
		for _, href := range selectedHrefs {
			href = strings.TrimSpace(href)
			calendar, ok := availableByHref[href]
			if !ok {
				continue
			}
			if _, exists := seen[href]; exists {
				continue
			}
			seen[href] = struct{}{}
			selected = append(selected, db.SelectedCalendar{
				Href:        calendar.Href,
				DisplayName: calendar.DisplayName,
			})
		}

		if len(selected) == 0 {
			state.CalendarsError = "mindestens ein kalender muss ausgewählt werden"
			renderSettingsPage(w, r, deps, state, http.StatusOK)
			return
		}
		if defaultHref == "" {
			state.CalendarsError = "ein default-projekt ist erforderlich"
			renderSettingsPage(w, r, deps, state, http.StatusOK)
			return
		}
		if _, ok := seen[defaultHref]; !ok {
			state.CalendarsError = "default-projekt muss ein ausgewählter kalender sein"
			renderSettingsPage(w, r, deps, state, http.StatusOK)
			return
		}

		capabilities, err := deps.database.LoadCalDAVServerCapabilities(r.Context())
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		if err := deps.database.SaveSettingsCalendars(r.Context(), selected, defaultHref, initialSyncStrategy(capabilities)); err != nil {
			state.CalendarsError = "kalenderauswahl konnte nicht gespeichert werden"
			renderSettingsPage(w, r, deps, state, http.StatusOK)
			return
		}

		http.Redirect(w, r, "/settings", http.StatusSeeOther)
	}
}

// SettingsSyncUpdate persists sync interval changes from settings page.
func SettingsSyncUpdate(database *db.Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if database == nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		if err := r.ParseForm(); err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		intervalMinutes, err := strconv.Atoi(strings.TrimSpace(r.FormValue("sync_interval_minutes")))
		if err != nil || intervalMinutes < 5 {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		if err := database.SaveSyncInterval(r.Context(), intervalMinutes); err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/settings", http.StatusSeeOther)
	}
}

// SettingsUIUpdate persists UI setting changes from settings page.
func SettingsUIUpdate(database *db.Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if database == nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		if err := r.ParseForm(); err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		upcomingDays, err := strconv.Atoi(strings.TrimSpace(r.FormValue("upcoming_days")))
		if err != nil || upcomingDays < 1 {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		showCompleted := strings.EqualFold(strings.TrimSpace(r.FormValue("show_completed")), "on")
		uiLanguage := strings.TrimSpace(r.FormValue("ui_language"))
		darkMode := strings.TrimSpace(r.FormValue("dark_mode"))
		taskNoteDisplay := strings.TrimSpace(r.FormValue("task_note_display"))
		if !isSupportedUILanguage(uiLanguage) || !isSupportedDarkMode(darkMode) || !isSupportedTaskNoteDisplay(taskNoteDisplay) {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		if err := database.SaveUISettings(r.Context(), showCompleted, upcomingDays, uiLanguage, darkMode, taskNoteDisplay); err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/settings", http.StatusSeeOther)
	}
}

func isSupportedUILanguage(language string) bool {
	return language == "de" || language == "en"
}

func isSupportedDarkMode(mode string) bool {
	return mode == "light" || mode == "dark" || mode == "system"
}

func isSupportedTaskNoteDisplay(mode string) bool {
	switch mode {
	case "none", "full", "first_line", "first_two_lines":
		return true
	default:
		return false
	}
}
