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

// ProjectOption contains one selectable project for task editing.
type ProjectOption struct {
	ID          string
	DisplayName string
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

// ListProjectOptions returns projects in the same order used by navigation.
func (d *Database) ListProjectOptions(ctx context.Context) ([]ProjectOption, error) {
	rows, err := d.Conn.QueryContext(ctx, `
SELECT id, display_name
FROM projects
ORDER BY is_default DESC, display_name COLLATE NOCASE ASC, id ASC;
`)
	if err != nil {
		return nil, fmt.Errorf("list project options: %w", err)
	}
	defer rows.Close()

	projects := make([]ProjectOption, 0)
	for rows.Next() {
		var project ProjectOption
		if err := rows.Scan(&project.ID, &project.DisplayName); err != nil {
			return nil, fmt.Errorf("list project options: scan project: %w", err)
		}
		projects = append(projects, project)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list project options: iterate projects: %w", err)
	}

	return projects, nil
}
