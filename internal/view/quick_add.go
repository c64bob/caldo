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
		if _, err := fmt.Fprint(w, `<section class="caldo-page"><h2 class="caldo-page-title">Quick Add</h2><form method="post" action="/quick-add/preview" hx-post="/quick-add/preview" hx-target="#quick-add-preview" hx-swap="outerHTML" hx-disabled-elt="find button" class="space-y-3"><label for="quick-add-text" class="caldo-label">Aufgabe</label><input id="quick-add-text" name="text" type="text" class="caldo-input" autofocus value="`+html.EscapeString(text)+`"/><button type="submit" accesskey="q" class="caldo-button caldo-button-secondary">Vorschau</button><span class="htmx-indicator caldo-meta" aria-live="polite">Vorschau wird erstellt ...</span></form><div id="quick-add-preview" class="min-h-24">`); err != nil {
			return err
		}
		if errorMessage != "" {
			if _, err := fmt.Fprintf(w, `<p class="caldo-alert caldo-alert-error">%s</p>`, html.EscapeString(errorMessage)); err != nil {
				return err
			}
		}
		if draft != nil {
			if err := quickAddPreviewContent(*draft).Render(ctx, w); err != nil {
				return err
			}
			_, err := fmt.Fprint(w, `</div></section>`)
			return err
		}
		_, err := fmt.Fprint(w, `</div></section>`)
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
		if draft.ProjectUnresolved {
			projectDetail += ` <span class="text-amber-600">(unbekannt – wird beim Speichern ignoriert)</span>`
		}
		_, err := fmt.Fprint(w, `<section id="quick-add-preview" class="caldo-card"><h3 class="font-medium">Vorschau</h3><ul class="mt-2 text-sm space-y-1"><li><strong>Titel:</strong> `+html.EscapeString(draft.Title)+`</li><li><strong>Projekt:</strong> `+projectDetail+`</li><li><strong>Labels:</strong> `+labels+`</li><li><strong>Datum:</strong> `+html.EscapeString(draft.Due)+`</li><li><strong>Wiederholung:</strong> `+html.EscapeString(draft.Recurrence)+`</li><li><strong>Priorität:</strong> `+html.EscapeString(draft.Priority)+`</li></ul><form method="post" action="/tasks" hx-post="/tasks" hx-disabled-elt="find button" hx-headers='{"X-CSRF-Token":"`+csrfToken+`"}' class="mt-3"><input type="hidden" name="title" value="`+html.EscapeString(draft.Title)+`"/><input type="hidden" name="project_id" value="`+html.EscapeString(draft.ProjectID)+`"/><input type="hidden" name="labels" value="`+labels+`"/><input type="hidden" name="priority" value="`+html.EscapeString(draft.Priority)+`"/><input type="hidden" name="recurrence" value="`+html.EscapeString(draft.Recurrence)+`"/><button type="submit" class="caldo-button caldo-button-primary">Speichern</button><span class="htmx-indicator caldo-meta ml-2" aria-live="polite">Speichern ...</span></form></section>`)
		return err
	})
}
