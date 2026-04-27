package migrations

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"embed"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

//go:embed sql/*.sql
var embeddedMigrations embed.FS

var migrationNamePattern = regexp.MustCompile(`^(\d+)_(.+)\.sql$`)
var nowUTC = func() time.Time { return time.Now().UTC() }
var backupRandomBytes = func(buf []byte) error {
	_, err := rand.Read(buf)
	return err
}

// Migration describes a single schema migration.
type Migration struct {
	Version  int
	Name     string
	Checksum string
	SQL      string
}

// Run applies pending embedded migrations and validates checksums of previously applied migrations.
func Run(ctx context.Context, db *sql.DB, dbPath string) error {
	return RunFromFS(ctx, db, dbPath, embeddedMigrations)
}

// RunFromFS applies pending migrations from the provided filesystem.
func RunFromFS(ctx context.Context, db *sql.DB, dbPath string, migrationFS fs.FS) error {
	migrations, err := LoadFromFS(migrationFS)
	if err != nil {
		return fmt.Errorf("load migrations: %w", err)
	}

	if err := ensureMigrationsTable(ctx, db); err != nil {
		return fmt.Errorf("ensure migrations table: %w", err)
	}

	applied, err := loadAppliedMigrations(ctx, db)
	if err != nil {
		return fmt.Errorf("load applied migrations: %w", err)
	}

	pending, err := validateAndCollectPending(migrations, applied)
	if err != nil {
		return err
	}

	if len(pending) == 0 {
		return nil
	}

	if err := backupSQLite(ctx, db, dbPath); err != nil {
		return fmt.Errorf("backup database before migrations: %w", err)
	}

	for _, migration := range pending {
		if err := applyMigration(ctx, db, migration); err != nil {
			return fmt.Errorf("apply migration %d_%s: %w", migration.Version, migration.Name, err)
		}
	}

	return nil
}

// LoadFromFS loads and validates migration files from an embedded filesystem.
func LoadFromFS(migrationFS fs.FS) ([]Migration, error) {
	entries, err := fs.ReadDir(migrationFS, "sql")
	if err != nil {
		return nil, fmt.Errorf("read migrations directory: %w", err)
	}

	migrations := make([]Migration, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		parts := migrationNamePattern.FindStringSubmatch(entry.Name())
		if len(parts) != 3 {
			return nil, fmt.Errorf("invalid migration file name: %s", entry.Name())
		}

		version, err := strconv.Atoi(parts[1])
		if err != nil {
			return nil, fmt.Errorf("parse migration version %q: %w", parts[1], err)
		}

		sqlBytes, err := fs.ReadFile(migrationFS, filepath.ToSlash(filepath.Join("sql", entry.Name())))
		if err != nil {
			return nil, fmt.Errorf("read migration file %s: %w", entry.Name(), err)
		}

		sqlText := string(sqlBytes)
		checksum := sha256.Sum256(sqlBytes)
		migrations = append(migrations, Migration{
			Version:  version,
			Name:     parts[2],
			Checksum: fmt.Sprintf("%x", checksum[:]),
			SQL:      sqlText,
		})
	}

	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	for i := 1; i < len(migrations); i++ {
		if migrations[i-1].Version == migrations[i].Version {
			return nil, fmt.Errorf("duplicate migration version: %d", migrations[i].Version)
		}
	}

	return migrations, nil
}

type appliedMigration struct {
	Version  int
	Checksum string
}

func ensureMigrationsTable(ctx context.Context, db *sql.DB) error {
	const ddl = `
CREATE TABLE IF NOT EXISTS schema_migrations (
    version INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    applied_at TEXT NOT NULL,
    checksum TEXT NOT NULL
);`
	if _, err := db.ExecContext(ctx, ddl); err != nil {
		return err
	}
	return nil
}

func loadAppliedMigrations(ctx context.Context, db *sql.DB) (map[int]appliedMigration, error) {
	rows, err := db.QueryContext(ctx, `SELECT version, checksum FROM schema_migrations`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	result := make(map[int]appliedMigration)
	for rows.Next() {
		var migration appliedMigration
		if err := rows.Scan(&migration.Version, &migration.Checksum); err != nil {
			return nil, err
		}
		result[migration.Version] = migration
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

func validateAndCollectPending(migrations []Migration, applied map[int]appliedMigration) ([]Migration, error) {
	pending := make([]Migration, 0)

	for _, migration := range migrations {
		appliedMigration, ok := applied[migration.Version]
		if !ok {
			pending = append(pending, migration)
			continue
		}

		if appliedMigration.Checksum != migration.Checksum {
			return nil, fmt.Errorf("migration checksum mismatch for version %d", migration.Version)
		}
	}

	for version := range applied {
		found := false
		for _, migration := range migrations {
			if migration.Version == version {
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("applied migration version %d missing from embedded migrations", version)
		}
	}

	return pending, nil
}

func backupSQLite(ctx context.Context, db *sql.DB, dbPath string) error {
	if dbPath == "" || dbPath == ":memory:" || strings.HasPrefix(dbPath, "file::memory:") {
		return errors.New("backup requires file-backed sqlite path")
	}

	for attempt := 0; attempt < 10; attempt++ {
		backupPath, err := generateBackupPath(dbPath, attempt)
		if err != nil {
			return err
		}
		if _, err := os.Stat(backupPath); err == nil {
			continue
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("check backup path: %w", err)
		}

		if _, err := db.ExecContext(ctx, fmt.Sprintf("VACUUM INTO %s", quoteSQLiteString(backupPath))); err != nil {
			return err
		}
		return nil
	}

	return errors.New("exhausted unique backup path attempts")
}

func generateBackupPath(dbPath string, attempt int) (string, error) {
	now := nowUTC()
	randomBytes := make([]byte, 4)
	if err := backupRandomBytes(randomBytes); err != nil {
		return "", fmt.Errorf("generate backup suffix: %w", err)
	}

	return fmt.Sprintf(
		"%s.backup-%s-%09d-%s-%02d",
		dbPath,
		now.Format("20060102T150405Z"),
		now.Nanosecond(),
		hex.EncodeToString(randomBytes),
		attempt,
	), nil
}

func quoteSQLiteString(value string) string {
	replacer := strings.NewReplacer("'", "''")
	return fmt.Sprintf("'%s'", replacer.Replace(value))
}

func applyMigration(ctx context.Context, db *sql.DB, migration Migration) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, migration.SQL); err != nil {
		_ = tx.Rollback()
		return err
	}

	if _, err := tx.ExecContext(ctx, `
INSERT INTO schema_migrations (version, name, applied_at, checksum)
VALUES (?, ?, ?, ?)
`, migration.Version, migration.Name, time.Now().UTC().Format(time.RFC3339Nano), migration.Checksum); err != nil {
		_ = tx.Rollback()
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}
