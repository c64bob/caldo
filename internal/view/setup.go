package view

import (
	"caldo/internal/caldav"
	"context"
	"fmt"
	"html"
	"io"

	"github.com/a-h/templ"
)

// SetupCalDAVContent renders the setup CalDAV credential form.
func SetupCalDAVContent(errorMessage string) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		_, err := fmt.Fprint(w, `<section class="max-w-xl">
		<h2 class="text-xl font-semibold">CalDAV einrichten</h2>
		<p class="mt-2 text-sm text-slate-700 dark:text-slate-300">Verbindung zu deinem CalDAV-Server testen.</p>`)
		if err != nil {
			return err
		}

		if errorMessage != "" {
			if _, err := fmt.Fprintf(w, `<p class="mt-4 rounded border border-red-300 bg-red-50 p-3 text-sm text-red-800 dark:border-red-800 dark:bg-red-950 dark:text-red-200">%s</p>`, html.EscapeString(errorMessage)); err != nil {
				return err
			}
		}

		csrfToken := html.EscapeString(CSRFToken(ctx))
		_, err = fmt.Fprintf(w, `<form class="mt-6 space-y-4" method="post" action="/setup/caldav" hx-post="/setup/caldav" hx-target="body" hx-swap="outerHTML" hx-push-url="false" hx-headers='{"X-CSRF-Token":"%s"}'>
			<div>
				<label for="caldav_url" class="block text-sm font-medium">CalDAV-URL</label>
				<input id="caldav_url" name="caldav_url" type="url" required class="mt-1 w-full rounded border border-slate-300 px-3 py-2 dark:border-slate-700 dark:bg-slate-900"/>
			</div>
			<div>
				<label for="caldav_username" class="block text-sm font-medium">Benutzername</label>
				<input id="caldav_username" name="caldav_username" type="text" required class="mt-1 w-full rounded border border-slate-300 px-3 py-2 dark:border-slate-700 dark:bg-slate-900"/>
			</div>
			<div>
				<label for="caldav_password" class="block text-sm font-medium">Passwort / App-Passwort</label>
				<input id="caldav_password" name="caldav_password" type="password" required class="mt-1 w-full rounded border border-slate-300 px-3 py-2 dark:border-slate-700 dark:bg-slate-900"/>
			</div>
			<button type="submit" class="rounded bg-slate-900 px-4 py-2 text-white dark:bg-slate-100 dark:text-slate-900">Verbindung testen</button>
		</form>
	</section>`, csrfToken)
		return err
	})
}

// SetupCalendarsContent renders setup step 2 for calendar selection and default project choice.
func SetupCalendarsContent(calendars []caldav.Calendar, errorMessage string, selectedHrefs []string) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		selected := make(map[string]struct{}, len(selectedHrefs))
		for _, href := range selectedHrefs {
			selected[href] = struct{}{}
		}
		if len(selected) == 0 {
			for _, calendar := range calendars {
				selected[calendar.Href] = struct{}{}
			}
		}
		firstSelectedHref := ""
		for _, calendar := range calendars {
			if _, ok := selected[calendar.Href]; ok {
				firstSelectedHref = calendar.Href
				break
			}
		}

		if _, err := fmt.Fprint(w, `<section class="max-w-2xl">
		<h2 class="text-xl font-semibold">Kalender auswählen</h2>
		<p class="mt-2 text-sm text-slate-700 dark:text-slate-300">Wähle die Kalender, die als Projekte synchronisiert werden sollen, und setze ein Default-Projekt.</p>`); err != nil {
			return err
		}
		if errorMessage != "" {
			if _, err := fmt.Fprintf(w, `<p class="mt-4 rounded border border-red-300 bg-red-50 p-3 text-sm text-red-800 dark:border-red-800 dark:bg-red-950 dark:text-red-200">%s</p>`, html.EscapeString(errorMessage)); err != nil {
				return err
			}
		}

		csrfToken := html.EscapeString(CSRFToken(ctx))
		if _, err := fmt.Fprintf(w, `<form class="mt-6 space-y-6" method="post" action="/setup/calendars" hx-post="/setup/calendars" hx-target="body" hx-swap="outerHTML" hx-push-url="false" hx-headers='{"X-CSRF-Token":"%s"}'>`, csrfToken); err != nil {
			return err
		}

		if len(calendars) == 0 {
			if _, err := fmt.Fprint(w, `<p class="rounded border border-amber-300 bg-amber-50 p-3 text-sm text-amber-800 dark:border-amber-800 dark:bg-amber-950 dark:text-amber-200">Keine Kalender gefunden.</p>`); err != nil {
				return err
			}
		} else {
			if _, err := fmt.Fprint(w, `<div class="space-y-3">`); err != nil {
				return err
			}
			for index, calendar := range calendars {
				isSelected := false
				_, isSelected = selected[calendar.Href]
				checked := ""
				defaultChecked := ""
				if isSelected {
					checked = ` checked`
				}
				if (index == 0 && firstSelectedHref == "") || calendar.Href == firstSelectedHref {
					defaultChecked = ` checked`
				}

				if _, err := fmt.Fprintf(w, `<div class="rounded border border-slate-200 p-3 dark:border-slate-700">
<label class="flex items-center gap-2">
  <input type="checkbox" name="calendar_href" value="%s"%s />
  <span class="font-medium">%s</span>
</label>
<label class="mt-2 flex items-center gap-2 text-sm text-slate-600 dark:text-slate-300">
  <input type="radio" name="default_calendar_href" value="%s"%s />
  <span>Als Default-Projekt verwenden</span>
</label>
</div>`, html.EscapeString(calendar.Href), checked, html.EscapeString(calendar.DisplayName), html.EscapeString(calendar.Href), defaultChecked); err != nil {
					return err
				}
			}
			if _, err := fmt.Fprint(w, `</div>`); err != nil {
				return err
			}
		}

		if _, err := fmt.Fprint(w, `<div class="space-y-2">
<label for="new_default_project_name" class="block text-sm font-medium">Optional: neues Default-Projekt anlegen</label>
<input id="new_default_project_name" name="new_default_project_name" type="text" class="mt-1 w-full rounded border border-slate-300 px-3 py-2 dark:border-slate-700 dark:bg-slate-900" placeholder="z. B. Inbox"/>
<p class="text-xs text-slate-600 dark:text-slate-300">Wenn gesetzt, wird ein neuer CalDAV-Kalender angelegt und als Default-Projekt verwendet.</p>
</div>
<button type="submit" class="rounded bg-slate-900 px-4 py-2 text-white dark:bg-slate-100 dark:text-slate-900">Weiter zum Import</button>
</form>
</section>`); err != nil {
			return err
		}
		return nil
	})
}
