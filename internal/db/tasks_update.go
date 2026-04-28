package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	// ErrTaskNotFound indicates the referenced task does not exist.
	ErrTaskNotFound = errors.New("task not found")
	// ErrTaskVersionMismatch indicates optimistic-lock check failure.
	ErrTaskVersionMismatch = errors.New("task version mismatch")
)

// TaskUpdateInput contains all mutable fields for Story 7.2 task editing.
type TaskUpdateInput struct {
	TaskID          string
	ExpectedVersion int
	SessionID       string
	TabID           string

	ProjectID   string
	ProjectName string
	Href        string
	ETag        string
	RawVTODO    string

	Title       string
	Description string
	Status      string
	DueDate     sql.NullString
	DueAt       sql.NullTime
	Priority    sql.NullInt64
	LabelNames  sql.NullString
}

// PreparedTaskUpdate describes the local pending state persisted before CalDAV write.
type PreparedTaskUpdate struct {
	TaskID         string
	PreviousHref   string
	PreviousETag   string
	NextHref       string
	PendingVersion int
	ProjectChanged bool
}

// LoadTaskUpdateBase resolves task and target project metadata required for task editing.
func (d *Database) LoadTaskUpdateBase(ctx context.Context, taskID string, projectID string) (TaskUpdateInput, error) {
	trimmedTaskID := strings.TrimSpace(taskID)
	if trimmedTaskID == "" {
		return TaskUpdateInput{}, fmt.Errorf("load task update base: task id is required")
	}

	var base TaskUpdateInput
	base.TaskID = trimmedTaskID

	var currentProjectID, currentProjectName, uid, currentHref string
	var currentETag sql.NullString
	var currentVersion int
	if err := d.Conn.QueryRowContext(ctx, `
SELECT t.project_id, p.display_name, p.calendar_href, t.uid, t.href, t.etag, t.server_version, t.raw_vtodo
FROM tasks t
JOIN projects p ON p.id = t.project_id
WHERE t.id = ?;
`, trimmedTaskID).Scan(&currentProjectID, &currentProjectName, new(string), &uid, &currentHref, &currentETag, &currentVersion, &base.RawVTODO); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return TaskUpdateInput{}, ErrTaskNotFound
		}
		return TaskUpdateInput{}, fmt.Errorf("load task update base: query task row: %w", err)
	}

	base.ProjectID = currentProjectID
	base.ProjectName = currentProjectName
	base.Href = currentHref
	base.ETag = strings.TrimSpace(currentETag.String)
	base.ExpectedVersion = currentVersion

	targetProjectID := strings.TrimSpace(projectID)
	if targetProjectID == "" || targetProjectID == currentProjectID {
		return base, nil
	}

	var targetDisplayName, targetCalendarHref string
	if err := d.Conn.QueryRowContext(ctx, `
SELECT display_name, calendar_href
FROM projects
WHERE id = ?;
`, targetProjectID).Scan(&targetDisplayName, &targetCalendarHref); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return TaskUpdateInput{}, ErrTaskProjectNotFound
		}
		return TaskUpdateInput{}, fmt.Errorf("load task update base: load target project: %w", err)
	}

	base.ProjectID = targetProjectID
	base.ProjectName = targetDisplayName
	base.Href = joinCalendarTaskHref(targetCalendarHref, uid)

	return base, nil
}

// PrepareTaskUpdate marks the task row as pending and persists an undo snapshot in one transaction.
func (d *Database) PrepareTaskUpdate(ctx context.Context, input TaskUpdateInput) (PreparedTaskUpdate, error) {
	d.WriteMu.Lock()
	defer d.WriteMu.Unlock()

	tx, err := d.Conn.BeginTx(ctx, nil)
	if err != nil {
		return PreparedTaskUpdate{}, fmt.Errorf("prepare task update: begin transaction: %w", err)
	}
	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()

	var snapshotRaw string
	var snapshotETag sql.NullString
	var snapshotHref string
	if err := tx.QueryRowContext(ctx, `
SELECT raw_vtodo, etag, href
FROM tasks
WHERE id = ? AND server_version = ?;
`, input.TaskID, input.ExpectedVersion).Scan(&snapshotRaw, &snapshotETag, &snapshotHref); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return PreparedTaskUpdate{}, ErrTaskVersionMismatch
		}
		return PreparedTaskUpdate{}, fmt.Errorf("prepare task update: load current task row: %w", err)
	}

	expiresAt := time.Now().UTC().Add(5 * time.Minute)
	if _, err := tx.ExecContext(ctx, `
INSERT INTO undo_snapshots (
    id, session_id, tab_id, task_id, action_type, snapshot_vtodo, snapshot_fields, etag_at_snapshot, created_at, expires_at
) VALUES (?, ?, ?, ?, 'task_updated', ?, '{}', ?, CURRENT_TIMESTAMP, ?)
ON CONFLICT(session_id, tab_id) DO UPDATE SET
    id = excluded.id,
    task_id = excluded.task_id,
    action_type = excluded.action_type,
    snapshot_vtodo = excluded.snapshot_vtodo,
    snapshot_fields = excluded.snapshot_fields,
    etag_at_snapshot = excluded.etag_at_snapshot,
    created_at = CURRENT_TIMESTAMP,
    expires_at = excluded.expires_at;
`, uuid.NewString(), input.SessionID, input.TabID, input.TaskID, snapshotRaw, nullableString(snapshotETag.String), expiresAt.Format("2006-01-02T15:04:05Z")); err != nil {
		return PreparedTaskUpdate{}, fmt.Errorf("prepare task update: upsert undo snapshot: %w", err)
	}

	result, err := tx.ExecContext(ctx, `
UPDATE tasks
SET project_id = ?,
    title = ?,
    description = ?,
    status = ?,
    due_date = ?,
    due_at = ?,
    priority = ?,
    label_names = ?,
    project_name = ?,
    href = ?,
    raw_vtodo = ?,
    server_version = server_version + 1,
    sync_status = 'pending',
    updated_at = CURRENT_TIMESTAMP
WHERE id = ? AND server_version = ?;
`, input.ProjectID, input.Title, nullableString(input.Description), input.Status, nullValue(input.DueDate), nullValue(input.DueAt), nullValue(input.Priority), nullValue(input.LabelNames), nullableString(input.ProjectName), input.Href, input.RawVTODO, input.TaskID, input.ExpectedVersion)
	if err != nil {
		return PreparedTaskUpdate{}, fmt.Errorf("prepare task update: update task: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return PreparedTaskUpdate{}, fmt.Errorf("prepare task update: read affected rows: %w", err)
	}
	if affected != 1 {
		return PreparedTaskUpdate{}, ErrTaskVersionMismatch
	}

	if err := tx.Commit(); err != nil {
		return PreparedTaskUpdate{}, fmt.Errorf("prepare task update: commit transaction: %w", err)
	}
	tx = nil

	return PreparedTaskUpdate{
		TaskID:         input.TaskID,
		PreviousHref:   snapshotHref,
		PreviousETag:   strings.TrimSpace(snapshotETag.String),
		NextHref:       input.Href,
		PendingVersion: input.ExpectedVersion + 1,
		ProjectChanged: strings.TrimSpace(snapshotHref) != strings.TrimSpace(input.Href),
	}, nil
}

// MarkTaskUpdateSynced marks a pending task update as synced after successful CalDAV write.
func (d *Database) MarkTaskUpdateSynced(ctx context.Context, taskID string, expectedVersion int, etag string) error {
	d.WriteMu.Lock()
	defer d.WriteMu.Unlock()

	result, err := d.Conn.ExecContext(ctx, `
UPDATE tasks
SET etag = ?,
    sync_status = 'synced',
    server_version = server_version + 1,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ? AND server_version = ?;
`, nullableString(etag), taskID, expectedVersion)
	if err != nil {
		return fmt.Errorf("mark task update synced: update task: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("mark task update synced: read affected rows: %w", err)
	}
	if affected != 1 {
		return ErrTaskVersionMismatch
	}

	return nil
}

// MarkTaskUpdateError marks a pending task update as error when synchronous CalDAV write fails.
func (d *Database) MarkTaskUpdateError(ctx context.Context, taskID string, expectedVersion int) error {
	d.WriteMu.Lock()
	defer d.WriteMu.Unlock()

	result, err := d.Conn.ExecContext(ctx, `
UPDATE tasks
SET sync_status = 'error',
    updated_at = CURRENT_TIMESTAMP
WHERE id = ? AND server_version = ?;
`, taskID, expectedVersion)
	if err != nil {
		return fmt.Errorf("mark task update error: update task: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("mark task update error: read affected rows: %w", err)
	}
	if affected != 1 {
		return ErrTaskVersionMismatch
	}

	return nil
}

// MarkTaskUpdateConflict marks a pending task update as conflict when CalDAV reports an etag mismatch.
func (d *Database) MarkTaskUpdateConflict(ctx context.Context, taskID string, expectedVersion int) error {
	d.WriteMu.Lock()
	defer d.WriteMu.Unlock()

	result, err := d.Conn.ExecContext(ctx, `
UPDATE tasks
SET sync_status = 'conflict',
    updated_at = CURRENT_TIMESTAMP
WHERE id = ? AND server_version = ?;
`, taskID, expectedVersion)
	if err != nil {
		return fmt.Errorf("mark task update conflict: update task: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("mark task update conflict: read affected rows: %w", err)
	}
	if affected != 1 {
		return ErrTaskVersionMismatch
	}

	return nil
}

// MarkTaskUpdateErrorWithETag marks a pending task update as error and persists the latest etag.
func (d *Database) MarkTaskUpdateErrorWithETag(ctx context.Context, taskID string, expectedVersion int, etag string) error {
	d.WriteMu.Lock()
	defer d.WriteMu.Unlock()

	result, err := d.Conn.ExecContext(ctx, `
UPDATE tasks
SET etag = ?,
    sync_status = 'error',
    updated_at = CURRENT_TIMESTAMP
WHERE id = ? AND server_version = ?;
`, nullableString(etag), taskID, expectedVersion)
	if err != nil {
		return fmt.Errorf("mark task update error with etag: update task: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("mark task update error with etag: read affected rows: %w", err)
	}
	if affected != 1 {
		return ErrTaskVersionMismatch
	}

	return nil
}

func nullValue(value any) any {
	switch typed := value.(type) {
	case sql.NullString:
		if !typed.Valid {
			return nil
		}
		return typed.String
	case sql.NullTime:
		if !typed.Valid {
			return nil
		}
		return typed.Time
	case sql.NullInt64:
		if !typed.Valid {
			return nil
		}
		return typed.Int64
	default:
		return nil
	}
}

func joinCalendarTaskHref(calendarHref string, uid string) string {
	trimmed := strings.TrimSpace(calendarHref)
	if strings.HasSuffix(trimmed, "/") {
		return trimmed + strings.TrimSpace(uid) + ".ics"
	}
	return trimmed + "/" + strings.TrimSpace(uid) + ".ics"
}
