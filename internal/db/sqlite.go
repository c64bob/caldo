package db

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

	"caldo/internal/migrations"
	_ "modernc.org/sqlite"
)

const (
	sqliteDriverName = "sqlite"
	busyTimeoutMs    = 5000
)

// Database wraps the SQLite handle and the global write mutex.
type Database struct {
	Conn    *sql.DB
	WriteMu sync.Mutex
}

// OpenSQLite opens the SQLite database, configures required PRAGMAs, and runs migrations.
func OpenSQLite(path string) (*Database, error) {
	database, err := OpenSQLiteConnection(path)
	if err != nil {
		return nil, err
	}

	if err := database.RunMigrations(context.Background(), path); err != nil {
		_ = database.Close()
		return nil, err
	}

	return database, nil
}

// OpenSQLiteConnection opens the SQLite database and configures required PRAGMAs without running migrations.
func OpenSQLiteConnection(path string) (*Database, error) {
	dbConn, err := sql.Open(sqliteDriverName, path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite database: %w", err)
	}

	dbConn.SetMaxOpenConns(1)

	if _, err := dbConn.Exec("PRAGMA journal_mode = WAL;"); err != nil {
		_ = dbConn.Close()
		return nil, fmt.Errorf("set pragma journal_mode: %w", err)
	}
	var journalMode string
	if err := dbConn.QueryRow("PRAGMA journal_mode;").Scan(&journalMode); err != nil {
		_ = dbConn.Close()
		return nil, fmt.Errorf("read pragma journal_mode: %w", err)
	}
	if journalMode != "wal" {
		_ = dbConn.Close()
		return nil, fmt.Errorf("unexpected pragma journal_mode: got %q want %q", journalMode, "wal")
	}

	if _, err := dbConn.Exec("PRAGMA synchronous = NORMAL;"); err != nil {
		_ = dbConn.Close()
		return nil, fmt.Errorf("set pragma synchronous: %w", err)
	}

	if _, err := dbConn.Exec(fmt.Sprintf("PRAGMA busy_timeout = %d;", busyTimeoutMs)); err != nil {
		_ = dbConn.Close()
		return nil, fmt.Errorf("set pragma busy_timeout: %w", err)
	}

	return &Database{Conn: dbConn}, nil
}

// RunMigrations applies pending embedded SQLite migrations and enables foreign key checks.
func (d *Database) RunMigrations(ctx context.Context, path string) error {
	if d == nil || d.Conn == nil {
		return fmt.Errorf("run migrations: database is not open")
	}

	if err := migrations.Run(ctx, d.Conn, path); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}

	if _, err := d.Conn.ExecContext(ctx, "PRAGMA foreign_keys = ON;"); err != nil {
		return fmt.Errorf("set pragma foreign_keys: %w", err)
	}

	return nil
}

// Close closes the wrapped SQLite database connection.
func (d *Database) Close() error {
	if d == nil || d.Conn == nil {
		return nil
	}

	if err := d.Conn.Close(); err != nil {
		return fmt.Errorf("close sqlite database: %w", err)
	}

	d.Conn = nil
	return nil
}
