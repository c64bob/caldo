package view

import (
	"github.com/a-h/templ"
	templruntime "github.com/a-h/templ/runtime"
)

func SyncStatusBadge(state string, lastSuccess string) templ.Component {
	return templruntime.GeneratedTemplate(func(in templruntime.GeneratedComponentInput) error {
		w := in.Writer
		if _, err := w.Write([]byte("<section class=\"flex items-center gap-2\" id=\"sync-status\" sse-connect=\"/events\" sse-swap=\"sync-status\"><button type=\"button\" class=\"rounded border border-slate-300 px-3 py-1 text-sm dark:border-slate-700\" hx-post=\"/sync/manual\" hx-target=\"#sync-status\" hx-swap=\"outerHTML\">Jetzt synchronisieren</button><p class=\"text-sm text-slate-600 dark:text-slate-400\">Status: ")); err != nil { return err }
		if _, err := w.Write([]byte(templ.EscapeString(state))); err != nil { return err }
		if _, err := w.Write([]byte(" · Letzter erfolgreicher Sync: ")); err != nil { return err }
		if _, err := w.Write([]byte(templ.EscapeString(lastSuccess))); err != nil { return err }
		_, err := w.Write([]byte("</p></section>"))
		return err
	})
}
