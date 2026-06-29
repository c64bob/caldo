package view

import (
	"bytes"
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	"caldo/internal/caldav"
	"caldo/internal/db"
)

func TestSettingsPageContentRendersCalDAVCalendarSyncUIAndSecuritySettings(t *testing.T) {
	t.Parallel()

	ctx := WithCSRFToken(context.Background(), "token-123")
	lastSync := time.Date(2026, time.January, 2, 3, 4, 0, 0, time.Local)
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
				{ID: "project-1", CalendarHref: "/cal/work/", DisplayName: "Work", SyncStrategy: "fullscan", IsDefault: true, OpenTaskCount: 2, TaskCount: 3},
				{ID: "project-archive", CalendarHref: "/cal/archive/", DisplayName: "Archive", SyncStrategy: "fullscan", TaskCount: 1},
			},
		},
		SyncStatus: db.SyncStatus{
			State:         "idle",
			LastFinished:  sql.NullTime{Time: lastSync, Valid: true},
			LastSuccessAt: sql.NullTime{Time: lastSync, Valid: true},
			LastErrorCode: sql.NullString{String: "sync_failed", Valid: true},
		},
		Available: []caldav.Calendar{
			{Href: "/cal/work/", DisplayName: "Work"},
			{Href: "/cal/home/", DisplayName: "Home"},
		},
		CalendarsLoaded:  true,
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
		`data-caldav-test-result="success"`,
		`caldo-alert-success`,
		`verbindungstest erfolgreich`,
		`name="caldav_action" value="test"`,
		`Verbindung testen`,
		`name="caldav_action" value="save"`,
		`CalDAV speichern`,
		`unverändert lassen`,
		`data-settings-calendars`,
		`action="/settings/calendars"`,
		`data-settings-calendar-state="synced"`,
		`name="calendar_href" value="/cal/work/" checked`,
		`name="default_calendar_href" value="/cal/work/" checked`,
		`Lokal und remote synchronisiert`,
		`Default-Projekt`,
		`Projekt: Work`,
		`Sync-Strategie: fullscan`,
		`data-calendar-remove-impact`,
		`Beim Entfernen bleiben lokale Aufgaben erhalten`,
		`data-settings-calendar-state="remote-only"`,
		`Remote verfügbar, noch nicht lokal synchronisiert`,
		`data-settings-calendar-state="remote-missing"`,
		`Remote nicht gefunden`,
		`Archive`,
		`data-settings-sync-state="idle"`,
		`Letzter erfolgreicher Sync`,
		`02.01.2026 03:04`,
		`data-settings-sync-error`,
		`sync fehlgeschlagen`,
		`action="/settings/sync" hx-post="/settings/sync" hx-target="body" hx-swap="outerHTML" hx-push-url="false"`,
		`name="sync_interval_minutes" value="15"`,
		`action="/sync/manual" hx-post="/sync/manual" hx-target="#sync-status" hx-swap="outerHTML"`,
		`name="show_completed" checked`,
		`action="/settings/ui" hx-post="/settings/ui" hx-target="body" hx-swap="outerHTML" hx-push-url="false"`,
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
	if strings.Contains(output, `value="secret"`) {
		t.Fatalf("settings page must not render plaintext caldav password: %s", output)
	}
}

func TestSettingsPageContentDoesNotMarkRemoteMissingWhenCalendarsDidNotLoad(t *testing.T) {
	t.Parallel()

	ctx := WithCSRFToken(context.Background(), "token-123")
	component := SettingsPageContent(SettingsPageView{
		Settings: db.AppSettings{
			SyncIntervalMinutes: 15,
			UpcomingDays:        7,
			UILanguage:          "de",
			DarkMode:            "system",
			Projects: []db.SettingsProject{
				{ID: "project-1", CalendarHref: "/cal/work/", DisplayName: "Work", SyncStrategy: "fullscan", IsDefault: true, TaskCount: 1},
			},
		},
		CalendarsError: "kalender konnten nicht geladen werden",
	})

	var rendered bytes.Buffer
	if err := component.Render(ctx, &rendered); err != nil {
		t.Fatalf("render settings page: %v", err)
	}

	output := rendered.String()
	if strings.Contains(output, `data-settings-calendar-state="remote-missing"`) {
		t.Fatalf("calendar load failure must not be rendered as remote-missing state: %s", output)
	}
	if !strings.Contains(output, "kalender konnten nicht geladen werden") {
		t.Fatalf("expected calendar load error in settings page: %s", output)
	}
}
