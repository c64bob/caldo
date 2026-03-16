package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"caldo/internal/security"
	"caldo/internal/store/sqlite"
)

func newSettingsServiceForTest(t *testing.T) (*SettingsService, string, []byte) {
	t.Helper()
	key := []byte("0123456789abcdef0123456789abcdef")
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "caldo.db")
	db, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	repo := sqlite.NewDAVAccountsRepo(db)
	return NewSettingsService(repo, key, ""), dbPath + ".dav_accounts.json", key
}

func TestSaveDAVAccount_MissingRequiredFieldsFails(t *testing.T) {
	svc, _, _ := newSettingsServiceForTest(t)

	err := svc.SaveDAVAccount(context.Background(), SaveDAVAccountInput{})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestSaveDAVAccount_ConnectivityAuthError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	svc, _, _ := newSettingsServiceForTest(t)
	err := svc.SaveDAVAccount(context.Background(), SaveDAVAccountInput{
		PrincipalID: "alice",
		ServerURL:   srv.URL,
		Username:    "alice",
		Password:    "wrong",
	})
	if err == nil || !strings.Contains(err.Error(), "Anmeldung fehlgeschlagen") {
		t.Fatalf("expected auth error, got %v", err)
	}
}

func TestSaveDAVAccount_ConnectivityServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	svc, _, _ := newSettingsServiceForTest(t)
	err := svc.SaveDAVAccount(context.Background(), SaveDAVAccountInput{
		PrincipalID: "alice",
		ServerURL:   srv.URL,
		Username:    "alice",
		Password:    "pw",
	})
	if err == nil || !strings.Contains(err.Error(), "erhalten 500") {
		t.Fatalf("expected server status error, got %v", err)
	}
}

func TestSaveDAVAccount_Success_StoresEncryptedPassword(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PROPFIND" {
			t.Fatalf("expected PROPFIND, got %s", r.Method)
		}
		if got := r.Header.Get("Depth"); got != "0" {
			t.Fatalf("expected Depth=0, got %q", got)
		}
		user, pass, ok := r.BasicAuth()
		if !ok || user != "alice" || pass != "topsecret" {
			t.Fatalf("unexpected basic auth: user=%q ok=%v", user, ok)
		}
		w.WriteHeader(http.StatusMultiStatus)
	}))
	defer srv.Close()

	svc, storePath, key := newSettingsServiceForTest(t)
	err := svc.SaveDAVAccount(context.Background(), SaveDAVAccountInput{
		PrincipalID: "alice@example.com",
		ServerURL:   srv.URL,
		Username:    "alice",
		Password:    "topsecret",
	})
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}

	data, err := os.ReadFile(storePath)
	if err != nil {
		t.Fatalf("read store file: %v", err)
	}

	stored := map[string]sqlite.DAVAccount{}
	if err := json.Unmarshal(data, &stored); err != nil {
		t.Fatalf("unmarshal store: %v", err)
	}
	acc, ok := stored["alice@example.com"]
	if !ok {
		t.Fatal("expected persisted account")
	}
	if string(acc.PasswordEncrypted) == "topsecret" {
		t.Fatal("password must be stored encrypted")
	}
	plain, err := security.DecryptAESGCM(key, acc.PasswordEncrypted)
	if err != nil {
		t.Fatalf("decrypt stored password: %v", err)
	}
	if string(plain) != "topsecret" {
		t.Fatalf("unexpected decrypted password: %q", plain)
	}
}

func TestSaveDAVAccount_RejectsNonDAV200Response(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	svc, _, _ := newSettingsServiceForTest(t)
	err := svc.SaveDAVAccount(context.Background(), SaveDAVAccountInput{
		PrincipalID: "alice",
		ServerURL:   srv.URL,
		Username:    "alice",
		Password:    "pw",
	})
	if err == nil || !strings.Contains(err.Error(), "erwartet HTTP 207") {
		t.Fatalf("expected DAV 207 validation error, got %v", err)
	}
}

func TestSaveDAVAccount_RejectsMismatchingHost(t *testing.T) {
	svc, _, _ := newSettingsServiceForTest(t)
	svc.allowedHostPort = "nextcloud.example.com"

	err := svc.SaveDAVAccount(context.Background(), SaveDAVAccountInput{
		PrincipalID: "alice",
		ServerURL:   "https://evil.example.com/remote.php/dav",
		Username:    "alice",
		Password:    "pw",
	})
	if err == nil || !strings.Contains(err.Error(), "konfigurierten CalDAV-Host") {
		t.Fatalf("expected host mismatch error, got %v", err)
	}
}

func TestSaveDAVAccount_RejectsEmbeddedCredentialsInURL(t *testing.T) {
	svc, _, _ := newSettingsServiceForTest(t)
	err := svc.SaveDAVAccount(context.Background(), SaveDAVAccountInput{
		PrincipalID: "alice",
		ServerURL:   "https://user:pass@nextcloud.example.com/remote.php/dav",
		Username:    "alice",
		Password:    "pw",
	})
	if err == nil || !strings.Contains(err.Error(), "eingebetteten Zugangsdaten") {
		t.Fatalf("expected embedded credentials validation error, got %v", err)
	}
}

func TestSaveDAVAccount_RejectsRedirectResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", "/login")
		w.WriteHeader(http.StatusFound)
	}))
	defer srv.Close()

	svc, _, _ := newSettingsServiceForTest(t)
	err := svc.SaveDAVAccount(context.Background(), SaveDAVAccountInput{
		PrincipalID: "alice",
		ServerURL:   srv.URL,
		Username:    "alice",
		Password:    "pw",
	})
	if err == nil || !strings.Contains(err.Error(), "erwartet HTTP 207") {
		t.Fatalf("expected redirect to fail DAV validation, got %v", err)
	}
}
