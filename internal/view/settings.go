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
		_, err := fmt.Fprintf(w, `<section class="max-w-3xl space-y-6">
<h2 class="text-xl font-semibold">Einstellungen</h2>
<div class="rounded border border-slate-300 p-4 dark:border-slate-700">
<h3 class="font-medium">CalDAV & Projekte</h3>
<p class="mt-1 text-sm text-slate-600 dark:text-slate-300">CalDAV-Zugang, Kalenderauswahl, Projekt-Mapping und Default-Projekt werden über die Einstellungen-Routen aktualisiert.</p>
</div>
<div class="rounded border border-slate-300 p-4 dark:border-slate-700">
<h3 class="font-medium">Sync</h3>
<form class="mt-3 space-y-2" method="post" action="/settings/sync" hx-post="/settings/sync" hx-headers='{"X-CSRF-Token":"%s"}'>
<label class="block text-sm">Intervall (Minuten)
<input class="mt-1 w-32 rounded border border-slate-300 px-2 py-1 dark:border-slate-700" type="number" min="1" name="sync_interval_minutes" value="%d">
</label>
<button type="submit" class="rounded border border-slate-300 px-3 py-1 text-sm dark:border-slate-700">Sync-Einstellungen speichern</button>
</form>
<form class="mt-3" method="post" action="/sync/manual" hx-post="/sync/manual" hx-headers='{"X-CSRF-Token":"%s"}'>
<button type="submit" class="rounded bg-slate-900 px-3 py-1 text-white dark:bg-slate-100 dark:text-slate-900">Jetzt synchronisieren</button>
</form>
</div>
<div class="rounded border border-slate-300 p-4 dark:border-slate-700">
<h3 class="font-medium">UI</h3>
<form class="mt-3 space-y-3 text-sm" method="post" action="/settings/ui" hx-post="/settings/ui" hx-headers='{"X-CSRF-Token":"%s"}'>
<label class="flex items-center gap-2"><input type="checkbox" name="show_completed" %s> Erledigte Aufgaben anzeigen</label>
<label class="block">Demnächst-Zeitraum (Tage)
<input class="mt-1 w-32 rounded border border-slate-300 px-2 py-1 dark:border-slate-700" type="number" min="1" name="upcoming_days" value="%d">
</label>
<label class="block">Sprache
<select class="mt-1 w-48 rounded border border-slate-300 px-2 py-1 dark:border-slate-700 dark:bg-slate-900" name="ui_language">
<option value="de" %s>Deutsch</option>
<option value="en" %s>English</option>
</select>
</label>
<label class="block">Dark Mode
<select class="mt-1 w-48 rounded border border-slate-300 px-2 py-1 dark:border-slate-700 dark:bg-slate-900" name="dark_mode">
<option value="system" %s>System</option>
<option value="light" %s>Hell</option>
<option value="dark" %s>Dunkel</option>
</select>
</label>
<button type="submit" class="rounded border border-slate-300 px-3 py-1 dark:border-slate-700">UI-Einstellungen speichern</button>
</form>
</div>
<div class="rounded border border-slate-300 p-4 dark:border-slate-700 text-sm">
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
