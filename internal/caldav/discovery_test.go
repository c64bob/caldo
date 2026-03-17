package caldav

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDiscoverTaskCollections_UsesServerPrincipalAndListsVTODOCollections(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml; charset=utf-8")
		if r.Method != "PROPFIND" {
			t.Fatalf("expected PROPFIND, got %s", r.Method)
		}
		switch r.URL.Path {
		case "", "/":
			w.WriteHeader(http.StatusMultiStatus)
			_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<multistatus xmlns="DAV:">
  <response>
    <href>/</href>
    <propstat>
      <prop><current-user-principal><href>/principals/users/alice/</href></current-user-principal></prop>
      <status>HTTP/1.1 200 OK</status>
    </propstat>
  </response>
</multistatus>`))
		case "/principals/users/alice/":
			w.WriteHeader(http.StatusMultiStatus)
			_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<multistatus xmlns="DAV:" xmlns:c="urn:ietf:params:xml:ns:caldav">
  <response>
    <href>/principals/users/alice/</href>
    <propstat>
      <prop><c:calendar-home-set><href>/dav/calendars/alice/</href></c:calendar-home-set></prop>
      <status>HTTP/1.1 200 OK</status>
    </propstat>
  </response>
</multistatus>`))
		case "/dav/calendars/alice/":
			w.WriteHeader(http.StatusMultiStatus)
			_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<multistatus xmlns="DAV:" xmlns:c="urn:ietf:params:xml:ns:caldav">
  <response>
    <href>/dav/calendars/alice/tasks/</href>
    <propstat>
      <prop>
        <displayname>Tasks</displayname>
        <resourcetype><collection/><c:calendar/></resourcetype>
        <c:supported-calendar-component-set><c:comp name="VTODO"/></c:supported-calendar-component-set>
      </prop>
      <status>HTTP/1.1 200 OK</status>
    </propstat>
  </response>
  <response>
    <href>/dav/calendars/alice/personal/</href>
    <propstat>
      <prop>
        <displayname>Personal</displayname>
        <resourcetype><collection/><c:calendar/></resourcetype>
        <c:supported-calendar-component-set><c:comp name="VTODO"/></c:supported-calendar-component-set>
      </prop>
      <status>HTTP/1.1 200 OK</status>
    </propstat>
  </response>
  <response>
    <href>/dav/calendars/alice/events/</href>
    <propstat>
      <prop>
        <displayname>Events</displayname>
        <resourcetype><collection/><c:calendar/></resourcetype>
        <c:supported-calendar-component-set><c:comp name="VEVENT"/></c:supported-calendar-component-set>
      </prop>
      <status>HTTP/1.1 200 OK</status>
    </propstat>
  </response>
</multistatus>`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	client := NewClient()
	result, err := client.DiscoverTaskCollections(context.Background(), srv.URL, "alice", "pw", "Personal")
	if err != nil {
		t.Fatalf("discover: %v", err)
	}
	if result.PrincipalHref != "/principals/users/alice/" {
		t.Fatalf("expected server principal href, got %q", result.PrincipalHref)
	}
	if len(result.Collections) != 2 {
		t.Fatalf("expected 2 VTODO collections, got %d", len(result.Collections))
	}
	if result.Collections[0].DisplayName != "Personal" {
		t.Fatalf("expected default list promoted to first, got %q", result.Collections[0].DisplayName)
	}
	if result.Collections[1].DisplayName != "Tasks" {
		t.Fatalf("expected second list Tasks, got %q", result.Collections[1].DisplayName)
	}
}
