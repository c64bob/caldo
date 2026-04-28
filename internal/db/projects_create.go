package db

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// NewProjectInput captures required data to persist a newly created project.
type NewProjectInput struct {
	CalendarHref string
	DisplayName  string
	SyncStrategy string
}

// ProjectRecord represents one persisted project.
type ProjectRecord struct {
	ID           string
	CalendarHref string
	DisplayName  string
	SyncStrategy string
}

// InsertProject inserts a newly created project after successful remote calendar creation.
func (d *Database) InsertProject(ctx context.Context, input NewProjectInput) (ProjectRecord, error) {
	calendarHref := strings.TrimSpace(input.CalendarHref)
	if calendarHref == "" {
		return ProjectRecord{}, fmt.Errorf("insert project: calendar href is required")
	}

	displayName := strings.TrimSpace(input.DisplayName)
	if displayName == "" {
		return ProjectRecord{}, fmt.Errorf("insert project: display name is required")
	}

	syncStrategy := strings.TrimSpace(input.SyncStrategy)
	if syncStrategy == "" {
		syncStrategy = "fullscan"
	}

	project := ProjectRecord{
		ID:           uuid.NewString(),
		CalendarHref: calendarHref,
		DisplayName:  displayName,
		SyncStrategy: syncStrategy,
	}

	d.WriteMu.Lock()
	defer d.WriteMu.Unlock()

	if _, err := d.Conn.ExecContext(ctx, `
INSERT INTO projects (
    id, calendar_href, display_name, sync_strategy, created_at, updated_at
) VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
`, project.ID, project.CalendarHref, project.DisplayName, project.SyncStrategy); err != nil {
		return ProjectRecord{}, fmt.Errorf("insert project: %w", err)
	}

	return project, nil
}
