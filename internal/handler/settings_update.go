package handler

import (
	"net/http"
	"strconv"
	"strings"

	"caldo/internal/db"
)

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
		if err != nil || intervalMinutes < 1 {
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
		if uiLanguage == "" || darkMode == "" {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		if err := database.SaveUISettings(r.Context(), showCompleted, upcomingDays, uiLanguage, darkMode); err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/settings", http.StatusSeeOther)
	}
}
