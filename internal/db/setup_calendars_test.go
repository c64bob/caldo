package db

import (
	"context"
	"path/filepath"
	"testing"
)

func TestSaveSetupCalendarsStoresProjectsDefaultAndStep(t *testing.T) {
	t.Parallel()

	database, err := OpenSQLite(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	err = database.SaveSetupCalendars(context.Background(), []SelectedCalendar{
		{Href: "/calendars/work/", DisplayName: "Work"},
		{Href: "/calendars/home/", DisplayName: "Home"},
	}, "/calendars/home/", "webdav_sync")
	if err != nil {
		t.Fatalf("save setup calendars: %v", err)
	}

	var (
		projectCount int
		defaultCount int
		setupStep    string
	)
	if err := database.Conn.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM projects;`).Scan(&projectCount); err != nil {
		t.Fatalf("count projects: %v", err)
	}
	if projectCount != 2 {
		t.Fatalf("unexpected project count: got %d want %d", projectCount, 2)
	}

	if err := database.Conn.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM projects WHERE is_default = TRUE;`).Scan(&defaultCount); err != nil {
		t.Fatalf("count default projects: %v", err)
	}
	if defaultCount != 1 {
		t.Fatalf("unexpected default project count: got %d want %d", defaultCount, 1)
	}

	if err := database.Conn.QueryRowContext(context.Background(), `SELECT setup_step FROM settings WHERE id = 'default';`).Scan(&setupStep); err != nil {
		t.Fatalf("load setup step: %v", err)
	}
	if setupStep != "import" {
		t.Fatalf("unexpected setup step: got %q want %q", setupStep, "import")
	}
}

func TestLoadCalDAVServerCapabilitiesDefaultsToFullScan(t *testing.T) {
	t.Parallel()

	database, err := OpenSQLite(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	capabilities, err := database.LoadCalDAVServerCapabilities(context.Background())
	if err != nil {
		t.Fatalf("load capabilities: %v", err)
	}
	if capabilities.WebDAVSync || capabilities.CTag || capabilities.ETag || !capabilities.FullScan {
		t.Fatalf("unexpected default capabilities: %#v", capabilities)
	}
}
