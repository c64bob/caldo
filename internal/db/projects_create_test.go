package db

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
)

func TestInsertProjectPersistsRecord(t *testing.T) {
	t.Parallel()

	database, err := OpenSQLite(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	project, err := database.InsertProject(context.Background(), NewProjectInput{
		CalendarHref: "/calendars/work/",
		DisplayName:  "Work",
		SyncStrategy: "ctag",
	})
	if err != nil {
		t.Fatalf("insert project: %v", err)
	}
	if project.ID == "" {
		t.Fatal("expected generated project id")
	}

	var gotHref, gotName, gotStrategy string
	var gotVersion int
	if err := database.Conn.QueryRowContext(context.Background(), `
SELECT calendar_href, display_name, sync_strategy, server_version
FROM projects
WHERE id = ?;
`, project.ID).Scan(&gotHref, &gotName, &gotStrategy, &gotVersion); err != nil {
		t.Fatalf("query project: %v", err)
	}

	if gotHref != "/calendars/work/" || gotName != "Work" || gotStrategy != "ctag" || gotVersion != 1 {
		t.Fatalf("unexpected persisted project: href=%q name=%q strategy=%q version=%d", gotHref, gotName, gotStrategy, gotVersion)
	}
}

func TestInsertProjectDefaultsSyncStrategy(t *testing.T) {
	t.Parallel()

	database, err := OpenSQLite(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	project, err := database.InsertProject(context.Background(), NewProjectInput{
		CalendarHref: "/calendars/inbox/",
		DisplayName:  "Inbox",
	})
	if err != nil {
		t.Fatalf("insert project: %v", err)
	}

	if project.SyncStrategy != "fullscan" {
		t.Fatalf("unexpected default strategy: got %q want %q", project.SyncStrategy, "fullscan")
	}
}

func TestInsertProjectValidatesRequiredFields(t *testing.T) {
	t.Parallel()

	database, err := OpenSQLite(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	_, err = database.InsertProject(context.Background(), NewProjectInput{DisplayName: "Name"})
	if err == nil || !strings.Contains(err.Error(), "calendar href is required") {
		t.Fatalf("expected calendar href validation error, got %v", err)
	}

	_, err = database.InsertProject(context.Background(), NewProjectInput{CalendarHref: "/cal/", DisplayName: "   "})
	if err == nil || !strings.Contains(err.Error(), "display name is required") {
		t.Fatalf("expected display name validation error, got %v", err)
	}
}
