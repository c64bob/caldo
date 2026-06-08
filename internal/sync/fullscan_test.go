package sync

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"caldo/internal/caldav"
	"caldo/internal/db"
)

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
