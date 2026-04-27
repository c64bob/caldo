package db

import (
	"context"
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
