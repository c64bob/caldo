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
<p class="mt-1 text-sm text-slate-600 dark:text-slate-300">CalDAV-Zugang, Kalenderauswahl, Projekt-Mapping und Default-Projekt werden über den Setup-Flow geändert.</p>
<a href="/setup" class="mt-3 inline-block rounded border border-slate-300 px-3 py-1 text-sm dark:border-slate-700">Setup öffnen</a>
</div>
<div class="rounded border border-slate-300 p-4 dark:border-slate-700">
<h3 class="font-medium">Sync</h3>
<p class="mt-2 text-sm">Intervall: %d Minuten</p>
<form class="mt-3" method="post" action="/sync/manual" hx-post="/sync/manual" hx-headers='{"X-CSRF-Token":"%s"}'>
<button type="submit" class="rounded bg-slate-900 px-3 py-1 text-white dark:bg-slate-100 dark:text-slate-900">Jetzt synchronisieren</button>
</form>
</div>
<div class="rounded border border-slate-300 p-4 dark:border-slate-700">
<h3 class="font-medium">UI</h3>
<ul class="mt-2 list-disc pl-5 text-sm text-slate-700 dark:text-slate-300">
<li>Erledigte Aufgaben anzeigen: %t</li>
<li>Demnächst-Zeitraum: %d Tage</li>
<li>Sprache: %s</li>
<li>Dark Mode: %s</li>
</ul>
</div>
<div class="rounded border border-slate-300 p-4 dark:border-slate-700 text-sm">
<h3 class="font-medium">Sicherheitsstatus</h3>
<p class="mt-2">Reverse-Proxy-Header: <code>%s</code></p>
<p>HTTPS-Status: %s</p>
</div>
</section>`, settings.SyncIntervalMinutes, csrfToken, settings.ShowCompleted, settings.UpcomingDays, html.EscapeString(settings.UILanguage), html.EscapeString(settings.DarkMode), html.EscapeString(proxyUserHeader), httpsStatus)
		return err
	})
}
