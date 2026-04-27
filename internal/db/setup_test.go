package db

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
)

func TestLoadSetupStatusReturnsDefaults(t *testing.T) {
	t.Parallel()

	database, err := OpenSQLite(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() {
		if err := database.Close(); err != nil {
			t.Fatalf("close db: %v", err)
		}
	})

	status, err := database.LoadSetupStatus(context.Background())
	if err != nil {
		t.Fatalf("load setup status: %v", err)
	}

	if status.Complete {
		t.Fatal("expected setup_complete default to false")
	}
	if status.Step != "caldav" {
		t.Fatalf("unexpected setup_step: got %q want %q", status.Step, "caldav")
	}
}

func TestSaveSetupStepPersistsStep(t *testing.T) {
	t.Parallel()

	database, err := OpenSQLite(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() {
		if err := database.Close(); err != nil {
			t.Fatalf("close db: %v", err)
		}
	})

	if err := database.SaveSetupStep(context.Background(), "calendars"); err != nil {
		t.Fatalf("save setup step: %v", err)
	}

	status, err := database.LoadSetupStatus(context.Background())
	if err != nil {
		t.Fatalf("load setup status: %v", err)
	}
	if status.Step != "calendars" {
		t.Fatalf("unexpected setup_step: got %q want %q", status.Step, "calendars")
	}
}

func TestCompleteSetupMarksCompleteAndStep(t *testing.T) {
	t.Parallel()

	database, err := OpenSQLite(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	if err := database.SaveSetupCalendars(context.Background(), []SelectedCalendar{{Href: "/cal/work/", DisplayName: "Work"}}, "/cal/work/", "fullscan"); err != nil {
		t.Fatalf("save setup calendars: %v", err)
	}

	if err := database.CompleteSetup(context.Background()); err != nil {
		t.Fatalf("complete setup: %v", err)
	}

	status, err := database.LoadSetupStatus(context.Background())
	if err != nil {
		t.Fatalf("load setup status: %v", err)
	}
	if !status.Complete {
		t.Fatal("expected setup to be complete")
	}
	if status.Step != "complete" {
		t.Fatalf("unexpected setup step: got %q want %q", status.Step, "complete")
	}
}

func TestCompleteSetupRequiresImportStep(t *testing.T) {
	t.Parallel()

	database, err := OpenSQLite(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	if err := database.CompleteSetup(context.Background()); !errors.Is(err, ErrSetupPrerequisitesNotMet) {
		t.Fatalf("expected ErrSetupPrerequisitesNotMet, got %v", err)
	}
}

func TestCompleteSetupRequiresNoUnsyncedTasks(t *testing.T) {
	t.Parallel()

	database, err := OpenSQLite(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	if err := database.SaveSetupCalendars(context.Background(), []SelectedCalendar{{Href: "/cal/work/", DisplayName: "Work"}}, "/cal/work/", "fullscan"); err != nil {
		t.Fatalf("save setup calendars: %v", err)
	}

	var projectID string
	if err := database.Conn.QueryRowContext(context.Background(), `SELECT id FROM projects LIMIT 1;`).Scan(&projectID); err != nil {
		t.Fatalf("load project id: %v", err)
	}

	if _, err := database.Conn.ExecContext(context.Background(), `
INSERT INTO tasks (
	id, project_id, uid, href, etag, title, status, raw_vtodo, sync_status, created_at, updated_at
) VALUES (
	'task-1', ?, 'uid-1', '/cal/work/uid-1.ics', '"etag-1"', 'Task', 'NEEDS-ACTION', 'BEGIN:VCALENDAR', 'pending', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
);`, projectID); err != nil {
		t.Fatalf("insert task: %v", err)
	}

	if err := database.CompleteSetup(context.Background()); !errors.Is(err, ErrSetupPrerequisitesNotMet) {
		t.Fatalf("expected ErrSetupPrerequisitesNotMet, got %v", err)
	}
}
