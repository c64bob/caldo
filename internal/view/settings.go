package view

import (
	"context"
	"fmt"
	"html"
	"io"

	"caldo/internal/db"
	"github.com/a-h/templ"
)

// SettingsPageContent renders normal-operation settings.
func SettingsPageContent(settings db.AppSettings, proxyUserHeader string, httpsConfigured bool) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		csrfToken := html.EscapeString(CSRFToken(ctx))
		httpsStatus := "aktiv"
		if !httpsConfigured {
			httpsStatus = "inkonsistent"
		}
		_, err := fmt.Fprintf(w, `<section class="caldo-page max-w-3xl">
<h2 class="caldo-page-title">Einstellungen</h2>
<div class="caldo-card">
<h3 class="font-medium">CalDAV & Projekte</h3>
<p class="caldo-muted mt-1">CalDAV-Zugang, Kalenderauswahl, Projekt-Mapping und Default-Projekt werden über die Einstellungen-Routen aktualisiert.</p>
</div>
<div class="caldo-card">
<h3 class="font-medium">Sync</h3>
<form class="mt-3 space-y-2" method="post" action="/settings/sync" hx-post="/settings/sync" hx-disabled-elt="find button" hx-headers='{"X-CSRF-Token":"%s"}'>
<label class="caldo-label">Intervall (Minuten)
<input class="caldo-input w-32" type="number" min="1" name="sync_interval_minutes" value="%d">
</label>
<button type="submit" class="caldo-button caldo-button-secondary">Sync-Einstellungen speichern</button>
<span class="htmx-indicator caldo-meta ml-2" aria-live="polite">Speichern ...</span>
</form>
<form class="mt-3" method="post" action="/sync/manual" hx-post="/sync/manual" hx-disabled-elt="find button" hx-headers='{"X-CSRF-Token":"%s"}'>
<button type="submit" class="caldo-button caldo-button-primary">Jetzt synchronisieren</button>
<span class="htmx-indicator caldo-meta ml-2" aria-live="polite">Synchronisieren ...</span>
</form>
</div>
<div class="caldo-card">
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
</div>
<div class="caldo-card text-sm">
<h3 class="font-medium">Sicherheitsstatus</h3>
<p class="mt-2">Reverse-Proxy-Header: <code>%s</code></p>
<p>HTTPS-Status: %s</p>
</div>
</section>`, csrfToken, settings.SyncIntervalMinutes, csrfToken, csrfToken, checkedAttr(settings.ShowCompleted), settings.UpcomingDays, selectedAttr(settings.UILanguage, "de"), selectedAttr(settings.UILanguage, "en"), selectedAttr(settings.DarkMode, "system"), selectedAttr(settings.DarkMode, "light"), selectedAttr(settings.DarkMode, "dark"), html.EscapeString(proxyUserHeader), httpsStatus)
		return err
	})
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
