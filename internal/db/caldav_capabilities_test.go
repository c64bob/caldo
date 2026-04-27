package db

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
)

func TestSaveCalDAVServerCapabilitiesPersistsJSON(t *testing.T) {
	t.Parallel()

	database := openSQLiteForCapabilitiesTest(t)
	t.Cleanup(func() {
		_ = database.Close()
	})

	capabilities := CalDAVServerCapabilities{
		WebDAVSync: true,
		CTag:       true,
		ETag:       false,
		FullScan:   true,
	}
	if err := database.SaveCalDAVServerCapabilities(context.Background(), capabilities); err != nil {
		t.Fatalf("save caldav server capabilities: %v", err)
	}

	var rawPayload string
	if err := database.Conn.QueryRowContext(context.Background(), `SELECT caldav_server_capabilities FROM settings WHERE id = 'default';`).Scan(&rawPayload); err != nil {
		t.Fatalf("query capabilities: %v", err)
	}

	var got CalDAVServerCapabilities
	if err := json.Unmarshal([]byte(rawPayload), &got); err != nil {
		t.Fatalf("unmarshal capabilities: %v", err)
	}

	if got != capabilities {
		t.Fatalf("capabilities mismatch: got %#v want %#v", got, capabilities)
	}
}

func openSQLiteForCapabilitiesTest(t *testing.T) *Database {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "caldo.db")
	database, err := OpenSQLite(dbPath)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	return database
}
