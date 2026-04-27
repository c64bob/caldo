package caldav_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"caldo/internal/caldav"
	"caldo/internal/db"
)

func TestConnectionProbeAndCapabilityPersistence(t *testing.T) {
	t.Parallel()

	database, err := db.OpenSQLite(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() {
		_ = database.Close()
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("DAV", "1, sync-collection")
		w.WriteHeader(http.StatusMultiStatus)
		_, _ = w.Write([]byte(`<d:multistatus xmlns:d="DAV:" xmlns:cs="http://calendarserver.org/ns/"><d:getetag>\"etag\"</d:getetag><cs:getctag>ctag</cs:getctag></d:multistatus>`))
	}))
	defer server.Close()

	tester := caldav.NewConnectionTester(server.Client())
	capabilities, err := tester.TestConnection(context.Background(), caldav.Credentials{
		URL:      server.URL,
		Username: "alice",
		Password: "secret",
	})
	if err != nil {
		t.Fatalf("test connection: %v", err)
	}

	if err := database.SaveCalDAVServerCapabilities(context.Background(), db.CalDAVServerCapabilities{
		WebDAVSync: capabilities.WebDAVSync,
		CTag:       capabilities.CTag,
		ETag:       capabilities.ETag,
		FullScan:   capabilities.FullScan,
	}); err != nil {
		t.Fatalf("save capabilities: %v", err)
	}

	var rawPayload string
	if err := database.Conn.QueryRowContext(context.Background(), `SELECT caldav_server_capabilities FROM settings WHERE id='default';`).Scan(&rawPayload); err != nil {
		t.Fatalf("query capabilities: %v", err)
	}

	var persisted map[string]bool
	if err := json.Unmarshal([]byte(rawPayload), &persisted); err != nil {
		t.Fatalf("unmarshal persisted capabilities: %v", err)
	}

	if !persisted["webdav_sync"] || !persisted["ctag"] || !persisted["etag"] || !persisted["fullscan"] {
		t.Fatalf("unexpected persisted capabilities: %#v", persisted)
	}
}
