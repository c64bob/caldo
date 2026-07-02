package sync

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"caldo/internal/caldav"
	"caldo/internal/db"
)

// CTagSyncRunner imports incremental remote VTODO changes using CTag and ETag comparison.
type CTagSyncRunner struct {
	database      *db.Database
	encryptionKey []byte
	todos         CTagSyncer
}

// NewCTagSyncRunner creates a CTag/ETag strategy runner.
func NewCTagSyncRunner(database *db.Database, encryptionKey []byte, todos CTagSyncer) (*CTagSyncRunner, error) {
	if database == nil || todos == nil {
		return nil, fmt.Errorf("new ctag sync runner: dependencies are required")
	}
	return &CTagSyncRunner{database: database, encryptionKey: encryptionKey, todos: todos}, nil
}

// Run executes one CTag/ETag comparison pass for a project.
func (r *CTagSyncRunner) Run(ctx context.Context, project ProjectState) error {
	credentials, err := r.database.LoadCalDAVCredentials(ctx, r.encryptionKey)
	if err != nil {
		return fmt.Errorf("ctag sync: load caldav credentials: %w", err)
	}
	todoCredentials := caldav.Credentials{
		URL:      credentials.URL,
		Username: credentials.Username,
		Password: credentials.Password,
	}

	remoteCTag, err := r.todos.CalendarCTag(ctx, todoCredentials, project.CalendarHref)
	if err != nil {
		if errors.Is(err, caldav.ErrCTagUnsupported) {
			return ErrFallbackRequired
		}
		return fmt.Errorf("ctag sync: load calendar ctag: %w", err)
	}
	remoteCTag = strings.TrimSpace(remoteCTag)
	if remoteCTag == "" {
		return ErrFallbackRequired
	}

	if strings.TrimSpace(project.CTag) != "" && strings.TrimSpace(project.CTag) == remoteCTag {
		return nil
	}

	localStates, err := r.database.ListProjectRemoteTaskState(ctx, project.ID)
	if err != nil {
		return fmt.Errorf("ctag sync: list local remote state: %w", err)
	}
	remoteStates, err := r.todos.ListVTODOETags(ctx, todoCredentials, project.CalendarHref)
	if err != nil {
		if errors.Is(err, caldav.ErrCTagUnsupported) {
			return ErrFallbackRequired
		}
		return fmt.Errorf("ctag sync: list remote etags: %w", err)
	}

	changed, deletedHrefs, err := r.changedObjects(ctx, todoCredentials, localStates, remoteStates)
	if err != nil {
		if errors.Is(err, caldav.ErrCTagUnsupported) {
			return ErrFallbackRequired
		}
		return err
	}

	tasks := MapCalendarObjects(changed, project.DisplayName)
	if _, err := r.database.ApplyCTagSyncProject(ctx, project.ID, tasks, deletedHrefs, remoteCTag, false); err != nil {
		return fmt.Errorf("ctag sync: apply project: %w", err)
	}
	return nil
}

func (r *CTagSyncRunner) changedObjects(ctx context.Context, credentials caldav.Credentials, localStates []db.ProjectRemoteTaskState, remoteStates []caldav.CalendarObject) ([]caldav.CalendarObject, []string, error) {
	localByHref := make(map[string]db.ProjectRemoteTaskState, len(localStates))
	for _, state := range localStates {
		hrefKey := normalizeRemoteHref(state.Href)
		if hrefKey == "" {
			continue
		}
		if _, exists := localByHref[hrefKey]; !exists {
			localByHref[hrefKey] = state
		}
	}

	remoteSeen := make(map[string]struct{}, len(remoteStates))
	changed := make([]caldav.CalendarObject, 0)
	deletedHrefs := make([]string, 0)
	for _, remote := range remoteStates {
		href := strings.TrimSpace(remote.Href)
		etag := strings.TrimSpace(remote.ETag)
		if href == "" || etag == "" {
			return nil, nil, caldav.ErrCTagUnsupported
		}

		hrefKey := normalizeRemoteHref(href)
		if hrefKey == "" {
			return nil, nil, caldav.ErrCTagUnsupported
		}
		remoteSeen[hrefKey] = struct{}{}

		local, exists := localByHref[hrefKey]
		if exists && strings.TrimSpace(local.ETag) == etag {
			continue
		}

		raw, getETag, err := r.todos.GetVTODO(ctx, credentials, href)
		if err != nil {
			return nil, nil, caldav.ErrCTagUnsupported
		}
		raw = strings.TrimSpace(raw)
		if !containsVTODO(raw) {
			if exists {
				deletedHrefs = append(deletedHrefs, local.Href)
			}
			continue
		}
		if strings.TrimSpace(getETag) != "" {
			etag = strings.TrimSpace(getETag)
		}
		changed = append(changed, caldav.CalendarObject{Href: href, ETag: etag, RawVTODO: raw})
	}

	for _, local := range localStates {
		hrefKey := normalizeRemoteHref(local.Href)
		if hrefKey == "" {
			continue
		}
		if _, exists := remoteSeen[hrefKey]; !exists {
			deletedHrefs = append(deletedHrefs, local.Href)
		}
	}

	return changed, deletedHrefs, nil
}

func containsVTODO(raw string) bool {
	return strings.Contains(strings.ToUpper(raw), "BEGIN:VTODO")
}

func normalizeRemoteHref(href string) string {
	trimmed := strings.TrimSpace(href)
	if trimmed == "" {
		return ""
	}
	parsed, err := url.Parse(trimmed)
	if err == nil && parsed.Path != "" {
		trimmed = parsed.Path
	}
	return strings.TrimSpace(trimmed)
}
