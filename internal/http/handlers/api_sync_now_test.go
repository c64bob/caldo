package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"caldo/internal/http/middleware"
	"caldo/internal/security"
	"caldo/internal/service"
	"caldo/internal/store/sqlite"
)

func TestAPISyncNow_IgnoresPrincipalOverrideFromForm(t *testing.T) {
	var usedUser string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, _, _ := r.BasicAuth()
		usedUser = u
		if r.Method == "PROPFIND" {
			w.Header().Set("Content-Type", "application/xml; charset=utf-8")
			w.WriteHeader(http.StatusMultiStatus)
			switch r.URL.Path {
			case "", "/":
				_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<multistatus xmlns="DAV:">
  <response>
    <href>/</href>
    <propstat>
      <prop><current-user-principal><href>/principals/alice/</href></current-user-principal></prop>
      <status>HTTP/1.1 200 OK</status>
    </propstat>
  </response>
</multistatus>`))
			case "/principals/alice/":
				_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<multistatus xmlns="DAV:" xmlns:c="urn:ietf:params:xml:ns:caldav">
  <response>
    <href>/principals/alice/</href>
    <propstat>
      <prop><c:calendar-home-set><href>/tasks/</href></c:calendar-home-set></prop>
      <status>HTTP/1.1 200 OK</status>
    </propstat>
  </response>
</multistatus>`))
			case "/tasks/":
				_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<multistatus xmlns="DAV:" xmlns:c="urn:ietf:params:xml:ns:caldav">
  <response>
    <href>/tasks/</href>
    <propstat>
      <prop>
        <displayname>Tasks</displayname>
        <resourcetype><collection/><c:calendar/></resourcetype>
        <c:supported-calendar-component-set><c:comp name="VTODO"/></c:supported-calendar-component-set>
      </prop>
      <status>HTTP/1.1 200 OK</status>
    </propstat>
  </response>
</multistatus>`))
			default:
				w.WriteHeader(http.StatusNotFound)
			}
			return
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

	key := []byte("0123456789abcdef0123456789abcdef")
	db, err := sqlite.Open(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	accounts := sqlite.NewDAVAccountsRepo(db)
	encrypted, err := security.EncryptAESGCM(key, []byte("secret"))
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	for _, principal := range []string{"alice", "bob"} {
		if err := accounts.Upsert(context.Background(), sqlite.DAVAccount{
			PrincipalID:       principal,
			ServerURL:         srv.URL,
			Username:          principal,
			PasswordEncrypted: encrypted,
		}); err != nil {
			t.Fatalf("upsert account: %v", err)
		}
	}

	syncSvc := service.NewSyncService(accounts, sqlite.NewSyncStateRepo(db), key, "Tasks")
	h := &TasksHandler{SyncService: syncSvc}
	wrapped := middleware.ProxyAuth("X-Forwarded-User")(http.HandlerFunc(h.APISyncNow))

	form := url.Values{}
	form.Set("principal_id", "bob")
	req := httptest.NewRequest(http.MethodPost, "/api/sync/now", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Forwarded-User", "alice")
	rr := httptest.NewRecorder()

	wrapped.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", rr.Code, rr.Body.String())
	}
	if usedUser != "alice" {
		t.Fatalf("expected sync using authenticated principal alice, got %q", usedUser)
	}
}
