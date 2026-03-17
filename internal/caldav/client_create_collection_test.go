package caldav

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCreateCollection_EscapesDisplayName(t *testing.T) {
	var body string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "MKCALENDAR" {
			t.Fatalf("expected MKCALENDAR, got %s", r.Method)
		}
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		body = string(raw)
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	c := NewClient()
	err := c.CreateCollection(context.Background(), srv.URL, "alice", "pw", "/", "my-list", "A & B <Tasks>")
	if err != nil {
		t.Fatalf("create collection: %v", err)
	}
	if !strings.Contains(body, "A &amp; B &lt;Tasks&gt;") {
		t.Fatalf("expected escaped displayname in xml body, got: %s", body)
	}
}
