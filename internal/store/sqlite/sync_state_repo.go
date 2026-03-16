package sqlite

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type SyncState struct {
	PrincipalID   string    `json:"principal_id"`
	CollectionID  string    `json:"collection_id"`
	SyncToken     string    `json:"sync_token"`
	ETagDigest    string    `json:"etag_digest"`
	ResourceCount int       `json:"resource_count"`
	LastMode      string    `json:"last_mode"`
	LastSyncedAt  time.Time `json:"last_synced_at"`
	LastError     string    `json:"last_error,omitempty"`
	LastErrorAt   time.Time `json:"last_error_at,omitempty"`
}

type SyncStateRepo struct {
	filePath string
	mu       sync.Mutex
}

func NewSyncStateRepo(db *DB) *SyncStateRepo {
	return &SyncStateRepo{filePath: db.Path + ".sync_state.json"}
}

func (r *SyncStateRepo) Get(_ context.Context, principalID, collectionID string) (SyncState, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	all, err := r.readAllLocked()
	if err != nil {
		return SyncState{}, false, err
	}
	state, ok := all[r.makeKey(principalID, collectionID)]
	return state, ok, nil
}

func (r *SyncStateRepo) Upsert(_ context.Context, state SyncState) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	all, err := r.readAllLocked()
	if err != nil {
		return err
	}
	all[r.makeKey(state.PrincipalID, state.CollectionID)] = state
	return r.writeAllLocked(all)
}

func (r *SyncStateRepo) SaveError(_ context.Context, principalID, collectionID, message string, now time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	all, err := r.readAllLocked()
	if err != nil {
		return err
	}
	key := r.makeKey(principalID, collectionID)
	state := all[key]
	state.PrincipalID = principalID
	state.CollectionID = collectionID
	state.LastError = strings.TrimSpace(message)
	state.LastErrorAt = now.UTC()
	all[key] = state
	return r.writeAllLocked(all)
}

func (r *SyncStateRepo) readAllLocked() (map[string]SyncState, error) {
	data, err := os.ReadFile(r.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]SyncState{}, nil
		}
		return nil, fmt.Errorf("read sync-state store: %w", err)
	}
	if len(data) == 0 {
		return map[string]SyncState{}, nil
	}
	out := map[string]SyncState{}
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("decode sync-state store: %w", err)
	}
	return out, nil
}

func (r *SyncStateRepo) writeAllLocked(all map[string]SyncState) error {
	encoded, err := json.MarshalIndent(all, "", "  ")
	if err != nil {
		return fmt.Errorf("encode sync-state store: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(r.filePath), 0o755); err != nil {
		return fmt.Errorf("create sync-state dir: %w", err)
	}
	if err := os.WriteFile(r.filePath, encoded, 0o600); err != nil {
		return fmt.Errorf("write sync-state store: %w", err)
	}
	return nil
}

func (r *SyncStateRepo) makeKey(principalID, collectionID string) string {
	return strings.TrimSpace(principalID) + "::" + strings.TrimSpace(collectionID)
}
