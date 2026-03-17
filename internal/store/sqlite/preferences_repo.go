package sqlite

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type Preferences struct {
	PrincipalID         string   `json:"principal_id"`
	DefaultView         string   `json:"default_view"`
	SyncIntervalSeconds int      `json:"sync_interval_seconds"`
	VisibleColumns      []string `json:"visible_columns"`
}

type PreferencesRepo struct {
	filePath string
	mu       sync.Mutex
}

func NewPreferencesRepo(db *DB) *PreferencesRepo {
	return &PreferencesRepo{filePath: db.Path + ".preferences.json"}
}

func (r *PreferencesRepo) GetByPrincipal(_ context.Context, principalID string) (Preferences, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	all, err := r.loadAll()
	if err != nil {
		return Preferences{}, false, err
	}
	p, ok := all[principalID]
	return p, ok, nil
}

func (r *PreferencesRepo) Upsert(_ context.Context, p Preferences) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	all, err := r.loadAll()
	if err != nil {
		return err
	}
	all[p.PrincipalID] = p
	enc, err := json.MarshalIndent(all, "", "  ")
	if err != nil {
		return fmt.Errorf("encode preferences store: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(r.filePath), 0o755); err != nil {
		return fmt.Errorf("create preferences store dir: %w", err)
	}
	if err := os.WriteFile(r.filePath, enc, 0o600); err != nil {
		return fmt.Errorf("write preferences store: %w", err)
	}
	return nil
}

func (r *PreferencesRepo) loadAll() (map[string]Preferences, error) {
	data, err := os.ReadFile(r.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]Preferences{}, nil
		}
		return nil, fmt.Errorf("read preferences store: %w", err)
	}
	if len(data) == 0 {
		return map[string]Preferences{}, nil
	}
	all := map[string]Preferences{}
	if err := json.Unmarshal(data, &all); err != nil {
		return nil, fmt.Errorf("decode preferences store: %w", err)
	}
	return all, nil
}
