package caldav

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSyncCollection_WithoutTokenUsesETagFallbackMode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "REPORT" {
			t.Fatalf("expected REPORT, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/xml; charset=utf-8")
		w.WriteHeader(http.StatusMultiStatus)
		_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<multistatus xmlns="DAV:" xmlns:c="urn:ietf:params:xml:ns:caldav">
  <response>
    <href>/tasks/a.ics</href>
    <propstat>
      <prop>
        <getetag>"e1"</getetag>
        <c:calendar-data>BEGIN:VCALENDAR
BEGIN:VTODO
UID:a
SUMMARY:Task A
END:VTODO
END:VCALENDAR</c:calendar-data>
      </prop>
      <status>HTTP/1.1 200 OK</status>
    </propstat>
  </response>
</multistatus>`))
	}))
	defer srv.Close()

	repo := NewTasksRepo(NewClient())
	snap, err := repo.SyncCollection(context.Background(), srv.URL, "alice", "secret", Collection{ID: "tasks", Href: "/tasks/", SupportsVTODO: true}, "")
	if err != nil {
		t.Fatalf("sync collection: %v", err)
	}
	if snap.Mode != "etag-fallback" {
		t.Fatalf("expected etag-fallback, got %q", snap.Mode)
	}
}
