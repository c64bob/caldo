package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"caldo/internal/security"
	"caldo/internal/store/sqlite"
)

func newSyncServiceForTest(t *testing.T, serverURL string) (*SyncService, *sqlite.SyncStateRepo, *sqlite.DAVAccountsRepo) {
	t.Helper()
	key := []byte("0123456789abcdef0123456789abcdef")
	db, err := sqlite.Open(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	accounts := sqlite.NewDAVAccountsRepo(db)
	syncStates := sqlite.NewSyncStateRepo(db)
	enc, err := security.EncryptAESGCM(key, []byte("secret"))
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	if err := accounts.Upsert(context.Background(), sqlite.DAVAccount{
		PrincipalID:       "alice",
		ServerURL:         serverURL,
		Username:          "alice",
		PasswordEncrypted: enc,
	}); err != nil {
		t.Fatalf("upsert account: %v", err)
	}
	return NewSyncService(accounts, syncStates, key, "Tasks"), syncStates, accounts
}

func TestSyncNow_SuccessPersistsSyncState(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if respondDiscoveryPropfind(w, r) {
			return
		}
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

	svc, repo, _ := newSyncServiceForTest(t, srv.URL)
	result, err := svc.SyncNow(context.Background(), "alice")
	if err != nil {
		t.Fatalf("sync now: %v", err)
	}
	if result.SyncedCollections != 1 {
		t.Fatalf("expected 1 synced collection, got %d", result.SyncedCollections)
	}
	state, ok, err := repo.Get(context.Background(), "alice", "tasks")
	if err != nil {
		t.Fatalf("get sync-state: %v", err)
	}
	if !ok {
		t.Fatal("expected sync-state")
	}
	if strings.TrimSpace(state.LastMode) == "" {
		t.Fatal("expected sync mode")
	}
}

func TestSyncNow_FailsWithoutAccount(t *testing.T) {
	key := []byte("0123456789abcdef0123456789abcdef")
	db, err := sqlite.Open(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	svc := NewSyncService(sqlite.NewDAVAccountsRepo(db), sqlite.NewSyncStateRepo(db), key, "Tasks")
	if _, err := svc.SyncNow(context.Background(), "missing"); err == nil {
		t.Fatal("expected error")
	}
}
