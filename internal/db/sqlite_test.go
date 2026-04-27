package db

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOpenSQLiteConfiguresPragmasAndPool(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "caldo.db")
	database, err := OpenSQLite(dbPath)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() {
		if err := database.Close(); err != nil {
			t.Fatalf("close sqlite: %v", err)
		}
	})

	if _, err := os.Stat(dbPath); err != nil {
		t.Fatalf("expected sqlite file at configured path: %v", err)
	}

	if got := database.Conn.Stats().MaxOpenConnections; got != 1 {
		t.Fatalf("unexpected max open conns: got %d want 1", got)
	}

	assertSingleTextResult(t, database, "PRAGMA journal_mode;", "wal")
	assertSingleIntResult(t, database, "PRAGMA synchronous;", 1)
	assertSingleIntResult(t, database, "PRAGMA busy_timeout;", busyTimeoutMs)
}

func TestOpenSQLiteRunsMigrationsAndCreatesBackup(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "caldo.db")
	database, err := OpenSQLite(dbPath)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() {
		if err := database.Close(); err != nil {
			t.Fatalf("close sqlite: %v", err)
		}
	})

	assertSingleIntResult(t, database, `SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='schema_migrations';`, 1)
	assertSingleIntResult(t, database, `SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='settings';`, 1)
	assertSingleIntResult(t, database, `SELECT COUNT(*) FROM settings WHERE id = 'default';`, 1)

	backupMatches, err := filepath.Glob(dbPath + ".backup-*")
	if err != nil {
		t.Fatalf("glob backup files: %v", err)
	}
	if len(backupMatches) == 0 {
		t.Fatal("expected backup file before first pending migration")
	}
}

func TestOpenSQLiteSeedsSettingsSingletonWithExpectedDefaults(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "caldo.db")
	database, err := OpenSQLite(dbPath)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() {
		if err := database.Close(); err != nil {
			t.Fatalf("close sqlite: %v", err)
		}
	})

	var (
		id                  string
		setupComplete       bool
		setupStep           string
		syncIntervalMinutes int
		uiLanguage          string
		darkMode            string
		defaultProjectID    *string
	)

	err = database.Conn.QueryRow(`
SELECT id, setup_complete, setup_step, sync_interval_minutes, ui_language, dark_mode, default_project_id
FROM settings
`).Scan(&id, &setupComplete, &setupStep, &syncIntervalMinutes, &uiLanguage, &darkMode, &defaultProjectID)
	if err != nil {
		t.Fatalf("query settings singleton: %v", err)
	}

	if id != "default" {
		t.Fatalf("unexpected settings id: got %q want %q", id, "default")
	}
	if setupComplete {
		t.Fatal("setup_complete should default to false")
	}
	if setupStep != "caldav" {
		t.Fatalf("unexpected setup_step default: got %q want %q", setupStep, "caldav")
	}
	if syncIntervalMinutes != 15 {
		t.Fatalf("unexpected sync interval default: got %d want %d", syncIntervalMinutes, 15)
	}
	if uiLanguage != "de" {
		t.Fatalf("unexpected ui language default: got %q want %q", uiLanguage, "de")
	}
	if darkMode != "system" {
		t.Fatalf("unexpected dark_mode default: got %q want %q", darkMode, "system")
	}
	if defaultProjectID != nil {
		t.Fatalf("default_project_id should be NULL before setup completion, got %q", *defaultProjectID)
	}
}

func TestDatabaseCloseNilReceiver(t *testing.T) {
	t.Parallel()

	var database *Database
	if err := database.Close(); err != nil {
		t.Fatalf("nil close should be no-op: %v", err)
	}
}

func TestOpenSQLiteFailsWhenWalUnavailable(t *testing.T) {
	t.Parallel()

	database, err := OpenSQLite(":memory:")
	if err == nil {
		_ = database.Close()
		t.Fatal("expected OpenSQLite to fail when WAL mode is unavailable")
	}
}

func assertSingleTextResult(t *testing.T, database *Database, query, want string) {
	t.Helper()

	var got string
	if err := database.Conn.QueryRow(query).Scan(&got); err != nil {
		t.Fatalf("query %q failed: %v", query, err)
	}

	if got != want {
		t.Fatalf("unexpected result for %q: got %q want %q", query, got, want)
	}
}

func assertSingleIntResult(t *testing.T, database *Database, query string, want int) {
	t.Helper()

	var got int
	if err := database.Conn.QueryRow(query).Scan(&got); err != nil {
		t.Fatalf("query %q failed: %v", query, err)
	}

	if got != want {
		t.Fatalf("unexpected result for %q: got %d want %d", query, got, want)
	}
}
