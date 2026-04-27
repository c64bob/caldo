package view

import (
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

		_, err = fmt.Fprint(w, `<form class="mt-6 space-y-4" method="post" action="/setup/caldav">
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
	</section>`)
		return err
	})
}
