package sync

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"caldo/internal/caldav"
	"caldo/internal/db"
)

func TestRunnerCTagSkipsUnchangedCalendar(t *testing.T) {
	t.Parallel()

	var depthOnePropfinds atomic.Int32
	var getRequests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PROPFIND" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		switch r.Header.Get("Depth") {
		case "0":
			writeCTagResponse(w, `"ctag-1"`)
		case "1":
			depthOnePropfinds.Add(1)
			w.WriteHeader(http.StatusMultiStatus)
		default:
			t.Fatalf("unexpected depth: %s", r.Header.Get("Depth"))
		}
	}))
	t.Cleanup(server.Close)

	database, key := newCTagTestDatabase(t, server.URL)
	seedCTagProject(t, database, `"ctag-1"`, `"etag-keep"`, false)

	runner, err := NewRunner(database, key, caldav.NewTodoClient(server.Client()))
	if err != nil {
		t.Fatalf("new runner: %v", err)
	}
	if err := runner.Run(context.Background()); err != nil {
		t.Fatalf("run: %v", err)
	}

	if got := depthOnePropfinds.Load(); got != 0 {
		t.Fatalf("unchanged ctag should skip etag listing, got %d", got)
	}
	if got := getRequests.Load(); got != 0 {
		t.Fatalf("unchanged ctag should skip get requests, got %d", got)
	}
	assertTaskTitle(t, database, "uid-unchanged", "Keep")
}

func TestRunnerCTagAppliesChangedDeletedAndConflictStates(t *testing.T) {
	t.Parallel()

	rawClean := "BEGIN:VCALENDAR\nBEGIN:VTODO\nUID:uid-clean\nSUMMARY:Remote Clean\nSTATUS:NEEDS-ACTION\nEND:VTODO\nEND:VCALENDAR"
	rawDirty := "BEGIN:VCALENDAR\nBEGIN:VTODO\nUID:uid-dirty\nSUMMARY:Remote Dirty\nSTATUS:NEEDS-ACTION\nEND:VTODO\nEND:VCALENDAR"
	var getRequests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "PROPFIND":
			if r.Header.Get("Depth") == "0" {
				writeCTagResponse(w, `"ctag-2"`)
				return
			}
			if r.Header.Get("Depth") != "1" {
				t.Fatalf("unexpected propfind depth: %s", r.Header.Get("Depth"))
			}
			writeETagResponse(w, []caldav.CalendarObject{
				{Href: "/cal/work/uid-unchanged.ics", ETag: `"etag-keep"`},
				{Href: "/cal/work/uid-clean.ics", ETag: `"etag-new"`},
				{Href: "/cal/work/uid-dirty.ics", ETag: `"etag-remote"`},
			})
		case http.MethodGet:
			getRequests.Add(1)
			switch r.URL.Path {
			case "/cal/work/uid-clean.ics":
				w.Header().Set("ETag", `"etag-new"`)
				_, _ = w.Write([]byte(rawClean))
			case "/cal/work/uid-dirty.ics":
				w.Header().Set("ETag", `"etag-remote"`)
				_, _ = w.Write([]byte(rawDirty))
			default:
				t.Fatalf("unexpected get path: %s", r.URL.Path)
			}
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)

	database, key := newCTagTestDatabase(t, server.URL)
	seedCTagProject(t, database, `"ctag-1"`, `"etag-keep"`, true)

	runner, err := NewRunner(database, key, caldav.NewTodoClient(server.Client()))
	if err != nil {
		t.Fatalf("new runner: %v", err)
	}
	if err := runner.Run(context.Background()); err != nil {
		t.Fatalf("run: %v", err)
	}

	if got := getRequests.Load(); got != 2 {
		t.Fatalf("expected only changed resources to be fetched, got %d", got)
	}
	assertTaskTitle(t, database, "uid-unchanged", "Keep")
	assertTaskTitle(t, database, "uid-clean", "Remote Clean")
	assertSyncSingleInt(t, database, `SELECT COUNT(*) FROM tasks WHERE uid='uid-deleted';`, 0)
	assertSyncSingleText(t, database, `SELECT sync_status FROM tasks WHERE uid='uid-dirty';`, "conflict")
	assertSyncSingleInt(t, database, `SELECT COUNT(*) FROM conflicts WHERE task_id='task-dirty' AND conflict_type='field_conflict' AND remote_vtodo LIKE '%Remote Dirty%';`, 1)
	assertSyncSingleText(t, database, `SELECT ctag FROM projects WHERE id='project-1';`, `"ctag-2"`)
	assertSyncSingleText(t, database, `SELECT sync_strategy FROM projects WHERE id='project-1';`, StrategyCTag)
}

func TestRunnerCTagFallsBackToFullScanWhenETagMissing(t *testing.T) {
	t.Parallel()

	rawVTODO := "BEGIN:VCALENDAR\nBEGIN:VTODO\nUID:uid-fullscan\nSUMMARY:Full Scan Task\nSTATUS:NEEDS-ACTION\nEND:VTODO\nEND:VCALENDAR"
	var fullScanReports atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "PROPFIND":
			if r.Header.Get("Depth") == "0" {
				writeCTagResponse(w, `"ctag-2"`)
				return
			}
			w.Header().Set("Content-Type", "application/xml; charset=utf-8")
			w.WriteHeader(http.StatusMultiStatus)
			_, _ = w.Write([]byte(`<?xml version="1.0"?>
<d:multistatus xmlns:d="DAV:">
  <d:response>
    <d:href>/cal/work/uid-fullscan.ics</d:href>
    <d:propstat><d:prop></d:prop><d:status>HTTP/1.1 200 OK</d:status></d:propstat>
  </d:response>
</d:multistatus>`))
		case "REPORT":
			body := readSyncTestBody(t, r)
			if !strings.Contains(body, "calendar-query") {
				t.Fatalf("fallback should use full-scan calendar-query: %s", body)
			}
			fullScanReports.Add(1)
			w.Header().Set("Content-Type", "application/xml; charset=utf-8")
			w.WriteHeader(http.StatusMultiStatus)
			_, _ = w.Write([]byte(`<d:multistatus xmlns:d="DAV:" xmlns:c="urn:ietf:params:xml:ns:caldav">
<d:response>
  <d:href>/cal/work/uid-fullscan.ics</d:href>
  <d:propstat>
    <d:prop>
      <d:getetag>"etag-fullscan"</d:getetag>
      <c:calendar-data><![CDATA[` + rawVTODO + `]]></c:calendar-data>
    </d:prop>
  </d:propstat>
</d:response>
</d:multistatus>`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)

	database, key := newCTagTestDatabase(t, server.URL)
	if _, err := database.Conn.ExecContext(context.Background(), `
INSERT INTO projects (id, calendar_href, display_name, sync_strategy, ctag, created_at, updated_at)
VALUES ('project-1', '/cal/work/', 'Work', 'ctag', '"ctag-1"', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
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

	if got := fullScanReports.Load(); got != 1 {
		t.Fatalf("expected one full-scan fallback report, got %d", got)
	}
	assertTaskTitle(t, database, "uid-fullscan", "Full Scan Task")
	assertSyncSingleText(t, database, `SELECT sync_strategy FROM projects WHERE id='project-1';`, StrategyFullScan)
}

func newCTagTestDatabase(t *testing.T, serverURL string) (*db.Database, []byte) {
	t.Helper()

	database, err := db.OpenSQLite(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	key := []byte("12345678901234567890123456789012")
	if err := database.SaveCalDAVCredentials(context.Background(), key, db.CalDAVCredentials{URL: serverURL, Username: "alice", Password: "secret"}); err != nil {
		t.Fatalf("save credentials: %v", err)
	}
	return database, key
}

func seedCTagProject(t *testing.T, database *db.Database, ctag string, unchangedETag string, includeChangedRows bool) {
	t.Helper()

	if _, err := database.Conn.ExecContext(context.Background(), `
INSERT INTO projects (id, calendar_href, display_name, sync_strategy, ctag, created_at, updated_at)
VALUES ('project-1', '/cal/work/', 'Work', 'ctag', ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
`, ctag); err != nil {
		t.Fatalf("seed project state: %v", err)
	}

	if _, err := database.Conn.ExecContext(context.Background(), `
INSERT INTO tasks (
	id, project_id, uid, href, etag, server_version, title, status, raw_vtodo, base_vtodo, project_name, sync_status, created_at, updated_at
) VALUES
	('task-unchanged', 'project-1', 'uid-unchanged', '/cal/work/uid-unchanged.ics', ?, 1, 'Keep', 'needs-action', 'BEGIN:VTODO\nUID:uid-unchanged\nSUMMARY:Keep\nEND:VTODO', 'BEGIN:VTODO\nUID:uid-unchanged\nSUMMARY:Keep\nEND:VTODO', 'Work', 'synced', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
`, unchangedETag); err != nil {
		t.Fatalf("seed unchanged project state: %v", err)
	}

	if !includeChangedRows {
		return
	}
	if _, err := database.Conn.ExecContext(context.Background(), `
INSERT INTO tasks (
	id, project_id, uid, href, etag, server_version, title, status, raw_vtodo, base_vtodo, project_name, sync_status, created_at, updated_at
) VALUES
	('task-clean', 'project-1', 'uid-clean', '/cal/work/uid-clean.ics', '"etag-old"', 2, 'Old Clean', 'needs-action', 'BEGIN:VTODO\nUID:uid-clean\nSUMMARY:Old Clean\nEND:VTODO', 'BEGIN:VTODO\nUID:uid-clean\nSUMMARY:Old Clean\nEND:VTODO', 'Work', 'synced', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
	('task-deleted', 'project-1', 'uid-deleted', '/cal/work/uid-deleted.ics', '"etag-del"', 3, 'Deleted', 'needs-action', 'BEGIN:VTODO\nUID:uid-deleted\nSUMMARY:Deleted\nEND:VTODO', 'BEGIN:VTODO\nUID:uid-deleted\nSUMMARY:Deleted\nEND:VTODO', 'Work', 'synced', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
`); err != nil {
		t.Fatalf("seed changed project state: %v", err)
	}

	dirtyLocal := "BEGIN:VCALENDAR\nBEGIN:VTODO\nUID:uid-dirty\nSUMMARY:Local Dirty\nEND:VTODO\nEND:VCALENDAR"
	dirtyBase := "BEGIN:VCALENDAR\nBEGIN:VTODO\nUID:uid-dirty\nSUMMARY:Base Dirty\nEND:VTODO\nEND:VCALENDAR"
	if _, err := database.Conn.ExecContext(context.Background(), `
INSERT INTO tasks (
	id, project_id, uid, href, etag, server_version, title, status, raw_vtodo, base_vtodo, project_name, sync_status, created_at, updated_at
) VALUES (
	'task-dirty', 'project-1', 'uid-dirty', '/cal/work/uid-dirty.ics', '"etag-base"', 4, 'Local Dirty', 'needs-action', ?, ?, 'Work', 'pending', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
);
`, dirtyLocal, dirtyBase); err != nil {
		t.Fatalf("seed dirty project state: %v", err)
	}
}

func writeCTagResponse(w http.ResponseWriter, ctag string) {
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.WriteHeader(http.StatusMultiStatus)
	_, _ = w.Write([]byte(`<?xml version="1.0"?>
<d:multistatus xmlns:d="DAV:" xmlns:cs="http://calendarserver.org/ns/">
  <d:response>
    <d:href>/cal/work/</d:href>
    <d:propstat>
      <d:prop><cs:getctag>` + ctag + `</cs:getctag></d:prop>
      <d:status>HTTP/1.1 200 OK</d:status>
    </d:propstat>
  </d:response>
</d:multistatus>`))
}

func writeETagResponse(w http.ResponseWriter, objects []caldav.CalendarObject) {
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.WriteHeader(http.StatusMultiStatus)
	var body strings.Builder
	body.WriteString(`<?xml version="1.0"?><d:multistatus xmlns:d="DAV:">`)
	body.WriteString(`<d:response><d:href>/cal/work/</d:href><d:propstat><d:prop><d:getetag>"calendar"</d:getetag></d:prop><d:status>HTTP/1.1 200 OK</d:status></d:propstat></d:response>`)
	for _, object := range objects {
		body.WriteString(`<d:response><d:href>`)
		body.WriteString(object.Href)
		body.WriteString(`</d:href><d:propstat><d:prop><d:getetag>`)
		body.WriteString(object.ETag)
		body.WriteString(`</d:getetag></d:prop><d:status>HTTP/1.1 200 OK</d:status></d:propstat></d:response>`)
	}
	body.WriteString(`</d:multistatus>`)
	_, _ = w.Write([]byte(body.String()))
}

func assertTaskTitle(t *testing.T, database *db.Database, uid string, want string) {
	t.Helper()

	var title string
	if err := database.Conn.QueryRowContext(context.Background(), `SELECT title FROM tasks WHERE uid = ?;`, uid).Scan(&title); err != nil {
		t.Fatalf("query task title for %s: %v", uid, err)
	}
	if title != want {
		t.Fatalf("unexpected title for %s: got %q want %q", uid, title, want)
	}
}

func assertSyncSingleText(t *testing.T, database *db.Database, query string, want string) {
	t.Helper()

	var got string
	if err := database.Conn.QueryRowContext(context.Background(), query).Scan(&got); err != nil {
		t.Fatalf("query text result: %v", err)
	}
	if got != want {
		t.Fatalf("unexpected text result: got %q want %q", got, want)
	}
}

func assertSyncSingleInt(t *testing.T, database *db.Database, query string, want int) {
	t.Helper()

	var got int
	if err := database.Conn.QueryRowContext(context.Background(), query).Scan(&got); err != nil {
		t.Fatalf("query int result: %v", err)
	}
	if got != want {
		t.Fatalf("unexpected int result: got %d want %d", got, want)
	}
}
