package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// TaskDeleteInput contains required values to prepare synchronous task deletion.
type TaskDeleteInput struct {
	TaskID          string
	ExpectedVersion int
	SessionID       string
	TabID           string
}

// PreparedTaskDelete contains persisted pending delete state before CalDAV deletion.
type PreparedTaskDelete struct {
	TaskID         string
	Href           string
	ETag           string
	PendingVersion int
}

// PrepareTaskDelete persists undo snapshot and marks task as pending delete.
func (d *Database) PrepareTaskDelete(ctx context.Context, input TaskDeleteInput) (PreparedTaskDelete, error) {
	d.WriteMu.Lock()
	defer d.WriteMu.Unlock()

	tx, err := d.Conn.BeginTx(ctx, nil)
	if err != nil {
		return PreparedTaskDelete{}, fmt.Errorf("prepare task delete: begin transaction: %w", err)
	}
	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()

	var snapshotRaw string
	var snapshotETag sql.NullString
	var snapshotHref string
	var snapshotProjectID string
	var snapshotTitle string
	var snapshotDescription sql.NullString
	var snapshotStatus string
	var snapshotDueDate sql.NullString
	var snapshotDueAt sql.NullTime
	var snapshotPriority sql.NullInt64
	var snapshotLabelNames sql.NullString
	if err := tx.QueryRowContext(ctx, `
SELECT raw_vtodo, etag, href, project_id, title, description, status, due_date, due_at, priority, label_names
FROM tasks
WHERE id = ? AND server_version = ?;
`, input.TaskID, input.ExpectedVersion).Scan(&snapshotRaw, &snapshotETag, &snapshotHref, &snapshotProjectID, &snapshotTitle, &snapshotDescription, &snapshotStatus, &snapshotDueDate, &snapshotDueAt, &snapshotPriority, &snapshotLabelNames); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return PreparedTaskDelete{}, ErrTaskVersionMismatch
		}
		return PreparedTaskDelete{}, fmt.Errorf("prepare task delete: load current task row: %w", err)
	}

	expiresAt := time.Now().UTC().Add(5 * time.Minute)
	if _, err := tx.ExecContext(ctx, `
INSERT INTO undo_snapshots (
    id, session_id, tab_id, task_id, action_type, snapshot_vtodo, snapshot_fields, etag_at_snapshot, created_at, expires_at
) VALUES (?, ?, ?, ?, 'task_deleted', ?, json_object('project_id', ?, 'title', ?, 'description', ?, 'status', ?, 'due_date', ?, 'due_at', ?, 'priority', ?, 'label_names', ?), ?, CURRENT_TIMESTAMP, ?)
ON CONFLICT(session_id, tab_id) DO UPDATE SET
    id = excluded.id,
    task_id = excluded.task_id,
    action_type = excluded.action_type,
    snapshot_vtodo = excluded.snapshot_vtodo,
    snapshot_fields = excluded.snapshot_fields,
    etag_at_snapshot = excluded.etag_at_snapshot,
    created_at = CURRENT_TIMESTAMP,
    expires_at = excluded.expires_at;
`, uuid.NewString(), input.SessionID, input.TabID, input.TaskID, snapshotRaw, snapshotProjectID, snapshotTitle, nullableString(snapshotDescription.String), snapshotStatus, nullableString(snapshotDueDate.String), nullableTimeToRFC3339(snapshotDueAt), nullableInt64(snapshotPriority), nullableString(snapshotLabelNames.String), nullableString(snapshotETag.String), expiresAt.Format("2006-01-02T15:04:05Z")); err != nil {
		return PreparedTaskDelete{}, fmt.Errorf("prepare task delete: upsert undo snapshot: %w", err)
	}

	result, err := tx.ExecContext(ctx, `
UPDATE tasks
SET sync_status = 'pending',
    server_version = server_version + 1,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ? AND server_version = ?;
`, input.TaskID, input.ExpectedVersion)
	if err != nil {
		return PreparedTaskDelete{}, fmt.Errorf("prepare task delete: mark task pending: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return PreparedTaskDelete{}, fmt.Errorf("prepare task delete: read affected rows: %w", err)
	}
	if affected != 1 {
		return PreparedTaskDelete{}, ErrTaskVersionMismatch
	}

	if err := tx.Commit(); err != nil {
		return PreparedTaskDelete{}, fmt.Errorf("prepare task delete: commit transaction: %w", err)
	}
	tx = nil

	return PreparedTaskDelete{
		TaskID:         input.TaskID,
		Href:           snapshotHref,
		ETag:           snapshotETag.String,
		PendingVersion: input.ExpectedVersion + 1,
	}, nil
}

// MarkTaskDeleteSynced removes task row after successful CalDAV delete.
func (d *Database) MarkTaskDeleteSynced(ctx context.Context, taskID string, expectedVersion int) error {
	d.WriteMu.Lock()
	defer d.WriteMu.Unlock()

	result, err := d.Conn.ExecContext(ctx, `
DELETE FROM tasks
WHERE id = ? AND server_version = ?;
`, taskID, expectedVersion)
	if err != nil {
		return fmt.Errorf("mark task delete synced: delete task: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("mark task delete synced: read affected rows: %w", err)
	}
	if affected != 1 {
		return ErrTaskVersionMismatch
	}

	return nil
}

// MarkTaskDeleteError marks pending delete as error when CalDAV delete fails.
func (d *Database) MarkTaskDeleteError(ctx context.Context, taskID string, expectedVersion int) error {
	d.WriteMu.Lock()
	defer d.WriteMu.Unlock()

	result, err := d.Conn.ExecContext(ctx, `
UPDATE tasks
SET sync_status = 'error',
    updated_at = CURRENT_TIMESTAMP
WHERE id = ? AND server_version = ?;
`, taskID, expectedVersion)
	if err != nil {
		return fmt.Errorf("mark task delete error: update task: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("mark task delete error: read affected rows: %w", err)
	}
	if affected != 1 {
		return ErrTaskVersionMismatch
	}

	return nil
}

// MarkTaskDeleteConflict marks pending delete as conflict when CalDAV rejects etag.
func (d *Database) MarkTaskDeleteConflict(ctx context.Context, taskID string, expectedVersion int) error {
	d.WriteMu.Lock()
	defer d.WriteMu.Unlock()

	result, err := d.Conn.ExecContext(ctx, `
UPDATE tasks
SET sync_status = 'conflict',
    updated_at = CURRENT_TIMESTAMP
WHERE id = ? AND server_version = ?;
`, taskID, expectedVersion)
	if err != nil {
		return fmt.Errorf("mark task delete conflict: update task: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("mark task delete conflict: read affected rows: %w", err)
	}
	if affected != 1 {
		return ErrTaskVersionMismatch
	}

	return nil
}
