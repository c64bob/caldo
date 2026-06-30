package view

import (
	"context"
	"database/sql"
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
	SyncStatus       db.SyncStatus
	Available        []caldav.Calendar
	CalendarsLoaded  bool
	CalDAVError      string
	CalDAVSuccess    string
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
		text := Text(ctx)
		if _, err := fmt.Fprintf(w, `<section class="caldo-page max-w-3xl">
<h2 class="caldo-page-title">%s</h2>`, html.EscapeString(text.SettingsPageTitle)); err != nil {
			return err
		}

		if err := renderCalDAVSettings(w, csrfToken, model, text); err != nil {
			return err
		}
		if err := renderCalendarSettings(w, csrfToken, model, text); err != nil {
			return err
		}
		if err := renderSyncSettings(w, csrfToken, model.Settings, model.SyncStatus, text); err != nil {
			return err
		}
		if err := renderUISettings(w, csrfToken, model.Settings, text); err != nil {
			return err
		}
		if err := renderSecurityStatus(w, model, text); err != nil {
			return err
		}

		_, err := fmt.Fprint(w, `</section>`)
		return err
	})
}

func renderCalDAVSettings(w io.Writer, csrfToken string, model SettingsPageView, text Texts) error {
	if _, err := fmt.Fprintf(w, `<div class="caldo-card" data-settings-caldav>
<h3 class="font-medium">CalDAV</h3>
<p class="caldo-muted mt-1">%s</p>`, html.EscapeString(text.SettingsCalDAVHelp)); err != nil {
		return err
	}
	if model.CalDAVError != "" {
		if _, err := fmt.Fprintf(w, `<p class="caldo-alert caldo-alert-error mt-3" data-caldav-test-result="error">%s</p>`, html.EscapeString(model.CalDAVError)); err != nil {
			return err
		}
	}
	if model.CalDAVSuccess != "" {
		if _, err := fmt.Fprintf(w, `<p class="caldo-alert caldo-alert-success mt-3" data-caldav-test-result="success">%s</p>`, html.EscapeString(model.CalDAVSuccess)); err != nil {
			return err
		}
	}

	_, err := fmt.Fprintf(w, `<form class="mt-4 space-y-3" method="post" action="/settings/caldav" hx-post="/settings/caldav" hx-target="body" hx-swap="outerHTML" hx-push-url="false" hx-disabled-elt="find button" hx-headers='{"X-CSRF-Token":"%s"}'>
<div>
<label for="settings_caldav_url" class="caldo-label">%s</label>
<input id="settings_caldav_url" name="caldav_url" type="url" required class="caldo-input" value="%s"/>
</div>
<div>
<label for="settings_caldav_username" class="caldo-label">%s</label>
<input id="settings_caldav_username" name="caldav_username" type="text" required class="caldo-input" value="%s"/>
</div>
<div>
<label for="settings_caldav_password" class="caldo-label">%s</label>
<input id="settings_caldav_password" name="caldav_password" type="password" class="caldo-input" autocomplete="new-password" placeholder="%s"/>
<p class="caldo-meta mt-1">%s</p>
</div>
<div class="flex flex-wrap items-center gap-2">
<button type="submit" name="caldav_action" value="test" class="caldo-button caldo-button-secondary">%s</button>
<button type="submit" name="caldav_action" value="save" class="caldo-button caldo-button-primary">%s</button>
<span class="htmx-indicator caldo-meta ml-2" aria-live="polite">%s</span>
</div>
</form>
</div>`,
		csrfToken,
		html.EscapeString(text.SettingsCalDAVURL),
		html.EscapeString(model.Settings.CalDAVURL),
		html.EscapeString(text.SettingsCalDAVUsername),
		html.EscapeString(model.Settings.CalDAVUsername),
		html.EscapeString(text.SettingsCalDAVPassword),
		passwordPlaceholder(model.Settings.CalDAVConfigured, text),
		passwordHelp(model.Settings.CalDAVConfigured, text),
		html.EscapeString(text.SettingsCalDAVTest),
		html.EscapeString(text.SettingsCalDAVSubmit),
		html.EscapeString(text.SettingsCalDAVPending),
	)
	return err
}

func renderCalendarSettings(w io.Writer, csrfToken string, model SettingsPageView, text Texts) error {
	if _, err := fmt.Fprintf(w, `<div class="caldo-card" data-settings-calendars>
<h3 class="font-medium">%s</h3>
<p class="caldo-muted mt-1">%s</p>`, html.EscapeString(text.SettingsCalendarsTitle), html.EscapeString(text.SettingsCalendarsHelp)); err != nil {
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
		if _, err := fmt.Fprintf(w, `<p class="caldo-alert caldo-alert-warning">%s</p>`, html.EscapeString(text.SettingsNoCalendars)); err != nil {
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
			state := "remote-only"
			stateLabel := text.SettingsCalRemoteOnly
			stateBadgeClass := "caldo-badge"
			if project.ID != "" {
				state = "synced"
				stateLabel = text.SettingsCalSynced
				stateBadgeClass = "caldo-badge caldo-badge-accent"
			}
			if _, err := fmt.Fprintf(w, `<div class="caldo-list-row" data-settings-calendar-state="%s">
<label class="flex items-center gap-2">
<input class="caldo-check" type="checkbox" name="calendar_href" value="%s"%s/>
<span class="font-medium">%s</span>
</label>
<div class="mt-2 flex flex-wrap items-center gap-2">
<span class="%s">%s</span>`,
				state,
				html.EscapeString(calendar.Href),
				checked,
				html.EscapeString(calendar.DisplayName),
				stateBadgeClass,
				html.EscapeString(stateLabel),
			); err != nil {
				return err
			}
			if defaultChecked != "" {
				if _, err := fmt.Fprintf(w, `<span class="caldo-badge caldo-badge-accent">%s</span>`, html.EscapeString(text.SettingsCalDefault)); err != nil {
					return err
				}
			}
			if _, err := fmt.Fprintf(w, `</div>
<p class="caldo-meta mt-1">%s</p>`, html.EscapeString(settingsCalendarMeta(project, text))); err != nil {
				return err
			}
			if project.ID != "" {
				if _, err := fmt.Fprintf(w, `<p class="caldo-meta mt-1" data-calendar-remove-impact>%s</p>`, html.EscapeString(settingsCalendarRemoveImpact(project, text))); err != nil {
					return err
				}
			}
			if _, err := fmt.Fprintf(w, `
<label class="caldo-muted mt-2 flex items-center gap-2">
<input class="caldo-check" type="radio" name="default_calendar_href" value="%s"%s/>
<span>%s</span>
</label>
</div>`,
				html.EscapeString(calendar.Href),
				defaultChecked,
				html.EscapeString(text.SettingsUseAsDefault),
			); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprint(w, `</div>`); err != nil {
			return err
		}
	}

	if err := renderLocalOnlyProjects(w, model, text); err != nil {
		return err
	}

	_, err := fmt.Fprintf(w, `<button type="submit" class="caldo-button caldo-button-secondary">%s</button>
<span class="htmx-indicator caldo-meta ml-2" aria-live="polite">%s</span>
</form>
</div>`, html.EscapeString(text.SettingsSaveCalendars), html.EscapeString(text.SettingsCalendarsPending))
	return err
}

func renderSyncSettings(w io.Writer, csrfToken string, settings db.AppSettings, syncStatus db.SyncStatus, text Texts) error {
	if _, err := fmt.Fprintf(w, `<div class="caldo-card">
<h3 class="font-medium">%s</h3>`, html.EscapeString(text.SettingsSyncTitle)); err != nil {
		return err
	}
	if err := renderSettingsSyncSummary(w, syncStatus, text); err != nil {
		return err
	}
	_, err := fmt.Fprintf(w, `
	<form class="mt-3 space-y-2" method="post" action="/settings/sync" hx-post="/settings/sync" hx-target="body" hx-swap="outerHTML" hx-push-url="false" hx-disabled-elt="find button" hx-headers='{"X-CSRF-Token":"%s"}'>
<label class="caldo-label">%s
<input class="caldo-input w-32" type="number" min="5" name="sync_interval_minutes" value="%d">
</label>
<button type="submit" class="caldo-button caldo-button-secondary">%s</button>
<span class="htmx-indicator caldo-meta ml-2" aria-live="polite">%s</span>
</form>
		<form class="mt-3" method="post" action="/sync/manual" data-sync-request hx-post="/sync/manual" hx-target="#sync-status" hx-swap="innerHTML" hx-disabled-elt="find button" hx-headers='{"X-CSRF-Token":"%s"}'>
<button type="submit" class="caldo-button caldo-button-primary">%s</button>
<span class="htmx-indicator caldo-meta ml-2" aria-live="polite">%s</span>
</form>
</div>`, csrfToken, html.EscapeString(text.SettingsIntervalMinutes), settings.SyncIntervalMinutes, html.EscapeString(text.SettingsSaveSync), html.EscapeString(text.SettingsSyncPending), csrfToken, html.EscapeString(text.SettingsManualSync), html.EscapeString(text.SettingsManualPending))
	return err
}

func renderSettingsSyncSummary(w io.Writer, syncStatus db.SyncStatus, text Texts) error {
	if _, err := fmt.Fprintf(w, `<div class="caldo-settings-sync-summary" data-settings-sync-state="%s">
<div><span class="caldo-settings-sync-label">%s</span><strong>%s</strong></div>
<div><span class="caldo-settings-sync-label">%s</span><span>%s</span></div>
<div><span class="caldo-settings-sync-label">%s</span><span>%s</span></div>`,
		html.EscapeString(settingsSyncStateValue(syncStatus.State)),
		html.EscapeString(text.SettingsSyncStatus),
		html.EscapeString(settingsSyncStateLabel(syncStatus.State, text)),
		html.EscapeString(text.SettingsSyncLastOK),
		html.EscapeString(settingsSyncTimeLabel(syncStatus.LastSuccessAt, text)),
		html.EscapeString(text.SettingsSyncLastDone),
		html.EscapeString(settingsSyncTimeLabel(syncStatus.LastFinished, text)),
	); err != nil {
		return err
	}
	if syncStatus.LastErrorCode.Valid && strings.TrimSpace(syncStatus.LastErrorCode.String) != "" {
		if _, err := fmt.Fprintf(w, `<div data-settings-sync-error><span class="caldo-settings-sync-label">%s</span><span>%s</span></div>`,
			html.EscapeString(text.SettingsSyncLastErr),
			html.EscapeString(settingsSyncErrorLabel(syncStatus.LastErrorCode.String, text)),
		); err != nil {
			return err
		}
	}
	_, err := fmt.Fprint(w, `</div>`)
	return err
}

func renderUISettings(w io.Writer, csrfToken string, settings db.AppSettings, text Texts) error {
	_, err := fmt.Fprintf(w, `<div id="ui-settings" class="caldo-card">
<h3 class="font-medium">%s</h3>
	<form class="mt-3 space-y-3 text-sm" method="post" action="/settings/ui" hx-post="/settings/ui" hx-target="body" hx-swap="outerHTML" hx-push-url="false" hx-disabled-elt="find button" hx-headers='{"X-CSRF-Token":"%s"}'>
<label class="flex items-center gap-2"><input class="caldo-check" type="checkbox" name="show_completed" %s> %s</label>
<label class="caldo-label">%s
<input class="caldo-input w-32" type="number" min="1" name="upcoming_days" value="%d">
</label>
<label class="caldo-label">%s
<select class="caldo-select w-48" name="ui_language">
<option value="de" %s>Deutsch</option>
<option value="en" %s>English</option>
</select>
</label>
<label class="caldo-label">%s
<select class="caldo-select w-48" name="dark_mode">
<option value="system" %s>%s</option>
<option value="light" %s>%s</option>
<option value="dark" %s>%s</option>
</select>
</label>
<button type="submit" class="caldo-button caldo-button-secondary">%s</button>
<span class="htmx-indicator caldo-meta ml-2" aria-live="polite">%s</span>
</form>
</div>`, html.EscapeString(text.SettingsUITitle), csrfToken, checkedAttr(settings.ShowCompleted), html.EscapeString(text.SettingsShowCompleted), html.EscapeString(text.SettingsUpcomingDays), settings.UpcomingDays, html.EscapeString(text.SettingsLanguage), selectedAttr(settings.UILanguage, "de"), selectedAttr(settings.UILanguage, "en"), html.EscapeString(text.SettingsDarkMode), selectedAttr(settings.DarkMode, "system"), html.EscapeString(text.ThemeSystem), selectedAttr(settings.DarkMode, "light"), html.EscapeString(text.ThemeLight), selectedAttr(settings.DarkMode, "dark"), html.EscapeString(text.ThemeDark), html.EscapeString(text.SettingsSaveUI), html.EscapeString(text.SettingsSyncPending))
	return err
}

func renderSecurityStatus(w io.Writer, model SettingsPageView, text Texts) error {
	proxyStatus := text.SettingsNotDetected
	if model.ProxyUserPresent {
		proxyStatus = text.SettingsDetected
	}
	httpsStatus := text.SettingsActive
	if !model.HTTPSConfigured {
		httpsStatus = text.SettingsInconsistent
	}

	_, err := fmt.Fprintf(w, `<div class="caldo-card text-sm">
<h3 class="font-medium">%s</h3>
<p class="mt-2">%s: <code>%s</code> · %s</p>
<p>%s: %s</p>
</div>`, html.EscapeString(text.SettingsSecurityTitle), html.EscapeString(text.SettingsProxyHeader), html.EscapeString(model.ProxyUserHeader), html.EscapeString(proxyStatus), html.EscapeString(text.SettingsHTTPSStatus), html.EscapeString(httpsStatus))
	return err
}

func renderLocalOnlyProjects(w io.Writer, model SettingsPageView, text Texts) error {
	if !settingsCalendarsLoaded(model) {
		return nil
	}
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

	if _, err := fmt.Fprintf(w, `<div class="caldo-alert caldo-alert-warning" data-settings-calendar-state="remote-missing"><p class="font-medium">%s</p><p class="caldo-meta mt-1">%s</p><ul class="mt-2 space-y-2">`,
		html.EscapeString(text.SettingsLocalOnlyTitle),
		html.EscapeString(text.SettingsCalMissingHelp),
	); err != nil {
		return err
	}
	for _, project := range localOnly {
		if _, err := fmt.Fprintf(w, `<li data-settings-calendar-state="remote-missing"><div class="flex flex-wrap items-center gap-2"><span class="font-medium">%s</span><span class="caldo-badge caldo-badge-warning">%s</span>`,
			html.EscapeString(project.DisplayName),
			html.EscapeString(text.SettingsCalRemoteMissing),
		); err != nil {
			return err
		}
		if project.IsDefault {
			if _, err := fmt.Fprintf(w, `<span class="caldo-badge caldo-badge-accent">%s</span>`, html.EscapeString(text.SettingsCalDefault)); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintf(w, `</div><p class="caldo-meta mt-1">%d %s · %s</p></li>`,
			project.TaskCount,
			html.EscapeString(text.SettingsTasks),
			html.EscapeString(text.SettingsCalMissingImpact),
		); err != nil {
			return err
		}
	}
	_, err := fmt.Fprint(w, `</ul></div>`)
	return err
}

func settingsCalendarsLoaded(model SettingsPageView) bool {
	return model.CalendarsLoaded || model.Available != nil
}

func settingsCalendarMeta(project db.SettingsProject, text Texts) string {
	if project.ID == "" {
		return text.SettingsNotMapped
	}
	return fmt.Sprintf("%s: %s · %d %s · %d %s · %s: %s",
		text.SettingsProjectPrefix,
		project.DisplayName,
		project.OpenTaskCount,
		text.SettingsOpenTasks,
		project.TaskCount,
		text.SettingsTasks,
		text.SettingsCalSyncStrategy,
		project.SyncStrategy,
	)
}

func settingsCalendarRemoveImpact(project db.SettingsProject, text Texts) string {
	if project.TaskCount == 0 {
		return text.SettingsCalRemoveEmpty
	}
	return text.SettingsCalRemoveTasks
}

func passwordPlaceholder(configured bool, text Texts) string {
	if configured {
		return html.EscapeString(text.SettingsPasswordKeep)
	}
	return html.EscapeString(text.SettingsPasswordNew)
}

func passwordHelp(configured bool, text Texts) string {
	if configured {
		return html.EscapeString(text.SettingsPasswordKeepHelp)
	}
	return html.EscapeString(text.SettingsPasswordNewHelp)
}

func settingsSyncStateValue(state string) string {
	switch strings.TrimSpace(state) {
	case "running":
		return "running"
	case "error":
		return "error"
	default:
		return "idle"
	}
}

func settingsSyncStateLabel(state string, text Texts) string {
	switch settingsSyncStateValue(state) {
	case "running":
		return text.SettingsSyncRunning
	case "error":
		return text.SettingsSyncError
	default:
		return text.SettingsSyncIdle
	}
}

func settingsSyncTimeLabel(ts sql.NullTime, text Texts) string {
	if !ts.Valid {
		return text.SettingsSyncNever
	}
	return ts.Time.Local().Format("02.01.2006 15:04")
}

func settingsSyncErrorLabel(errorCode string, text Texts) string {
	switch strings.TrimSpace(errorCode) {
	case "sync_failed":
		return text.SettingsSyncErrFailed
	case "sync_unavailable":
		return text.SettingsSyncErrOffline
	default:
		return text.SettingsSyncErrUnknown
	}
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
