package caldav

import (
	"context"
	"net/http"
	"net/http/httptest"
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
