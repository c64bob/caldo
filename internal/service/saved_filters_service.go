package service

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"caldo/internal/store/sqlite"
)

type SavedFiltersService struct {
	repo *sqlite.SavedFiltersRepo
}

type SavedFilterInput struct {
	PrincipalID string
	Name        string
	ListID      string
	Priority    []string
	Status      []string
	DueFrom     string
	DueTo       string
	Folder      string
	Context     string
	Goal        string
	Tags        string
	Star        string
	Query       string
}

func NewSavedFiltersService(repo *sqlite.SavedFiltersRepo) *SavedFiltersService {
	return &SavedFiltersService{repo: repo}
}

func (s *SavedFiltersService) List(ctx context.Context, principalID string) ([]sqlite.SavedFilter, error) {
	return s.repo.ListByPrincipal(ctx, principalID)
}

func (s *SavedFiltersService) Save(ctx context.Context, in SavedFilterInput) error {
	name := strings.TrimSpace(in.Name)
	if strings.TrimSpace(in.PrincipalID) == "" || name == "" {
		return fmt.Errorf("missing principal or filter name")
	}
	return s.repo.Upsert(ctx, sqlite.SavedFilter{
		PrincipalID: strings.TrimSpace(in.PrincipalID),
		Name:        name,
		Slug:        slugify(name),
		ListID:      strings.TrimSpace(in.ListID),
		Priority:    dedupe(in.Priority),
		Status:      dedupe(in.Status),
		DueFrom:     strings.TrimSpace(in.DueFrom),
		DueTo:       strings.TrimSpace(in.DueTo),
		Folder:      strings.TrimSpace(in.Folder),
		Context:     strings.TrimSpace(in.Context),
		Goal:        strings.TrimSpace(in.Goal),
		Tags:        strings.TrimSpace(in.Tags),
		Star:        strings.TrimSpace(in.Star),
		Query:       strings.TrimSpace(in.Query),
	})
}

func (s *SavedFiltersService) Get(ctx context.Context, principalID, slug string) (sqlite.SavedFilter, bool, error) {
	return s.repo.GetBySlug(ctx, principalID, slug)
}

func dedupe(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, v := range values {
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			continue
		}
		key := strings.ToLower(trimmed)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}

var nonAlphaNum = regexp.MustCompile(`[^a-z0-9]+`)

func slugify(name string) string {
	slug := strings.ToLower(strings.TrimSpace(name))
	slug = nonAlphaNum.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	if slug == "" {
		return "filter"
	}
	return slug
}
