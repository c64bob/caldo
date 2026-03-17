package sqlite

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

type SavedFilter struct {
	PrincipalID string   `json:"principal_id"`
	Name        string   `json:"name"`
	Slug        string   `json:"slug"`
	ListID      string   `json:"list_id"`
	Priority    []string `json:"priority"`
	Status      []string `json:"status"`
	DueFrom     string   `json:"due_from"`
	DueTo       string   `json:"due_to"`
	Folder      string   `json:"folder"`
	Context     string   `json:"context"`
	Goal        string   `json:"goal"`
	Tags        string   `json:"tags"`
	Star        string   `json:"star"`
	Query       string   `json:"query"`
}

type SavedFiltersRepo struct {
	filePath string
	mu       sync.Mutex
}

func NewSavedFiltersRepo(db *DB) *SavedFiltersRepo {
	return &SavedFiltersRepo{filePath: db.Path + ".saved-filters.json"}
}

func (r *SavedFiltersRepo) ListByPrincipal(_ context.Context, principalID string) ([]SavedFilter, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	all, err := r.loadAll()
	if err != nil {
		return nil, err
	}
	items := append([]SavedFilter(nil), all[strings.TrimSpace(principalID)]...)
	sort.Slice(items, func(i, j int) bool {
		return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
	})
	return items, nil
}

func (r *SavedFiltersRepo) Upsert(_ context.Context, in SavedFilter) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	all, err := r.loadAll()
	if err != nil {
		return err
	}
	principal := strings.TrimSpace(in.PrincipalID)
	items := all[principal]
	replaced := false
	for i := range items {
		if strings.EqualFold(items[i].Slug, in.Slug) {
			items[i] = in
			replaced = true
			break
		}
	}
	if !replaced {
		items = append(items, in)
	}
	all[principal] = items
	return r.writeAll(all)
}

func (r *SavedFiltersRepo) GetBySlug(_ context.Context, principalID, slug string) (SavedFilter, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	all, err := r.loadAll()
	if err != nil {
		return SavedFilter{}, false, err
	}
	for _, item := range all[strings.TrimSpace(principalID)] {
		if strings.EqualFold(item.Slug, strings.TrimSpace(slug)) {
			return item, true, nil
		}
	}
	return SavedFilter{}, false, nil
}

func (r *SavedFiltersRepo) loadAll() (map[string][]SavedFilter, error) {
	data, err := os.ReadFile(r.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string][]SavedFilter{}, nil
		}
		return nil, fmt.Errorf("read saved filters store: %w", err)
	}
	all := map[string][]SavedFilter{}
	if len(data) == 0 {
		return all, nil
	}
	if err := json.Unmarshal(data, &all); err != nil {
		return nil, fmt.Errorf("decode saved filters store: %w", err)
	}
	return all, nil
}

func (r *SavedFiltersRepo) writeAll(all map[string][]SavedFilter) error {
	enc, err := json.MarshalIndent(all, "", "  ")
	if err != nil {
		return fmt.Errorf("encode saved filters store: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(r.filePath), 0o755); err != nil {
		return fmt.Errorf("create saved filters store dir: %w", err)
	}
	if err := os.WriteFile(r.filePath, enc, 0o600); err != nil {
		return fmt.Errorf("write saved filters store: %w", err)
	}
	return nil
}
