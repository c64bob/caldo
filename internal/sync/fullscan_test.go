package sync

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"caldo/internal/caldav"
	"caldo/internal/db"
)

func TestRunnerUsesWebDAVSyncWhenSupported(t *testing.T) {
	t.Parallel()

	const rawVTODO = "BEGIN:VCALENDAR\nBEGIN:VTODO\nUID:uid-1\nSUMMARY:Remote Task\nSTATUS:NEEDS-ACTION\nEND:VTODO\nEND:VCALENDAR"
	var sawWebDAVSync bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "REPORT" || r.URL.Path != "/cal/work/" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		body := readSyncTestBody(t, r)
		if !strings.Contains(body, "sync-collection") {
			t.Fatalf("runner should not use full calendar-query when webdav sync succeeds: %s", body)
		}
		sawWebDAVSync = true
		w.Header().Set("Content-Type", "application/xml; charset=utf-8")
		w.WriteHeader(http.StatusMultiStatus)
		_, _ = w.Write([]byte(`<d:multistatus xmlns:d="DAV:" xmlns:c="urn:ietf:params:xml:ns:caldav">
<d:sync-token>token-2</d:sync-token>
<d:response>
  <d:href>/cal/work/uid-1.ics</d:href>
  <d:propstat>
    <d:prop>
      <d:getetag>"etag-1"</d:getetag>
      <c:calendar-data><![CDATA[` + rawVTODO + `]]></c:calendar-data>
    </d:prop>
    <d:status>HTTP/1.1 200 OK</d:status>
  </d:propstat>
</d:response>
</d:multistatus>`))
	}))
	t.Cleanup(server.Close)

	database, err := db.OpenSQLite(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	key := []byte("12345678901234567890123456789012")
	if err := database.SaveCalDAVCredentials(context.Background(), key, db.CalDAVCredentials{URL: server.URL, Username: "alice", Password: "secret"}); err != nil {
		t.Fatalf("save credentials: %v", err)
	}
	if _, err := database.Conn.ExecContext(context.Background(), `
INSERT INTO projects (id, calendar_href, display_name, sync_strategy, sync_token, created_at, updated_at)
VALUES ('project-1', '/cal/work/', 'Work', 'webdav_sync', 'token-1', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
`); err != nil {
		t.Fatalf("seed project: %v", err)
	}

	runner, err := NewRunner(database, key, caldav.NewTodoClient(server.Client()))
	if err != nil {
		t.Fatalf("new runner: %v", err)
	}
	if err := runner.Run(context.Background()); err != nil {
		t.Fatalf("run: %v", err)
	}

	if !sawWebDAVSync {
		t.Fatal("expected webdav sync report")
	}
	var title, strategy, syncToken string
	if err := database.Conn.QueryRowContext(context.Background(), `SELECT title FROM tasks WHERE uid='uid-1';`).Scan(&title); err != nil {
		t.Fatalf("query imported task: %v", err)
	}
	if title != "Remote Task" {
		t.Fatalf("unexpected imported title: %q", title)
	}
	if err := database.Conn.QueryRowContext(context.Background(), `SELECT sync_strategy, sync_token FROM projects WHERE id='project-1';`).Scan(&strategy, &syncToken); err != nil {
		t.Fatalf("query project metadata: %v", err)
	}
	if strategy != StrategyWebDAVSync || syncToken != "token-2" {
		t.Fatalf("unexpected metadata: strategy=%q sync_token=%q", strategy, syncToken)
	}
}

func TestRunnerFallsBackToFullScanAgainstFakeCalDAVServer(t *testing.T) {
	t.Parallel()

	const rawVTODO = "BEGIN:VCALENDAR\nBEGIN:VTODO\nUID:uid-1\nSUMMARY:Remote Task\nSTATUS:NEEDS-ACTION\nCATEGORIES:home,STARRED\nEND:VTODO\nEND:VCALENDAR"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "REPORT" || r.URL.Path != "/cal/work/" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/xml; charset=utf-8")
		w.WriteHeader(http.StatusMultiStatus)
		_, _ = w.Write([]byte(`<d:multistatus xmlns:d="DAV:" xmlns:c="urn:ietf:params:xml:ns:caldav">
<d:response>
  <d:href>/cal/work/uid-1.ics</d:href>
  <d:propstat>
    <d:prop>
      <d:getetag>"etag-1"</d:getetag>
      <c:calendar-data><![CDATA[` + rawVTODO + `]]></c:calendar-data>
    </d:prop>
  </d:propstat>
</d:response>
</d:multistatus>`))
	}))
	t.Cleanup(server.Close)

	database, err := db.OpenSQLite(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	key := []byte("12345678901234567890123456789012")
	if err := database.SaveCalDAVCredentials(context.Background(), key, db.CalDAVCredentials{URL: server.URL, Username: "alice", Password: "secret"}); err != nil {
		t.Fatalf("save credentials: %v", err)
	}
	if _, err := database.Conn.ExecContext(context.Background(), `
INSERT INTO projects (id, calendar_href, display_name, sync_strategy, created_at, updated_at)
VALUES ('project-1', '/cal/work/', 'Work', 'webdav_sync', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
`); err != nil {
		t.Fatalf("seed project: %v", err)
	}

	runner, err := NewRunner(database, key, caldav.NewTodoClient(server.Client()))
	if err != nil {
		t.Fatalf("new runner: %v", err)
	}
	if err := runner.Run(context.Background()); err != nil {
		t.Fatalf("run: %v", err)
	}

	var title, labelNames, strategy string
	if err := database.Conn.QueryRowContext(context.Background(), `SELECT title, label_names FROM tasks WHERE uid='uid-1';`).Scan(&title, &labelNames); err != nil {
		t.Fatalf("query imported task: %v", err)
	}
	if title != "Remote Task" || labelNames != "home STARRED" {
		t.Fatalf("unexpected imported task: title=%q label_names=%q", title, labelNames)
	}
	if err := database.Conn.QueryRowContext(context.Background(), `SELECT sync_strategy FROM projects WHERE id='project-1';`).Scan(&strategy); err != nil {
		t.Fatalf("query strategy: %v", err)
	}
	if strategy != StrategyFullScan {
		t.Fatalf("unexpected strategy: got %q want %q", strategy, StrategyFullScan)
	}
}

func readSyncTestBody(t *testing.T, r *http.Request) string {
	t.Helper()
	defer r.Body.Close()
	body := new(strings.Builder)
	if _, err := io.Copy(body, r.Body); err != nil {
		t.Fatalf("read request body: %v", err)
	}
	return body.String()
}
