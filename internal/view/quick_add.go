package view

import (
	"caldo/internal/parser"
	"context"
	"fmt"
	"html"
	"io"
	"strings"

	"github.com/a-h/templ"
)

// QuickAddPage renders the quick-add page.
func QuickAddPage(draft *parser.QuickAddDraft, text string, errorMessage string) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		if _, err := fmt.Fprint(w, `<section class="space-y-4"><h2 class="text-xl font-semibold">Quick Add</h2><form method="post" action="/quick-add/preview" class="space-y-3"><label for="quick-add-text" class="block text-sm font-medium">Aufgabe</label><input id="quick-add-text" name="text" type="text" class="w-full rounded border border-slate-300 px-3 py-2 dark:border-slate-700 dark:bg-slate-900" autofocus value="`+html.EscapeString(text)+`"/><button type="submit" accesskey="q" class="rounded border border-slate-300 px-3 py-2 dark:border-slate-700">Vorschau (Shortcut: Alt+Shift+Q)</button></form>`); err != nil {
			return err
		}
		if errorMessage != "" {
			if _, err := fmt.Fprintf(w, `<p class="text-sm text-red-600">%s</p>`, html.EscapeString(errorMessage)); err != nil {
				return err
			}
		}
		if draft != nil {
			return quickAddPreviewContent(*draft).Render(ctx, w)
		}
		_, err := fmt.Fprint(w, `</section>`)
		return err
	})
}

// QuickAddPreview renders a quick-add preview snippet.
func QuickAddPreview(draft parser.QuickAddDraft, _ string) templ.Component {
	return quickAddPreviewContent(draft)
}

func quickAddPreviewContent(draft parser.QuickAddDraft) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		labels := html.EscapeString(strings.Join(draft.Labels, ", "))
		csrfToken := html.EscapeString(CSRFToken(ctx))
		projectDetail := html.EscapeString(draft.Project)
		if draft.ProjectNew {
			projectDetail += ` <span class="text-amber-600">(neu – wird beim Speichern erstellt)</span>`
		}
		_, err := fmt.Fprint(w, `<section class="rounded border border-slate-300 p-4 dark:border-slate-700"><h3 class="font-medium">Vorschau</h3><ul class="mt-2 text-sm space-y-1"><li><strong>Titel:</strong> `+html.EscapeString(draft.Title)+`</li><li><strong>Projekt:</strong> `+projectDetail+`</li><li><strong>Labels:</strong> `+labels+`</li><li><strong>Datum:</strong> `+html.EscapeString(draft.Due)+`</li><li><strong>Wiederholung:</strong> `+html.EscapeString(draft.Recurrence)+`</li><li><strong>Priorität:</strong> `+html.EscapeString(draft.Priority)+`</li></ul><form method="post" action="/tasks" hx-post="/tasks" hx-headers='{"X-CSRF-Token":"`+csrfToken+`"}' class="mt-3"><input type="hidden" name="title" value="`+html.EscapeString(draft.Title)+`"/><input type="hidden" name="project_id" value="`+html.EscapeString(draft.ProjectID)+`"/><button type="submit" class="rounded bg-slate-900 px-3 py-2 text-white dark:bg-slate-100 dark:text-slate-900">Speichern</button></form></section></section>`)
		return err
	})
}
