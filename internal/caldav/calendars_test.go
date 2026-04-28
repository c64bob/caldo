package caldav

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCalendarClientListCalendars(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PROPFIND" {
			t.Fatalf("unexpected method: got %q want %q", r.Method, "PROPFIND")
		}
		if got := r.Header.Get("Depth"); got != "1" {
			t.Fatalf("unexpected depth: got %q want %q", got, "1")
		}
		w.Header().Set("Content-Type", "application/xml; charset=utf-8")
		w.WriteHeader(http.StatusMultiStatus)
		_, _ = w.Write([]byte(`<?xml version="1.0" encoding="utf-8"?>
<d:multistatus xmlns:d="DAV:" xmlns:c="urn:ietf:params:xml:ns:caldav">
  <d:response>
    <d:href>/calendars/work/</d:href>
    <d:propstat>
      <d:prop>
        <d:displayname>Work</d:displayname>
        <d:resourcetype><d:collection/><c:calendar/></d:resourcetype>
      </d:prop>
    </d:propstat>
  </d:response>
  <d:response>
    <d:href>/files/</d:href>
    <d:propstat>
      <d:prop>
        <d:displayname>Files</d:displayname>
        <d:resourcetype><d:collection/></d:resourcetype>
      </d:prop>
    </d:propstat>
  </d:response>
</d:multistatus>`))
	}))
	t.Cleanup(server.Close)

	client := NewCalendarClient(server.Client())
	calendars, err := client.ListCalendars(context.Background(), Credentials{
		URL:      server.URL,
		Username: "alice",
		Password: "secret",
	})
	if err != nil {
		t.Fatalf("list calendars: %v", err)
	}
	if len(calendars) != 1 {
		t.Fatalf("unexpected calendar count: got %d want %d", len(calendars), 1)
	}
	if calendars[0].Href != "/calendars/work/" || calendars[0].DisplayName != "Work" {
		t.Fatalf("unexpected calendar: %#v", calendars[0])
	}
}

func TestCalendarClientCreateCalendar(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "MKCALENDAR" {
			t.Fatalf("unexpected method: got %q want %q", r.Method, "MKCALENDAR")
		}
		if !strings.HasSuffix(r.URL.Path, "/new-project/") {
			t.Fatalf("unexpected path: %q", r.URL.Path)
		}
		w.WriteHeader(http.StatusCreated)
	}))
	t.Cleanup(server.Close)

	client := NewCalendarClient(server.Client())
	calendar, err := client.CreateCalendar(context.Background(), Credentials{
		URL:      server.URL + "/calendars",
		Username: "alice",
		Password: "secret",
	}, "New Project")
	if err != nil {
		t.Fatalf("create calendar: %v", err)
	}
	if calendar.Href != "/calendars/new-project/" {
		t.Fatalf("unexpected href: got %q want %q", calendar.Href, "/calendars/new-project/")
	}
	if calendar.DisplayName != "New Project" {
		t.Fatalf("unexpected display name: got %q want %q", calendar.DisplayName, "New Project")
	}
}

func TestCalendarClientRenameCalendar(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PROPPATCH" {
			t.Fatalf("unexpected method: got %q want %q", r.Method, "PROPPATCH")
		}
		if got := r.URL.Path; got != "/calendars/work/" {
			t.Fatalf("unexpected path: got %q want %q", got, "/calendars/work/")
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		if !strings.Contains(string(body), "<d:displayname>Renamed Work</d:displayname>") {
			t.Fatalf("missing displayname in body: %q", string(body))
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	client := NewCalendarClient(server.Client())
	calendar, err := client.RenameCalendar(context.Background(), Credentials{
		URL:      server.URL,
		Username: "alice",
		Password: "secret",
	}, "/calendars/work/", "Renamed Work")
	if err != nil {
		t.Fatalf("rename calendar: %v", err)
	}
	if calendar.Href != "/calendars/work/" {
		t.Fatalf("unexpected href: got %q want %q", calendar.Href, "/calendars/work/")
	}
	if calendar.DisplayName != "Renamed Work" {
		t.Fatalf("unexpected display name: got %q want %q", calendar.DisplayName, "Renamed Work")
	}
}

func TestCalendarClientRenameCalendarFailsOnMultiStatusPropstatFailure(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PROPPATCH" {
			t.Fatalf("unexpected method: got %q want %q", r.Method, "PROPPATCH")
		}
		w.Header().Set("Content-Type", "application/xml; charset=utf-8")
		w.WriteHeader(http.StatusMultiStatus)
		_, _ = w.Write([]byte(`<?xml version="1.0" encoding="utf-8"?>
<d:multistatus xmlns:d="DAV:">
  <d:response>
    <d:href>/calendars/work/</d:href>
    <d:propstat>
      <d:prop><d:displayname/></d:prop>
      <d:status>HTTP/1.1 403 Forbidden</d:status>
    </d:propstat>
  </d:response>
</d:multistatus>`))
	}))
	t.Cleanup(server.Close)

	client := NewCalendarClient(server.Client())
	_, err := client.RenameCalendar(context.Background(), Credentials{
		URL:      server.URL,
		Username: "alice",
		Password: "secret",
	}, "/calendars/work/", "Renamed Work")
	if !errors.Is(err, ErrCalendarRenameFailed) {
		t.Fatalf("expected ErrCalendarRenameFailed, got %v", err)
	}
}
