package sqlite

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestSyncStateRepo_UpsertAndGet(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	repo := NewSyncStateRepo(db)
	state := SyncState{PrincipalID: "alice", CollectionID: "tasks", SyncToken: "abc", LastSyncedAt: time.Now().UTC()}
	if err := repo.Upsert(context.Background(), state); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	got, ok, err := repo.Get(context.Background(), "alice", "tasks")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if !ok {
		t.Fatal("expected state")
	}
	if got.SyncToken != "abc" {
		t.Fatalf("expected token abc, got %q", got.SyncToken)
	}
}
