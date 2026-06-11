package view

import (
	"context"
	"fmt"
	"html"
	"io"
	"strings"

	"caldo/internal/caldav"
	"caldo/internal/db"
	"github.com/a-h/templ"
)

// SettingsPageView contains all normal-operation settings page state.
type SettingsPageView struct {
	Settings         db.AppSettings
	Available        []caldav.Calendar
	CalDAVError      string
	CalendarsError   string
	SelectedHrefs    []string
	DefaultHref      string
	ProxyUserHeader  string
	ProxyUserPresent bool
	HTTPSConfigured  bool
}

// SettingsPageContent renders normal-operation settings.
func SettingsPageContent(model SettingsPageView) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		csrfToken := html.EscapeString(CSRFToken(ctx))
		if _, err := fmt.Fprint(w, `<section class="caldo-page max-w-3xl">
<h2 class="caldo-page-title">Einstellungen</h2>`); err != nil {
			return err
		}

		if err := renderCalDAVSettings(w, csrfToken, model); err != nil {
			return err
		}
		if err := renderCalendarSettings(w, csrfToken, model); err != nil {
			return err
		}
		if err := renderSyncSettings(w, csrfToken, model.Settings); err != nil {
			return err
		}
		if err := renderUISettings(w, csrfToken, model.Settings); err != nil {
			return err
		}
		if err := renderSecurityStatus(w, model); err != nil {
			return err
		}

		_, err := fmt.Fprint(w, `</section>`)
		return err
	})
}

func renderCalDAVSettings(w io.Writer, csrfToken string, model SettingsPageView) error {
	if _, err := fmt.Fprint(w, `<div class="caldo-card">
<h3 class="font-medium">CalDAV</h3>
<p class="caldo-muted mt-1">Verbindung speichern nur nach erfolgreichem Test.</p>`); err != nil {
		return err
	}
	if model.CalDAVError != "" {
		if _, err := fmt.Fprintf(w, `<p class="caldo-alert caldo-alert-error mt-3">%s</p>`, html.EscapeString(model.CalDAVError)); err != nil {
			return err
		}
	}

	_, err := fmt.Fprintf(w, `<form class="mt-4 space-y-3" method="post" action="/settings/caldav" hx-post="/settings/caldav" hx-target="body" hx-swap="outerHTML" hx-push-url="false" hx-disabled-elt="find button" hx-headers='{"X-CSRF-Token":"%s"}'>
<div>
<label for="settings_caldav_url" class="caldo-label">CalDAV-URL</label>
<input id="settings_caldav_url" name="caldav_url" type="url" required class="caldo-input" value="%s"/>
</div>
<div>
<label for="settings_caldav_username" class="caldo-label">Benutzername</label>
<input id="settings_caldav_username" name="caldav_username" type="text" required class="caldo-input" value="%s"/>
</div>
<div>
<label for="settings_caldav_password" class="caldo-label">Passwort / App-Passwort</label>
<input id="settings_caldav_password" name="caldav_password" type="password" class="caldo-input" autocomplete="new-password" placeholder="%s"/>
<p class="caldo-meta mt-1">%s</p>
</div>
<button type="submit" class="caldo-button caldo-button-primary">CalDAV speichern und testen</button>
<span class="htmx-indicator caldo-meta ml-2" aria-live="polite">Verbindung wird getestet ...</span>
</form>
</div>`,
		csrfToken,
		html.EscapeString(model.Settings.CalDAVURL),
		html.EscapeString(model.Settings.CalDAVUsername),
		passwordPlaceholder(model.Settings.CalDAVConfigured),
		passwordHelp(model.Settings.CalDAVConfigured),
	)
	return err
}

func renderCalendarSettings(w io.Writer, csrfToken string, model SettingsPageView) error {
	if _, err := fmt.Fprint(w, `<div class="caldo-card">
<h3 class="font-medium">Kalender & Projektmapping</h3>
<p class="caldo-muted mt-1">Ausgewählte CalDAV-Kalender werden als Projekte geführt. Bestehende Projekte mit Aufgaben bleiben erhalten.</p>`); err != nil {
		return err
	}
	if model.CalendarsError != "" {
		if _, err := fmt.Fprintf(w, `<p class="caldo-alert caldo-alert-error mt-3">%s</p>`, html.EscapeString(model.CalendarsError)); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintf(w, `<form class="mt-4 space-y-4" method="post" action="/settings/calendars" hx-post="/settings/calendars" hx-target="body" hx-swap="outerHTML" hx-push-url="false" hx-disabled-elt="find button" hx-headers='{"X-CSRF-Token":"%s"}'>`, csrfToken); err != nil {
		return err
	}

	if len(model.Available) == 0 {
		if _, err := fmt.Fprint(w, `<p class="caldo-alert caldo-alert-warning">Keine CalDAV-Kalender geladen. Prüfe die CalDAV-Verbindung.</p>`); err != nil {
			return err
		}
	} else {
		if _, err := fmt.Fprint(w, `<div class="space-y-3">`); err != nil {
			return err
		}
		selected := selectedCalendarSet(model)
		defaultHref := defaultCalendarHref(model)
		projectByHref := settingsProjectByHref(model.Settings.Projects)
		for _, calendar := range model.Available {
			project := projectByHref[calendar.Href]
			checked := ""
			if selected[calendar.Href] {
				checked = " checked"
			}
			defaultChecked := ""
			if calendar.Href == defaultHref {
				defaultChecked = " checked"
			}
			mapping := "Noch nicht als Projekt hinzugefügt"
			if project.ID != "" {
				mapping = fmt.Sprintf("Projekt: %s · %d offene Aufgaben", project.DisplayName, project.OpenTaskCount)
			}
			if _, err := fmt.Fprintf(w, `<div class="caldo-list-row">
<label class="flex items-center gap-2">
<input class="caldo-check" type="checkbox" name="calendar_href" value="%s"%s/>
<span class="font-medium">%s</span>
</label>
<p class="caldo-meta mt-1">%s</p>
<label class="caldo-muted mt-2 flex items-center gap-2">
<input class="caldo-check" type="radio" name="default_calendar_href" value="%s"%s/>
<span>Als Default-Projekt verwenden</span>
</label>
</div>`,
				html.EscapeString(calendar.Href),
				checked,
				html.EscapeString(calendar.DisplayName),
				html.EscapeString(mapping),
				html.EscapeString(calendar.Href),
				defaultChecked,
			); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprint(w, `</div>`); err != nil {
			return err
		}
	}

	if err := renderLocalOnlyProjects(w, model); err != nil {
		return err
	}

	_, err := fmt.Fprint(w, `<button type="submit" class="caldo-button caldo-button-secondary">Kalenderauswahl speichern</button>
<span class="htmx-indicator caldo-meta ml-2" aria-live="polite">Speichern ...</span>
</form>
</div>`)
	return err
}

func renderSyncSettings(w io.Writer, csrfToken string, settings db.AppSettings) error {
	_, err := fmt.Fprintf(w, `<div class="caldo-card">
<h3 class="font-medium">Sync</h3>
<form class="mt-3 space-y-2" method="post" action="/settings/sync" hx-post="/settings/sync" hx-disabled-elt="find button" hx-headers='{"X-CSRF-Token":"%s"}'>
<label class="caldo-label">Intervall (Minuten)
<input class="caldo-input w-32" type="number" min="5" name="sync_interval_minutes" value="%d">
</label>
<button type="submit" class="caldo-button caldo-button-secondary">Sync-Einstellungen speichern</button>
<span class="htmx-indicator caldo-meta ml-2" aria-live="polite">Speichern ...</span>
</form>
<form class="mt-3" method="post" action="/sync/manual" hx-post="/sync/manual" hx-disabled-elt="find button" hx-headers='{"X-CSRF-Token":"%s"}'>
<button type="submit" class="caldo-button caldo-button-primary">Jetzt synchronisieren</button>
<span class="htmx-indicator caldo-meta ml-2" aria-live="polite">Synchronisieren ...</span>
</form>
</div>`, csrfToken, settings.SyncIntervalMinutes, csrfToken)
	return err
}

func renderUISettings(w io.Writer, csrfToken string, settings db.AppSettings) error {
	_, err := fmt.Fprintf(w, `<div class="caldo-card">
<h3 class="font-medium">UI</h3>
<form class="mt-3 space-y-3 text-sm" method="post" action="/settings/ui" hx-post="/settings/ui" hx-disabled-elt="find button" hx-headers='{"X-CSRF-Token":"%s"}'>
<label class="flex items-center gap-2"><input class="caldo-check" type="checkbox" name="show_completed" %s> Erledigte Aufgaben anzeigen</label>
<label class="caldo-label">Demnächst-Zeitraum (Tage)
<input class="caldo-input w-32" type="number" min="1" name="upcoming_days" value="%d">
</label>
<label class="caldo-label">Sprache
<select class="caldo-select w-48" name="ui_language">
<option value="de" %s>Deutsch</option>
<option value="en" %s>English</option>
</select>
</label>
<label class="caldo-label">Dark Mode
<select class="caldo-select w-48" name="dark_mode">
<option value="system" %s>System</option>
<option value="light" %s>Hell</option>
<option value="dark" %s>Dunkel</option>
</select>
</label>
<button type="submit" class="caldo-button caldo-button-secondary">UI-Einstellungen speichern</button>
<span class="htmx-indicator caldo-meta ml-2" aria-live="polite">Speichern ...</span>
</form>
</div>`, csrfToken, checkedAttr(settings.ShowCompleted), settings.UpcomingDays, selectedAttr(settings.UILanguage, "de"), selectedAttr(settings.UILanguage, "en"), selectedAttr(settings.DarkMode, "system"), selectedAttr(settings.DarkMode, "light"), selectedAttr(settings.DarkMode, "dark"))
	return err
}

func renderSecurityStatus(w io.Writer, model SettingsPageView) error {
	proxyStatus := "nicht erkannt"
	if model.ProxyUserPresent {
		proxyStatus = "erkannt"
	}
	httpsStatus := "aktiv"
	if !model.HTTPSConfigured {
		httpsStatus = "inkonsistent"
	}

	_, err := fmt.Fprintf(w, `<div class="caldo-card text-sm">
<h3 class="font-medium">Sicherheitsstatus</h3>
<p class="mt-2">Reverse-Proxy-Header: <code>%s</code> · %s</p>
<p>HTTPS-Status: %s</p>
</div>`, html.EscapeString(model.ProxyUserHeader), proxyStatus, httpsStatus)
	return err
}

func renderLocalOnlyProjects(w io.Writer, model SettingsPageView) error {
	available := make(map[string]struct{}, len(model.Available))
	for _, calendar := range model.Available {
		available[calendar.Href] = struct{}{}
	}

	localOnly := make([]db.SettingsProject, 0)
	for _, project := range model.Settings.Projects {
		if _, ok := available[project.CalendarHref]; !ok {
			localOnly = append(localOnly, project)
		}
	}
	if len(localOnly) == 0 {
		return nil
	}

	if _, err := fmt.Fprint(w, `<div class="caldo-alert caldo-alert-warning"><p class="font-medium">Lokale Projekte ohne aktuell geladenen CalDAV-Kalender</p><ul class="mt-2 list-disc pl-5">`); err != nil {
		return err
	}
	for _, project := range localOnly {
		if _, err := fmt.Fprintf(w, `<li>%s · %d Aufgaben</li>`, html.EscapeString(project.DisplayName), project.TaskCount); err != nil {
			return err
		}
	}
	_, err := fmt.Fprint(w, `</ul></div>`)
	return err
}

func passwordPlaceholder(configured bool) string {
	if configured {
		return "unverändert lassen"
	}
	return "Passwort oder App-Passwort"
}

func passwordHelp(configured bool) string {
	if configured {
		return "Leer lassen, um das gespeicherte Passwort beizubehalten."
	}
	return "Ein Passwort oder App-Passwort ist für den Verbindungstest erforderlich."
}

func selectedCalendarSet(model SettingsPageView) map[string]bool {
	selected := make(map[string]bool)
	if len(model.SelectedHrefs) > 0 {
		for _, href := range model.SelectedHrefs {
			selected[strings.TrimSpace(href)] = true
		}
		return selected
	}
	for _, project := range model.Settings.Projects {
		selected[project.CalendarHref] = true
	}
	return selected
}

func defaultCalendarHref(model SettingsPageView) string {
	if strings.TrimSpace(model.DefaultHref) != "" {
		return strings.TrimSpace(model.DefaultHref)
	}
	for _, project := range model.Settings.Projects {
		if project.IsDefault {
			return project.CalendarHref
		}
	}
	if len(model.Available) > 0 {
		return model.Available[0].Href
	}
	return ""
}

func settingsProjectByHref(projects []db.SettingsProject) map[string]db.SettingsProject {
	byHref := make(map[string]db.SettingsProject, len(projects))
	for _, project := range projects {
		byHref[project.CalendarHref] = project
	}
	return byHref
}

func checkedAttr(v bool) string {
	if v {
		return "checked"
	}
	return ""
}

func selectedAttr(current string, expected string) string {
	if current == expected {
		return "selected"
	}
	return ""
}
