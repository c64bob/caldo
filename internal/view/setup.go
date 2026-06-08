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
		_, err := fmt.Fprint(w, `<section class="caldo-page max-w-xl">
		<h2 class="caldo-page-title">CalDAV einrichten</h2>
		<p class="caldo-muted mt-2">Verbindung zu deinem CalDAV-Server testen.</p>`)
		if err != nil {
			return err
		}

		if errorMessage != "" {
			if _, err := fmt.Fprintf(w, `<p class="caldo-alert caldo-alert-error mt-4">%s</p>`, html.EscapeString(errorMessage)); err != nil {
				return err
			}
		}

		csrfToken := html.EscapeString(CSRFToken(ctx))
		_, err = fmt.Fprintf(w, `<form class="mt-6 space-y-4" method="post" action="/setup/caldav" hx-post="/setup/caldav" hx-target="body" hx-swap="outerHTML" hx-push-url="false" hx-headers='{"X-CSRF-Token":"%s"}'>
			<div>
				<label for="caldav_url" class="caldo-label">CalDAV-URL</label>
				<input id="caldav_url" name="caldav_url" type="url" required class="caldo-input"/>
			</div>
			<div>
				<label for="caldav_username" class="caldo-label">Benutzername</label>
				<input id="caldav_username" name="caldav_username" type="text" required class="caldo-input"/>
			</div>
			<div>
				<label for="caldav_password" class="caldo-label">Passwort / App-Passwort</label>
				<input id="caldav_password" name="caldav_password" type="password" required class="caldo-input"/>
			</div>
			<button type="submit" class="caldo-button caldo-button-primary">Verbindung testen</button>
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

		if _, err := fmt.Fprint(w, `<section class="caldo-page max-w-2xl">
		<h2 class="caldo-page-title">Kalender auswählen</h2>
		<p class="caldo-muted mt-2">Wähle die Kalender, die als Projekte synchronisiert werden sollen, und setze ein Default-Projekt.</p>`); err != nil {
			return err
		}
		if errorMessage != "" {
			if _, err := fmt.Fprintf(w, `<p class="caldo-alert caldo-alert-error mt-4">%s</p>`, html.EscapeString(errorMessage)); err != nil {
				return err
			}
		}

		csrfToken := html.EscapeString(CSRFToken(ctx))
		if _, err := fmt.Fprintf(w, `<form class="mt-6 space-y-6" method="post" action="/setup/calendars" hx-post="/setup/calendars" hx-target="body" hx-swap="outerHTML" hx-push-url="false" hx-headers='{"X-CSRF-Token":"%s"}'>`, csrfToken); err != nil {
			return err
		}

		if len(calendars) == 0 {
			if _, err := fmt.Fprint(w, `<p class="caldo-alert caldo-alert-warning">Keine Kalender gefunden.</p>`); err != nil {
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

				if _, err := fmt.Fprintf(w, `<div class="caldo-list-row">
<label class="flex items-center gap-2">
  <input class="caldo-check" type="checkbox" name="calendar_href" value="%s"%s />
  <span class="font-medium">%s</span>
</label>
<label class="caldo-muted mt-2 flex items-center gap-2">
  <input class="caldo-check" type="radio" name="default_calendar_href" value="%s"%s />
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
<label for="new_default_project_name" class="caldo-label">Optional: neues Default-Projekt anlegen</label>
<input id="new_default_project_name" name="new_default_project_name" type="text" class="caldo-input" placeholder="z. B. Inbox"/>
<p class="caldo-meta">Wenn gesetzt, wird ein neuer CalDAV-Kalender angelegt und als Default-Projekt verwendet.</p>
</div>
<button type="submit" class="caldo-button caldo-button-primary">Weiter zum Import</button>
</form>
</section>`); err != nil {
			return err
		}
		return nil
	})
}
