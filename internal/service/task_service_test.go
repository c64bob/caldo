package service

import (
	"context"
	"path/filepath"
	"testing"

	"caldo/internal/security"
	"caldo/internal/store/sqlite"
)

func newTaskServiceForTest(t *testing.T) (*TaskService, *sqlite.DAVAccountsRepo, []byte) {
	t.Helper()
	tmp := t.TempDir()
	db, err := sqlite.Open(filepath.Join(tmp, "caldo.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	key := []byte("0123456789abcdef0123456789abcdef")
	repo := sqlite.NewDAVAccountsRepo(db)
	return NewTaskService(repo, key, "Tasks"), repo, key
}

func TestLoadTaskPage_NoCredentials(t *testing.T) {
	svc, _, _ := newTaskServiceForTest(t)
	data, err := svc.LoadTaskPage(context.Background(), "alice@example.com", "")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if data.HasCredentials {
		t.Fatal("expected HasCredentials=false")
	}
}

func TestLoadTaskPage_WithCredentialsReturnsListsAndTasks(t *testing.T) {
	svc, repo, key := newTaskServiceForTest(t)
	encrypted, err := security.EncryptAESGCM(key, []byte("pw"))
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	err = repo.Upsert(context.Background(), sqlite.DAVAccount{
		PrincipalID:       "alice@example.com",
		ServerURL:         "https://nextcloud.example.com/remote.php/dav",
		Username:          "alice",
		PasswordEncrypted: encrypted,
	})
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}

	data, err := svc.LoadTaskPage(context.Background(), "alice@example.com", "")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !data.HasCredentials {
		t.Fatal("expected HasCredentials=true")
	}
	if len(data.Lists) == 0 {
		t.Fatal("expected at least one list")
	}
	if len(data.Tasks) == 0 {
		t.Fatal("expected demo tasks")
	}
}
