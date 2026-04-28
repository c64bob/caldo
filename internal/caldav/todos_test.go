package caldav

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
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
