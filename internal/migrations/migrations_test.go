package migrations

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
	"time"

	_ "modernc.org/sqlite"
)

func TestRunFromFSAppliesPendingMigrationsAndCreatesBackup(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "caldo.db")
	db := openTestSQLite(t, dbPath)
	defer func() { _ = db.Close() }()

	testFS := fstest.MapFS{
		"sql/0001_create_items.sql": {Data: []byte(`CREATE TABLE items (id INTEGER PRIMARY KEY, name TEXT NOT NULL);`)},
	}

	if err := RunFromFS(context.Background(), db, dbPath, testFS); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	assertTableExists(t, db, "items")
	assertBackupCreated(t, dbPath)

	var version int
	var name string
	var checksum string
	if err := db.QueryRow(`SELECT version, name, checksum FROM schema_migrations`).Scan(&version, &name, &checksum); err != nil {
		t.Fatalf("read schema_migrations: %v", err)
	}
	if version != 1 {
		t.Fatalf("unexpected migration version: got %d want 1", version)
	}
	if name != "create_items" {
		t.Fatalf("unexpected migration name: got %q", name)
	}
	if checksum == "" {
		t.Fatal("expected migration checksum to be stored")
	}
}

func TestRunFromFSFailsOnChecksumMismatch(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "caldo.db")
	db := openTestSQLite(t, dbPath)
	defer func() { _ = db.Close() }()

	initialFS := fstest.MapFS{
		"sql/0001_create_items.sql": {Data: []byte(`CREATE TABLE items (id INTEGER PRIMARY KEY);`)},
	}
	if err := RunFromFS(context.Background(), db, dbPath, initialFS); err != nil {
		t.Fatalf("run initial migrations: %v", err)
	}

	changedFS := fstest.MapFS{
		"sql/0001_create_items.sql": {Data: []byte(`CREATE TABLE items (id INTEGER PRIMARY KEY, changed TEXT);`)},
	}
	if err := RunFromFS(context.Background(), db, dbPath, changedFS); err == nil {
		t.Fatal("expected checksum mismatch error")
	}
}

func TestRunFromFSRollsBackFailedMigration(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "caldo.db")
	db := openTestSQLite(t, dbPath)
	defer func() { _ = db.Close() }()

	testFS := fstest.MapFS{
		"sql/0001_create_ok.sql": {Data: []byte(`CREATE TABLE ok_table (id INTEGER PRIMARY KEY);`)},
		"sql/0002_fail.sql":      {Data: []byte(`CREATE TABL bad_sql;`)},
	}

	if err := RunFromFS(context.Background(), db, dbPath, testFS); err == nil {
		t.Fatal("expected migration failure")
	}

	assertTableExists(t, db, "ok_table")

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE version = 2`).Scan(&count); err != nil {
		t.Fatalf("read failed migration row: %v", err)
	}
	if count != 0 {
		t.Fatalf("failed migration should not be recorded, got count %d", count)
	}
}

func TestBackupSQLiteSkipsExistingBackupPath(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "caldo.db")
	db := openTestSQLite(t, dbPath)
	defer func() { _ = db.Close() }()

	originalNowUTC := nowUTC
	originalBackupRandomBytes := backupRandomBytes
	t.Cleanup(func() {
		nowUTC = originalNowUTC
		backupRandomBytes = originalBackupRandomBytes
	})

	fixedNow := time.Date(2026, time.April, 27, 12, 0, 0, 123456789, time.UTC)
	nowUTC = func() time.Time { return fixedNow }
	backupRandomBytes = func(buf []byte) error {
		for i := range buf {
			buf[i] = 0
		}
		return nil
	}

	existingBackupPath := fmt.Sprintf("%s.backup-%s-%09d-%s-%02d", dbPath, fixedNow.Format("20060102T150405Z"), fixedNow.Nanosecond(), "00000000", 0)
	if err := os.WriteFile(existingBackupPath, []byte("preexisting backup"), 0o600); err != nil {
		t.Fatalf("seed existing backup: %v", err)
	}

	if err := backupSQLite(context.Background(), db, dbPath); err != nil {
		t.Fatalf("backup sqlite: %v", err)
	}

	newBackupPath := fmt.Sprintf("%s.backup-%s-%09d-%s-%02d", dbPath, fixedNow.Format("20060102T150405Z"), fixedNow.Nanosecond(), "00000000", 1)
	if _, err := os.Stat(newBackupPath); err != nil {
		t.Fatalf("expected retried backup path to exist: %v", err)
	}
}

func openTestSQLite(t *testing.T, path string) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if _, err := db.Exec(`PRAGMA journal_mode = WAL;`); err != nil {
		t.Fatalf("set journal_mode wal: %v", err)
	}
	return db
}

func assertTableExists(t *testing.T, db *sql.DB, table string) {
	t.Helper()

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?`, table).Scan(&count); err != nil {
		t.Fatalf("query sqlite_master: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected table %q to exist", table)
	}
}

func assertBackupCreated(t *testing.T, dbPath string) {
	t.Helper()

	matches, err := filepath.Glob(dbPath + ".backup-*")
	if err != nil {
		t.Fatalf("glob backup files: %v", err)
	}
	if len(matches) == 0 {
		t.Fatalf("expected backup file for %s", dbPath)
	}
}
