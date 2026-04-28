package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

var (
	// ErrTaskProjectNotFound indicates that an explicitly selected project does not exist.
	ErrTaskProjectNotFound = errors.New("task project not found")
	// ErrTaskProjectUnavailable indicates that no valid default project is configured.
	ErrTaskProjectUnavailable = errors.New("task project unavailable")
)

// TaskProject describes the project context used for task creation.
type TaskProject struct {
	ID           string
	CalendarHref string
	DisplayName  string
}

// NewTaskInput contains required fields to insert a pending task row.
type NewTaskInput struct {
	ProjectID   string
	ProjectName string
	UID         string
	Href        string
	Title       string
	RawVTODO    string
}

// ResolveTaskProject resolves an explicit project or falls back to the configured default project.
func (d *Database) ResolveTaskProject(ctx context.Context, explicitProjectID string) (TaskProject, error) {
	projectID := strings.TrimSpace(explicitProjectID)
	if projectID != "" {
		project, err := d.loadProjectByID(ctx, projectID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return TaskProject{}, ErrTaskProjectNotFound
			}
			return TaskProject{}, fmt.Errorf("resolve task project: load explicit project: %w", err)
		}
		return project, nil
	}

	var defaultProjectID sql.NullString
	if err := d.Conn.QueryRowContext(ctx, `
SELECT default_project_id
FROM settings
WHERE id = 'default';
`).Scan(&defaultProjectID); err != nil {
		return TaskProject{}, fmt.Errorf("resolve task project: load default project id: %w", err)
	}
	if !defaultProjectID.Valid || strings.TrimSpace(defaultProjectID.String) == "" {
		return TaskProject{}, ErrTaskProjectUnavailable
	}

	project, err := d.loadProjectByID(ctx, defaultProjectID.String)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return TaskProject{}, ErrTaskProjectUnavailable
		}
		return TaskProject{}, fmt.Errorf("resolve task project: load default project: %w", err)
	}

	return project, nil
}

// InsertPendingTask inserts a new task row in pending sync state and returns its id.
func (d *Database) InsertPendingTask(ctx context.Context, input NewTaskInput) (string, error) {
	if strings.TrimSpace(input.ProjectID) == "" {
		return "", fmt.Errorf("insert pending task: project id is required")
	}
	if strings.TrimSpace(input.UID) == "" {
		return "", fmt.Errorf("insert pending task: uid is required")
	}
	if strings.TrimSpace(input.Href) == "" {
		return "", fmt.Errorf("insert pending task: href is required")
	}
	if strings.TrimSpace(input.Title) == "" {
		return "", fmt.Errorf("insert pending task: title is required")
	}
	if strings.TrimSpace(input.RawVTODO) == "" {
		return "", fmt.Errorf("insert pending task: raw vtodo is required")
	}

	taskID := uuid.NewString()

	d.WriteMu.Lock()
	defer d.WriteMu.Unlock()

	if _, err := d.Conn.ExecContext(ctx, `
INSERT INTO tasks (
    id, project_id, uid, href, title, status, raw_vtodo, base_vtodo, project_name, sync_status, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, 'needs-action', ?, ?, ?, 'pending', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
`, taskID, input.ProjectID, input.UID, input.Href, input.Title, input.RawVTODO, input.RawVTODO, nullableString(input.ProjectName)); err != nil {
		return "", fmt.Errorf("insert pending task: insert task: %w", err)
	}

	return taskID, nil
}

// MarkTaskCreateSynced marks a pending task as synced and stores the returned ETag.
func (d *Database) MarkTaskCreateSynced(ctx context.Context, taskID string, etag string) error {
	d.WriteMu.Lock()
	defer d.WriteMu.Unlock()

	result, err := d.Conn.ExecContext(ctx, `
UPDATE tasks
SET etag = ?,
    sync_status = 'synced',
    server_version = server_version + 1,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?;
`, nullableString(etag), taskID)
	if err != nil {
		return fmt.Errorf("mark task create synced: update task: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("mark task create synced: read affected rows: %w", err)
	}
	if affected != 1 {
		return fmt.Errorf("mark task create synced: expected 1 row affected, got %d", affected)
	}

	return nil
}

// MarkTaskCreateError marks a pending task as error when synchronous CalDAV create fails.
func (d *Database) MarkTaskCreateError(ctx context.Context, taskID string) error {
	d.WriteMu.Lock()
	defer d.WriteMu.Unlock()

	result, err := d.Conn.ExecContext(ctx, `
UPDATE tasks
SET sync_status = 'error',
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?;
`, taskID)
	if err != nil {
		return fmt.Errorf("mark task create error: update task: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("mark task create error: read affected rows: %w", err)
	}
	if affected != 1 {
		return fmt.Errorf("mark task create error: expected 1 row affected, got %d", affected)
	}

	return nil
}

func (d *Database) loadProjectByID(ctx context.Context, projectID string) (TaskProject, error) {
	var project TaskProject
	err := d.Conn.QueryRowContext(ctx, `
SELECT id, calendar_href, display_name
FROM projects
WHERE id = ?;
`, strings.TrimSpace(projectID)).Scan(&project.ID, &project.CalendarHref, &project.DisplayName)
	if err != nil {
		return TaskProject{}, err
	}
	return project, nil
}
