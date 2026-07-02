package caldav

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestTodoClientListVTODOs(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "REPORT" {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/xml; charset=utf-8")
		w.WriteHeader(http.StatusMultiStatus)
		_, _ = w.Write([]byte(`<?xml version="1.0"?>
<d:multistatus xmlns:d="DAV:" xmlns:c="urn:ietf:params:xml:ns:caldav">
  <d:response>
    <d:href>/cal/work/todo-1.ics</d:href>
    <d:propstat>
      <d:prop>
        <d:getetag>"etag-1"</d:getetag>
        <c:calendar-data>BEGIN:VCALENDAR
BEGIN:VTODO
UID:uid-1
SUMMARY:Task 1
END:VTODO
END:VCALENDAR</c:calendar-data>
      </d:prop>
    </d:propstat>
  </d:response>
</d:multistatus>`))
	}))
	t.Cleanup(server.Close)

	client := NewTodoClient(server.Client())
	items, err := client.ListVTODOs(context.Background(), Credentials{
		URL:      server.URL + "/caldav",
		Username: "alice",
		Password: "secret",
	}, "/cal/work/")
	if err != nil {
		t.Fatalf("list vtodos: %v", err)
	}

	if len(items) != 1 {
		t.Fatalf("unexpected item count: got %d", len(items))
	}
	if items[0].Href != "/cal/work/todo-1.ics" {
		t.Fatalf("unexpected href: %q", items[0].Href)
	}
	if items[0].ETag != "\"etag-1\"" {
		t.Fatalf("unexpected etag: %q", items[0].ETag)
	}
}

func TestTodoClientSyncCollection(t *testing.T) {
	t.Parallel()

	rawVTODO := "BEGIN:VCALENDAR\nBEGIN:VTODO\nUID:uid-1\nSUMMARY:Task 1\nEND:VTODO\nEND:VCALENDAR"
	var sawSyncReport bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "REPORT" {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		body := readRequestBody(t, r)
		if !strings.Contains(body, "sync-collection") || !strings.Contains(body, "token-1") {
			t.Fatalf("unexpected sync report body: %s", body)
		}
		sawSyncReport = true
		w.Header().Set("Content-Type", "application/xml; charset=utf-8")
		w.WriteHeader(http.StatusMultiStatus)
		_, _ = w.Write([]byte(`<?xml version="1.0"?>
<d:multistatus xmlns:d="DAV:" xmlns:c="urn:ietf:params:xml:ns:caldav">
  <d:sync-token>token-2</d:sync-token>
  <d:response>
    <d:href>/cal/work/todo-1.ics</d:href>
    <d:propstat>
      <d:prop>
        <d:getetag>"etag-2"</d:getetag>
        <c:calendar-data>` + rawVTODO + `</c:calendar-data>
      </d:prop>
      <d:status>HTTP/1.1 200 OK</d:status>
    </d:propstat>
  </d:response>
  <d:response>
    <d:href>/cal/work/deleted.ics</d:href>
    <d:status>HTTP/1.1 404 Not Found</d:status>
  </d:response>
</d:multistatus>`))
	}))
	t.Cleanup(server.Close)

	client := NewTodoClient(server.Client())
	result, err := client.SyncCollection(context.Background(), Credentials{
		URL:      server.URL,
		Username: "alice",
		Password: "secret",
	}, "/cal/work/", "token-1")
	if err != nil {
		t.Fatalf("sync collection: %v", err)
	}

	if !sawSyncReport {
		t.Fatal("expected sync collection report")
	}
	if result.SyncToken != "token-2" {
		t.Fatalf("unexpected sync token: %q", result.SyncToken)
	}
	if len(result.Changed) != 1 || result.Changed[0].Href != "/cal/work/todo-1.ics" || result.Changed[0].ETag != `"etag-2"` {
		t.Fatalf("unexpected changed objects: %#v", result.Changed)
	}
	if len(result.DeletedHrefs) != 1 || result.DeletedHrefs[0] != "/cal/work/deleted.ics" {
		t.Fatalf("unexpected deleted hrefs: %#v", result.DeletedHrefs)
	}
}

func TestTodoClientSyncCollectionFetchesMissingCalendarData(t *testing.T) {
	t.Parallel()

	var getRequests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "REPORT":
			w.Header().Set("Content-Type", "application/xml; charset=utf-8")
			w.WriteHeader(http.StatusMultiStatus)
			_, _ = w.Write([]byte(`<?xml version="1.0"?>
<d:multistatus xmlns:d="DAV:">
  <d:sync-token>token-2</d:sync-token>
  <d:response>
    <d:href>/cal/work/todo-1.ics</d:href>
    <d:propstat>
      <d:prop><d:getetag>"etag-2"</d:getetag></d:prop>
      <d:status>HTTP/1.1 200 OK</d:status>
    </d:propstat>
  </d:response>
</d:multistatus>`))
		case http.MethodGet:
			getRequests.Add(1)
			w.Header().Set("ETag", `"etag-2"`)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("BEGIN:VCALENDAR\nBEGIN:VTODO\nUID:uid-1\nSUMMARY:Task 1\nEND:VTODO\nEND:VCALENDAR"))
		default:
			t.Fatalf("unexpected method: %s", r.Method)
		}
	}))
	t.Cleanup(server.Close)

	client := NewTodoClient(server.Client())
	result, err := client.SyncCollection(context.Background(), Credentials{
		URL:      server.URL,
		Username: "alice",
		Password: "secret",
	}, "/cal/work/", "")
	if err != nil {
		t.Fatalf("sync collection: %v", err)
	}
	if got := getRequests.Load(); got != 1 {
		t.Fatalf("unexpected get requests: got %d want 1", got)
	}
	if len(result.Changed) != 1 || !strings.Contains(result.Changed[0].RawVTODO, "UID:uid-1") {
		t.Fatalf("unexpected changed objects: %#v", result.Changed)
	}
}

func TestTodoClientSyncCollectionUnsupported(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotImplemented)
	}))
	t.Cleanup(server.Close)

	client := NewTodoClient(server.Client())
	_, err := client.SyncCollection(context.Background(), Credentials{
		URL:      server.URL,
		Username: "alice",
		Password: "secret",
	}, "/cal/work/", "")
	if !errors.Is(err, ErrSyncCollectionUnsupported) {
		t.Fatalf("expected unsupported error, got %v", err)
	}
}

func TestTodoClientCalendarCTag(t *testing.T) {
	t.Parallel()

	var sawCTagPropfind bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PROPFIND" || r.Header.Get("Depth") != "0" {
			t.Fatalf("unexpected request: %s depth=%s", r.Method, r.Header.Get("Depth"))
		}
		body := readRequestBody(t, r)
		if !strings.Contains(body, "getctag") {
			t.Fatalf("unexpected ctag body: %s", body)
		}
		sawCTagPropfind = true
		w.Header().Set("Content-Type", "application/xml; charset=utf-8")
		w.WriteHeader(http.StatusMultiStatus)
		_, _ = w.Write([]byte(`<?xml version="1.0"?>
<d:multistatus xmlns:d="DAV:" xmlns:cs="http://calendarserver.org/ns/">
  <d:response>
    <d:href>/cal/work/</d:href>
    <d:propstat>
      <d:prop><cs:getctag>"ctag-2"</cs:getctag></d:prop>
      <d:status>HTTP/1.1 200 OK</d:status>
    </d:propstat>
  </d:response>
</d:multistatus>`))
	}))
	t.Cleanup(server.Close)

	client := NewTodoClient(server.Client())
	ctag, err := client.CalendarCTag(context.Background(), Credentials{
		URL:      server.URL,
		Username: "alice",
		Password: "secret",
	}, "/cal/work/")
	if err != nil {
		t.Fatalf("calendar ctag: %v", err)
	}
	if !sawCTagPropfind || ctag != `"ctag-2"` {
		t.Fatalf("unexpected ctag result: saw=%v ctag=%q", sawCTagPropfind, ctag)
	}
}

func TestTodoClientListVTODOETags(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PROPFIND" || r.Header.Get("Depth") != "1" {
			t.Fatalf("unexpected request: %s depth=%s", r.Method, r.Header.Get("Depth"))
		}
		body := readRequestBody(t, r)
		if !strings.Contains(body, "getetag") {
			t.Fatalf("unexpected etag body: %s", body)
		}
		w.Header().Set("Content-Type", "application/xml; charset=utf-8")
		w.WriteHeader(http.StatusMultiStatus)
		_, _ = w.Write([]byte(`<?xml version="1.0"?>
<d:multistatus xmlns:d="DAV:">
  <d:response>
    <d:href>/cal/work/</d:href>
    <d:propstat>
      <d:prop><d:getetag>"calendar-etag"</d:getetag></d:prop>
      <d:status>HTTP/1.1 200 OK</d:status>
    </d:propstat>
  </d:response>
  <d:response>
    <d:href>/cal/work/todo-1.ics</d:href>
    <d:propstat>
      <d:prop><d:getetag>"etag-1"</d:getetag></d:prop>
      <d:status>HTTP/1.1 200 OK</d:status>
    </d:propstat>
  </d:response>
</d:multistatus>`))
	}))
	t.Cleanup(server.Close)

	client := NewTodoClient(server.Client())
	objects, err := client.ListVTODOETags(context.Background(), Credentials{
		URL:      server.URL,
		Username: "alice",
		Password: "secret",
	}, "/cal/work/")
	if err != nil {
		t.Fatalf("list vtodo etags: %v", err)
	}
	if len(objects) != 1 || objects[0].Href != "/cal/work/todo-1.ics" || objects[0].ETag != `"etag-1"` {
		t.Fatalf("unexpected etag objects: %#v", objects)
	}
}

func TestTodoClientListVTODOETagsFallbackWhenETagMissing(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml; charset=utf-8")
		w.WriteHeader(http.StatusMultiStatus)
		_, _ = w.Write([]byte(`<?xml version="1.0"?>
<d:multistatus xmlns:d="DAV:">
  <d:response>
    <d:href>/cal/work/todo-1.ics</d:href>
    <d:propstat>
      <d:prop></d:prop>
      <d:status>HTTP/1.1 200 OK</d:status>
    </d:propstat>
  </d:response>
</d:multistatus>`))
	}))
	t.Cleanup(server.Close)

	client := NewTodoClient(server.Client())
	_, err := client.ListVTODOETags(context.Background(), Credentials{
		URL:      server.URL,
		Username: "alice",
		Password: "secret",
	}, "/cal/work/")
	if !errors.Is(err, ErrCTagUnsupported) {
		t.Fatalf("expected ctag fallback error, got %v", err)
	}
}

func TestTodoClientPutVTODOCreateDoesNotRetry(t *testing.T) {
	t.Parallel()

	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	t.Cleanup(server.Close)

	client := NewTodoClient(server.Client())
	_, err := client.PutVTODOCreate(context.Background(), Credentials{
		URL:      server.URL,
		Username: "alice",
		Password: "secret",
	}, "/cal/work/todo-1.ics", "BEGIN:VCALENDAR\nEND:VCALENDAR")
	if err == nil {
		t.Fatal("expected error")
	}
	if got := requests.Load(); got != 1 {
		t.Fatalf("unexpected retries for create: got %d want %d", got, 1)
	}
}

func readRequestBody(t *testing.T, r *http.Request) string {
	t.Helper()
	defer r.Body.Close()
	body := new(strings.Builder)
	if _, err := io.Copy(body, r.Body); err != nil {
		t.Fatalf("read request body: %v", err)
	}
	return body.String()
}

func TestTodoClientPutVTODOUpdateRetriesWithIfMatch(t *testing.T) {
	t.Parallel()

	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		current := requests.Add(1)
		if got := r.Header.Get("If-Match"); got != "\"etag-1\"" {
			t.Fatalf("missing if-match header: %q", got)
		}
		if current < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("ETag", "\"etag-2\"")
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(server.Close)

	client := NewTodoClient(server.Client())
	etag, err := client.PutVTODOUpdate(context.Background(), Credentials{
		URL:      server.URL,
		Username: "alice",
		Password: "secret",
	}, "/cal/work/todo-1.ics", "BEGIN:VCALENDAR\nEND:VCALENDAR", "\"etag-1\"")
	if err != nil {
		t.Fatalf("put update: %v", err)
	}
	if etag != "\"etag-2\"" {
		t.Fatalf("unexpected etag: %q", etag)
	}
	if got := requests.Load(); got != 3 {
		t.Fatalf("unexpected retries for update: got %d want %d", got, 3)
	}
}

func TestTodoClientPutVTODOUpdateReturnsPreconditionFailure(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusPreconditionFailed)
	}))
	t.Cleanup(server.Close)

	client := NewTodoClient(server.Client())
	_, err := client.PutVTODOUpdate(context.Background(), Credentials{
		URL:      server.URL,
		Username: "alice",
		Password: "secret",
	}, "/cal/work/todo-1.ics", "BEGIN:VCALENDAR\nEND:VCALENDAR", "\"etag-1\"")
	if !errors.Is(err, ErrPreconditionFailed) {
		t.Fatalf("expected precondition failure, got %v", err)
	}
}

func TestTodoClientDeleteVTODOTreatsNotFoundAsSuccess(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(server.Close)

	client := NewTodoClient(server.Client())
	if err := client.DeleteVTODO(context.Background(), Credentials{
		URL:      server.URL,
		Username: "alice",
		Password: "secret",
	}, "/cal/work/todo-1.ics", "\"etag-1\""); err != nil {
		t.Fatalf("delete vtodo: %v", err)
	}
}

func TestTodoClientGetVTODORetriesIdempotentOperation(t *testing.T) {
	t.Parallel()

	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		current := attempts.Add(1)
		if current < 2 {
			hj, ok := w.(http.Hijacker)
			if !ok {
				t.Fatal("hijacker not supported")
			}
			conn, _, err := hj.Hijack()
			if err != nil {
				t.Fatalf("hijack: %v", err)
			}
			_ = conn.Close()
			return
		}
		w.Header().Set("ETag", "\"etag-1\"")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("BEGIN:VCALENDAR\nEND:VCALENDAR"))
	}))
	t.Cleanup(server.Close)

	client := NewTodoClient(server.Client())
	raw, etag, err := client.GetVTODO(context.Background(), Credentials{
		URL:      server.URL,
		Username: "alice",
		Password: "secret",
	}, "/cal/work/todo-1.ics")
	if err != nil {
		t.Fatalf("get vtodo: %v", err)
	}
	if raw == "" || etag != "\"etag-1\"" {
		t.Fatalf("unexpected response: raw=%q etag=%q", raw, etag)
	}
	if got := attempts.Load(); got != 2 {
		t.Fatalf("unexpected attempts: got %d want %d", got, 2)
	}
}
