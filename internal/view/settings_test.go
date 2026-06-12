package view

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"caldo/internal/caldav"
	"caldo/internal/db"
)

func TestSettingsPageContentRendersCalDAVCalendarSyncUIAndSecuritySettings(t *testing.T) {
	t.Parallel()

	ctx := WithCSRFToken(context.Background(), "token-123")
	component := SettingsPageContent(SettingsPageView{
		Settings: db.AppSettings{
			SyncIntervalMinutes: 15,
			UpcomingDays:        7,
			ShowCompleted:       true,
			UILanguage:          "de",
			DarkMode:            "system",
			CalDAVURL:           "https://nextcloud.example/remote.php/dav",
			CalDAVUsername:      "alice",
			CalDAVConfigured:    true,
			Projects: []db.SettingsProject{
				{ID: "project-1", CalendarHref: "/cal/work/", DisplayName: "Work", IsDefault: true, OpenTaskCount: 2, TaskCount: 3},
			},
		},
		Available: []caldav.Calendar{
			{Href: "/cal/work/", DisplayName: "Work"},
			{Href: "/cal/home/", DisplayName: "Home"},
		},
		CalDAVSuccess:    "verbindungstest erfolgreich",
		ProxyUserHeader:  "X-Forwarded-User",
		ProxyUserPresent: true,
		HTTPSConfigured:  true,
	})

	var rendered bytes.Buffer
	if err := component.Render(ctx, &rendered); err != nil {
		t.Fatalf("render settings page: %v", err)
	}

	output := rendered.String()
	for _, want := range []string{
		`action="/settings/caldav"`,
		`hx-post="/settings/caldav"`,
		`name="caldav_url"`,
		`value="https://nextcloud.example/remote.php/dav"`,
		`name="caldav_username"`,
		`value="alice"`,
		`name="caldav_password"`,
		`caldo-alert-success`,
		`verbindungstest erfolgreich`,
		`name="caldav_action" value="test"`,
		`Verbindung testen`,
		`name="caldav_action" value="save"`,
		`CalDAV speichern`,
		`unverändert lassen`,
		`action="/settings/calendars"`,
		`name="calendar_href" value="/cal/work/" checked`,
		`name="default_calendar_href" value="/cal/work/" checked`,
		`Projekt: Work`,
		`name="sync_interval_minutes" value="15"`,
		`action="/sync/manual"`,
		`name="show_completed" checked`,
		`name="upcoming_days" value="7"`,
		`name="ui_language"`,
		`name="dark_mode"`,
		`Reverse-Proxy-Header`,
		`X-Forwarded-User`,
		`erkannt`,
		`HTTPS-Status: aktiv`,
		`hx-headers='{"X-CSRF-Token":"token-123"}'`,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected settings page to include %q in %s", want, output)
		}
	}
}
