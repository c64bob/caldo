package sync

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"caldo/internal/caldav"
	"caldo/internal/db"
)

// TodoLister lists remote VTODO objects for a calendar.
type TodoLister interface {
	ListVTODOs(ctx context.Context, credentials caldav.Credentials, calendarHref string) ([]caldav.CalendarObject, error)
}

// WebDAVSyncer performs WebDAV sync-collection reports for a calendar.
type WebDAVSyncer interface {
	SyncCollection(ctx context.Context, credentials caldav.Credentials, calendarHref string, syncToken string) (caldav.SyncCollectionResult, error)
}

// CTagSyncer performs CTag/ETag metadata discovery and resource fetches.
type CTagSyncer interface {
	CalendarCTag(ctx context.Context, credentials caldav.Credentials, calendarHref string) (string, error)
	ListVTODOETags(ctx context.Context, credentials caldav.Credentials, calendarHref string) ([]caldav.CalendarObject, error)
	GetVTODO(ctx context.Context, credentials caldav.Credentials, todoHref string) (string, string, error)
}

type databaseProjectStore struct {
	database *db.Database
}

// NewRunner creates the normal CalDAV sync runner with fallback strategies.
func NewRunner(database *db.Database, encryptionKey []byte, todos TodoLister) (*Engine, error) {
	if database == nil || todos == nil {
		return nil, fmt.Errorf("new sync runner: dependencies are required")
	}

	fullscan, err := NewFullScanRunner(database, encryptionKey, todos)
	if err != nil {
		return nil, err
	}

	var webdavRunner StrategyRunner = FallbackRunner{}
	if webdavSync, ok := todos.(WebDAVSyncer); ok {
		webdavRunner, err = NewWebDAVSyncRunner(database, encryptionKey, webdavSync)
		if err != nil {
			return nil, err
		}
	}

	var ctagRunner StrategyRunner = FallbackRunner{}
	if ctagSync, ok := todos.(CTagSyncer); ok {
		ctagRunner, err = NewCTagSyncRunner(database, encryptionKey, ctagSync)
		if err != nil {
			return nil, err
		}
	}

	return NewEngine(databaseProjectStore{database: database}, webdavRunner, ctagRunner, fullscan)
}

func (s databaseProjectStore) ListSyncProjects(ctx context.Context) ([]ProjectState, error) {
	projects, err := s.database.ListSyncProjects(ctx)
	if err != nil {
		return nil, err
	}

	states := make([]ProjectState, 0, len(projects))
	for _, project := range projects {
		states = append(states, ProjectState{
			ID:           project.ID,
			CalendarHref: project.CalendarHref,
			DisplayName:  project.DisplayName,
			SyncStrategy: project.SyncStrategy,
			SyncToken:    project.SyncToken,
			CTag:         project.CTag,
		})
	}
	return states, nil
}

func (s databaseProjectStore) UpdateProjectSyncStrategy(ctx context.Context, projectID string, strategy string) error {
	return s.database.UpdateProjectSyncStrategy(ctx, projectID, strategy)
}

// FallbackRunner is used for sync strategies that are not implemented yet.
type FallbackRunner struct{}

// Run reports that the next fallback strategy should be used.
func (FallbackRunner) Run(context.Context, ProjectState) error {
	return ErrFallbackRequired
}

// WebDAVSyncRunner imports incremental remote VTODO changes using WebDAV Sync.
type WebDAVSyncRunner struct {
	database      *db.Database
	encryptionKey []byte
	todos         WebDAVSyncer
}

// NewWebDAVSyncRunner creates a WebDAV Sync strategy runner.
func NewWebDAVSyncRunner(database *db.Database, encryptionKey []byte, todos WebDAVSyncer) (*WebDAVSyncRunner, error) {
	if database == nil || todos == nil {
		return nil, fmt.Errorf("new webdav sync runner: dependencies are required")
	}
	return &WebDAVSyncRunner{database: database, encryptionKey: encryptionKey, todos: todos}, nil
}

// Run executes one WebDAV sync-collection pass for a project.
func (r *WebDAVSyncRunner) Run(ctx context.Context, project ProjectState) error {
	credentials, err := r.database.LoadCalDAVCredentials(ctx, r.encryptionKey)
	if err != nil {
		return fmt.Errorf("webdav sync: load caldav credentials: %w", err)
	}

	result, err := r.todos.SyncCollection(ctx, caldav.Credentials{
		URL:      credentials.URL,
		Username: credentials.Username,
		Password: credentials.Password,
	}, project.CalendarHref, project.SyncToken)
	if err != nil {
		if errors.Is(err, ErrFallbackRequired) {
			return err
		}
		if errors.Is(err, caldav.ErrSyncCollectionUnsupported) {
			return ErrFallbackRequired
		}
		return fmt.Errorf("webdav sync: sync collection: %w", err)
	}

	tasks := MapCalendarObjects(result.Changed, project.DisplayName)
	baseline := strings.TrimSpace(project.SyncToken) == ""
	if _, err := r.database.ApplyWebDAVSyncProject(ctx, project.ID, tasks, result.DeletedHrefs, result.SyncToken, baseline); err != nil {
		return fmt.Errorf("webdav sync: apply project: %w", err)
	}
	return nil
}

// FullScanRunner imports remote VTODO state using a calendar-query full scan.
type FullScanRunner struct {
	database      *db.Database
	encryptionKey []byte
	todos         TodoLister
}

// NewFullScanRunner creates a full-scan strategy runner.
func NewFullScanRunner(database *db.Database, encryptionKey []byte, todos TodoLister) (*FullScanRunner, error) {
	if database == nil || todos == nil {
		return nil, fmt.Errorf("new fullscan runner: dependencies are required")
	}
	return &FullScanRunner{database: database, encryptionKey: encryptionKey, todos: todos}, nil
}

// Run executes one full-scan import for a project.
func (r *FullScanRunner) Run(ctx context.Context, project ProjectState) error {
	credentials, err := r.database.LoadCalDAVCredentials(ctx, r.encryptionKey)
	if err != nil {
		return fmt.Errorf("fullscan sync: load caldav credentials: %w", err)
	}

	objects, err := r.todos.ListVTODOs(ctx, caldav.Credentials{
		URL:      credentials.URL,
		Username: credentials.Username,
		Password: credentials.Password,
	}, project.CalendarHref)
	if err != nil {
		return fmt.Errorf("fullscan sync: list vtodos: %w", err)
	}

	tasks := MapCalendarObjects(objects, project.DisplayName)
	if _, err := r.database.ApplyFullScanProject(ctx, project.ID, tasks); err != nil {
		return fmt.Errorf("fullscan sync: apply project: %w", err)
	}
	return nil
}
