package caldav

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestConnectionTesterDetectsCapabilities(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PROPFIND" {
			t.Fatalf("unexpected method: got %q want %q", r.Method, "PROPFIND")
		}
		username, password, ok := r.BasicAuth()
		if !ok {
			t.Fatal("expected basic auth")
		}
		if username != "alice" || password != "secret" {
			t.Fatalf("unexpected basic auth credentials")
		}
		if got := r.Header.Get("Depth"); got != "0" {
			t.Fatalf("unexpected depth header: got %q", got)
		}

		w.Header().Set("DAV", "1, 2, calendar-access, sync-collection")
		w.Header().Set("Content-Type", "application/xml; charset=utf-8")
		w.WriteHeader(http.StatusMultiStatus)
		_, _ = w.Write([]byte(`<d:multistatus xmlns:d="DAV:" xmlns:cs="http://calendarserver.org/ns/">
<d:response>
<d:propstat><d:prop><d:getetag>\"123\"</d:getetag><cs:getctag>ctag-1</cs:getctag></d:prop></d:propstat>
</d:response></d:multistatus>`))
	}))
	defer server.Close()

	tester := NewConnectionTester(server.Client())
	capabilities, err := tester.TestConnection(context.Background(), Credentials{
		URL:      server.URL,
		Username: "alice",
		Password: "secret",
	})
	if err != nil {
		t.Fatalf("test connection: %v", err)
	}

	if !capabilities.WebDAVSync || !capabilities.CTag || !capabilities.ETag || !capabilities.FullScan {
		t.Fatalf("unexpected capabilities: %#v", capabilities)
	}
}

func TestConnectionTesterReturnsFailureForUnexpectedStatus(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	tester := NewConnectionTester(server.Client())
	_, err := tester.TestConnection(context.Background(), Credentials{
		URL:      server.URL,
		Username: "alice",
		Password: "secret",
	})
	if !errors.Is(err, ErrConnectionTestFailed) {
		t.Fatalf("expected connection test failure, got %v", err)
	}
}

func TestConnectionTesterHonorsTimeout(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(250 * time.Millisecond)
		w.WriteHeader(http.StatusMultiStatus)
	}))
	defer server.Close()

	tester := NewConnectionTester(server.Client())
	tester.timeout = 50 * time.Millisecond

	_, err := tester.TestConnection(context.Background(), Credentials{
		URL:      server.URL,
		Username: "alice",
		Password: "secret",
	})
	if !errors.Is(err, ErrConnectionTestFailed) {
		t.Fatalf("expected connection test failure, got %v", err)
	}
}

func TestDetectCapabilitiesFallbacks(t *testing.T) {
	t.Parallel()

	capabilities := detectCapabilities("1, calendar-access", "<d:multistatus/>")
	if capabilities.WebDAVSync {
		t.Fatal("webdav sync should be false when sync-collection is absent")
	}
	if capabilities.CTag {
		t.Fatal("ctag should be false when getctag is absent")
	}
	if capabilities.ETag {
		t.Fatal("etag should be false when getetag is absent")
	}
	if !capabilities.FullScan {
		t.Fatal("fullscan should always be true")
	}
}
