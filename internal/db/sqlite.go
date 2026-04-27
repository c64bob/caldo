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

// OpenSQLite opens the SQLite database and configures required PRAGMAs.
func OpenSQLite(path string) (*Database, error) {
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

	if err := migrations.Run(context.Background(), dbConn, path); err != nil {
		_ = dbConn.Close()
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	if _, err := dbConn.Exec("PRAGMA foreign_keys = ON;"); err != nil {
		_ = dbConn.Close()
		return nil, fmt.Errorf("set pragma foreign_keys: %w", err)
	}

	return &Database{Conn: dbConn}, nil
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
