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
	assertSingleIntResult(t, database, `SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='projects';`, 1)
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

func TestSettingsSingletonRejectsNullID(t *testing.T) {
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

	if _, err := database.Conn.Exec(`INSERT INTO settings (id, updated_at) VALUES (NULL, CURRENT_TIMESTAMP);`); err == nil {
		t.Fatal("expected NULL settings id insert to fail")
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

func TestProjectsSyncStrategyConstraint(t *testing.T) {
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

	for _, strategy := range []string{"webdav_sync", "ctag", "fullscan"} {
		if _, err := database.Conn.Exec(`
INSERT INTO projects (
    id, calendar_href, display_name, ctag, sync_token, sync_strategy, server_version, is_default, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
`, "project-"+strategy, "/calendars/"+strategy, "Project "+strategy, "ctag-1", "sync-token-1", strategy, 1, false); err != nil {
			t.Fatalf("insert project with sync_strategy %q: %v", strategy, err)
		}
	}

	if _, err := database.Conn.Exec(`
INSERT INTO projects (
    id, calendar_href, display_name, sync_strategy, created_at, updated_at
) VALUES ('project-invalid', '/calendars/invalid', 'Project invalid', 'invalid', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
`); err == nil {
		t.Fatal("expected invalid sync_strategy insert to fail")
	}
}

func TestProjectsCalendarHrefUniqueConstraint(t *testing.T) {
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

	if _, err := database.Conn.Exec(`
INSERT INTO projects (
    id, calendar_href, display_name, sync_strategy, created_at, updated_at
) VALUES ('project-1', '/calendars/duplicate', 'Project One', 'fullscan', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
`); err != nil {
		t.Fatalf("insert first project: %v", err)
	}

	if _, err := database.Conn.Exec(`
INSERT INTO projects (
    id, calendar_href, display_name, sync_strategy, created_at, updated_at
) VALUES ('project-2', '/calendars/duplicate', 'Project Two', 'fullscan', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
`); err == nil {
		t.Fatal("expected duplicate calendar_href insert to fail")
	}
}

func TestProjectsOptimisticLockingByServerVersion(t *testing.T) {
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

	if _, err := database.Conn.Exec(`
INSERT INTO projects (
    id, calendar_href, display_name, sync_strategy, server_version, is_default, created_at, updated_at
) VALUES ('project-1', '/calendars/p1', 'Initial name', 'fullscan', 1, TRUE, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
`); err != nil {
		t.Fatalf("insert project: %v", err)
	}

	result, err := database.Conn.Exec(`
UPDATE projects
SET display_name = ?, server_version = server_version + 1, updated_at = CURRENT_TIMESTAMP
WHERE id = ? AND server_version = ?
`, "Renamed project", "project-1", 1)
	if err != nil {
		t.Fatalf("update project with expected version: %v", err)
	}
	assertRowsAffected(t, result, 1)

	result, err = database.Conn.Exec(`
UPDATE projects
SET display_name = ?, server_version = server_version + 1, updated_at = CURRENT_TIMESTAMP
WHERE id = ? AND server_version = ?
`, "Should fail", "project-1", 1)
	if err != nil {
		t.Fatalf("update project with stale expected version: %v", err)
	}
	assertRowsAffected(t, result, 0)

	var (
		displayName   string
		serverVersion int
		isDefault     bool
	)
	err = database.Conn.QueryRow(`
SELECT display_name, server_version, is_default
FROM projects
WHERE id = 'project-1'
`).Scan(&displayName, &serverVersion, &isDefault)
	if err != nil {
		t.Fatalf("query project: %v", err)
	}

	if displayName != "Renamed project" {
		t.Fatalf("unexpected display_name: got %q want %q", displayName, "Renamed project")
	}
	if serverVersion != 2 {
		t.Fatalf("unexpected server_version: got %d want %d", serverVersion, 2)
	}
	if !isDefault {
		t.Fatal("expected is_default to remain true")
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

func assertRowsAffected(t *testing.T, result interface{ RowsAffected() (int64, error) }, want int64) {
	t.Helper()

	got, err := result.RowsAffected()
	if err != nil {
		t.Fatalf("rows affected failed: %v", err)
	}

	if got != want {
		t.Fatalf("unexpected rows affected: got %d want %d", got, want)
	}
}
