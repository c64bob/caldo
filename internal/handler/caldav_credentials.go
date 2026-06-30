package handler

import (
	"net/url"
	"strings"

	"caldo/internal/caldav"
	"caldo/internal/db"
)

func calendarOperationCredentials(credentials db.CalDAVCredentials, capabilities db.CalDAVServerCapabilities) caldav.Credentials {
	return caldav.Credentials{
		URL:      effectiveCalendarBaseURL(credentials.URL, capabilities.CalendarHomeSet),
		Username: credentials.Username,
		Password: credentials.Password,
	}
}

func effectiveCalendarBaseURL(baseURL string, calendarHomeSet string) string {
	trimmedBase := strings.TrimSpace(baseURL)
	trimmedHomeSet := strings.TrimSpace(calendarHomeSet)
	if trimmedHomeSet == "" {
		return trimmedBase
	}
	if strings.HasPrefix(trimmedHomeSet, "http://") || strings.HasPrefix(trimmedHomeSet, "https://") {
		return trimmedHomeSet
	}

	base, err := url.Parse(trimmedBase)
	if err != nil {
		return trimmedBase
	}

	href := trimmedHomeSet
	if !strings.HasPrefix(href, "/") {
		basePath := strings.TrimSuffix(base.Path, "/")
		if basePath == "" {
			href = "/" + href
		} else {
			href = basePath + "/" + href
		}
	}

	relative, err := url.Parse(href)
	if err != nil {
		return trimmedBase
	}
	return base.ResolveReference(relative).String()
}
