package view

import (
	"database/sql"
	"html"
	"time"
)

// LocalDateTimeView carries a UTC datetime and fallback text for browser-local rendering.
type LocalDateTimeView struct {
	Text string
	ISO  string
}

// LocalDateTimeFromNull converts a nullable timestamp into a browser-local render model.
func LocalDateTimeFromNull(ts sql.NullTime, empty string) LocalDateTimeView {
	if !ts.Valid {
		return LocalDateTimeView{Text: empty}
	}
	return LocalDateTimeFromTime(ts.Time, empty)
}

// LocalDateTimeFromTime converts a timestamp into a browser-local render model.
func LocalDateTimeFromTime(ts time.Time, empty string) LocalDateTimeView {
	if ts.IsZero() {
		return LocalDateTimeView{Text: empty}
	}
	utc := ts.UTC()
	return LocalDateTimeView{
		Text: utc.Format("02.01.2006 15:04"),
		ISO:  utc.Format(time.RFC3339),
	}
}

func localDateTimeHTML(value LocalDateTimeView) string {
	if value.ISO == "" {
		return html.EscapeString(value.Text)
	}
	return `<time datetime="` + html.EscapeString(value.ISO) + `" data-local-date-time>` + html.EscapeString(value.Text) + `</time>`
}
