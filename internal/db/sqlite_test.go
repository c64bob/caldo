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
