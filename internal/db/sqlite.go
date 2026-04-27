package db

import (
	"database/sql"
	"fmt"
	"sync"

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
