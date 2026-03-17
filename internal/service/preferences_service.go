package service

import (
	"context"
	"strings"

	"caldo/internal/store/sqlite"
)

type PreferencesService struct {
	repo *sqlite.PreferencesRepo
}

type PreferencesInput struct {
	PrincipalID         string
	DefaultView         string
	SyncIntervalSeconds int
	VisibleColumns      []string
}

func NewPreferencesService(repo *sqlite.PreferencesRepo) *PreferencesService {
	return &PreferencesService{repo: repo}
}

func (s *PreferencesService) GetOrDefault(ctx context.Context, principalID string) (sqlite.Preferences, error) {
	stored, ok, err := s.repo.GetByPrincipal(ctx, principalID)
	if err != nil {
		return sqlite.Preferences{}, err
	}
	if ok {
		if len(stored.VisibleColumns) == 0 {
			stored.VisibleColumns = defaultColumns()
		}
		if strings.TrimSpace(stored.DefaultView) == "" {
			stored.DefaultView = "main"
		}
		if stored.SyncIntervalSeconds <= 0 {
			stored.SyncIntervalSeconds = 300
		}
		return stored, nil
	}
	return sqlite.Preferences{PrincipalID: principalID, DefaultView: "main", SyncIntervalSeconds: 300, VisibleColumns: defaultColumns()}, nil
}

func (s *PreferencesService) Save(ctx context.Context, in PreferencesInput) error {
	if strings.TrimSpace(in.PrincipalID) == "" {
		return nil
	}
	view := strings.TrimSpace(in.DefaultView)
	if view == "" {
		view = "main"
	}
	interval := in.SyncIntervalSeconds
	if interval <= 0 {
		interval = 300
	}
	cols := sanitizeColumns(in.VisibleColumns)
	if len(cols) == 0 {
		cols = defaultColumns()
	}
	return s.repo.Upsert(ctx, sqlite.Preferences{
		PrincipalID:         strings.TrimSpace(in.PrincipalID),
		DefaultView:         view,
		SyncIntervalSeconds: interval,
		VisibleColumns:      cols,
	})
}

func defaultColumns() []string {
	return []string{"star", "check", "name", "folder", "context", "due", "priority"}
}

func sanitizeColumns(in []string) []string {
	allowed := map[string]struct{}{"star": {}, "check": {}, "name": {}, "folder": {}, "context": {}, "due": {}, "priority": {}}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, item := range in {
		v := strings.TrimSpace(item)
		if _, ok := allowed[v]; !ok {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}
