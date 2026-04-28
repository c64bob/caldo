package model

import (
	"fmt"
	"strings"
	"time"
)

// BuildTaskVTODO builds a minimal VTODO payload for a newly created task.
func BuildTaskVTODO(uid string, title string, now time.Time) string {
	escapedTitle := escapeICalendarText(strings.TrimSpace(title))
	timestamp := now.UTC().Format("20060102T150405Z")

	return strings.Join([]string{
		"BEGIN:VCALENDAR",
		"VERSION:2.0",
		"PRODID:-//Caldo//EN",
		"BEGIN:VTODO",
		fmt.Sprintf("UID:%s", strings.TrimSpace(uid)),
		fmt.Sprintf("DTSTAMP:%s", timestamp),
		fmt.Sprintf("SUMMARY:%s", escapedTitle),
		"STATUS:NEEDS-ACTION",
		"END:VTODO",
		"END:VCALENDAR",
		"",
	}, "\r\n")
}

func escapeICalendarText(value string) string {
	replacer := strings.NewReplacer(
		`\\`, `\\\\`,
		`;`, `\\;`,
		`,`, `\\,`,
		"\n", `\\n`,
	)
	return replacer.Replace(value)
}
