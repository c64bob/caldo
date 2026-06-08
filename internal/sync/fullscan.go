package sync

import (
	"context"
	"fmt"

	"caldo/internal/caldav"
	"caldo/internal/db"
)

// TodoLister lists remote VTODO objects for a calendar.
type TodoLister interface {
	ListVTODOs(ctx context.Context, credentials caldav.Credentials, calendarHref string) ([]caldav.CalendarObject, error)
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

	fallback := FallbackRunner{}
	return NewEngine(databaseProjectStore{database: database}, fallback, fallback, fullscan)
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
