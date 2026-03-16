package sqlite

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
)

func TestDAVAccountsRepo_UpsertConcurrentDoesNotLoseEntries(t *testing.T) {
	tmp := t.TempDir()
	db, err := Open(filepath.Join(tmp, "caldo.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	repo := NewDAVAccountsRepo(db)

	const n = 20
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		i := i
		go func() {
			defer wg.Done()
			principal := "user-" + strconv.Itoa(i)
			err := repo.Upsert(context.Background(), DAVAccount{
				PrincipalID:       principal,
				ServerURL:         "https://nextcloud.example.com/remote.php/dav",
				Username:          principal,
				PasswordEncrypted: []byte{1, 2, 3},
			})
			if err != nil {
				t.Errorf("upsert failed: %v", err)
			}
		}()
	}
	wg.Wait()

	dataPath := db.Path + ".dav_accounts.json"
	bytes, err := os.ReadFile(dataPath)
	if err != nil {
		t.Fatalf("read persisted store: %v", err)
	}
	stored := map[string]DAVAccount{}
	if err := json.Unmarshal(bytes, &stored); err != nil {
		t.Fatalf("decode persisted store: %v", err)
	}
	if len(stored) != n {
		t.Fatalf("expected %d persisted accounts, got %d", n, len(stored))
	}
}
