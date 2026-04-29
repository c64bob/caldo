package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"caldo/internal/model"
	"github.com/google/uuid"
)

var ErrConflictNotFound = errors.New("conflict not found")

type ConflictResolutionBase struct {
	ConflictID    string
	TaskID        string
	Href          string
	ETag          string
	ServerVersion int
	LocalVTODO    string
	RemoteVTODO   string
}

type ResolveConflictInput struct {
	ConflictID      string
	Resolution      string
	ResolvedVTODO   string
	NewETag         string
	ExpectedVersion int
}

type ResolveConflictSplitInput struct {
	ConflictID      string
	ResolvedVTODO   string
	NewTaskUID      string
	NewTaskHref     string
	NewTaskETag     string
	ExpectedVersion int
}

func (d *Database) LoadConflictResolutionBase(ctx context.Context, conflictID string) (ConflictResolutionBase, error) {
	var out ConflictResolutionBase
	out.ConflictID = conflictID
	var taskID sql.NullString
	row := d.Conn.QueryRowContext(ctx, `
SELECT c.task_id, COALESCE(c.local_vtodo, ''), COALESCE(c.remote_vtodo, ''), COALESCE(t.href, ''), COALESCE(t.etag, ''), COALESCE(t.server_version, 0)
FROM conflicts c
LEFT JOIN tasks t ON t.id = c.task_id
WHERE c.id = ? AND c.resolved_at IS NULL;
`, conflictID)
	if err := row.Scan(&taskID, &out.LocalVTODO, &out.RemoteVTODO, &out.Href, &out.ETag, &out.ServerVersion); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ConflictResolutionBase{}, ErrConflictNotFound
		}
		return ConflictResolutionBase{}, fmt.Errorf("load conflict resolution base: %w", err)
	}
	out.TaskID = taskID.String
	return out, nil
}

func (d *Database) MarkConflictResolved(ctx context.Context, input ResolveConflictInput) error {
	d.WriteMu.Lock()
	defer d.WriteMu.Unlock()

	tx, err := d.Conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("mark conflict resolved: begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	parsed := model.ParseVTODOFields(input.ResolvedVTODO)
	resultTask, err := tx.ExecContext(ctx, `
UPDATE tasks
SET raw_vtodo=?, title=?, description=?, status=?, due_date=?, due_at=?, priority=?, label_names=?, etag=?, sync_status='synced', server_version=server_version+1, updated_at=CURRENT_TIMESTAMP
WHERE id=(SELECT task_id FROM conflicts WHERE id=?) AND server_version=?;
`, input.ResolvedVTODO, parsed.Title, nullableString(parsed.Description), parsed.Status, nullValue(dueDateNull(parsed.DueDate)), nullValue(dueAtNull(parsed.DueAt)), nullValue(priorityNull(parsed.Priority)), nullValue(labelsNull(parsed.Categories)), nullableString(strings.TrimSpace(input.NewETag)), input.ConflictID, input.ExpectedVersion)
	if err != nil {
		return fmt.Errorf("mark conflict resolved: update task: %w", err)
	}
	taskRowsAffected, err := resultTask.RowsAffected()
	if err != nil {
		return fmt.Errorf("mark conflict resolved: task rows affected: %w", err)
	}
	if taskRowsAffected != 1 {
		return fmt.Errorf("mark conflict resolved: %w", ErrConflictNotFound)
	}

	result, err := tx.ExecContext(ctx, `
UPDATE conflicts
SET resolved_at=CURRENT_TIMESTAMP, resolution=?, resolved_vtodo=?
WHERE id=? AND resolved_at IS NULL;
`, input.Resolution, input.ResolvedVTODO, input.ConflictID)
	if err != nil {
		return fmt.Errorf("mark conflict resolved: update conflict: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil || affected != 1 {
		return ErrConflictNotFound
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("mark conflict resolved: commit transaction: %w", err)
	}
	return nil
}

// MarkConflictSplitResolved resolves a conflict by keeping the local task and inserting the remote version as a new task.
func (d *Database) MarkConflictSplitResolved(ctx context.Context, input ResolveConflictSplitInput) error {
	d.WriteMu.Lock()
	defer d.WriteMu.Unlock()

	tx, err := d.Conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("mark conflict split resolved: begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	resultTask, err := tx.ExecContext(ctx, `
UPDATE tasks
SET sync_status='synced', updated_at=CURRENT_TIMESTAMP, server_version=server_version+1
WHERE id=(SELECT task_id FROM conflicts WHERE id=?) AND server_version=?;
`, input.ConflictID, input.ExpectedVersion)
	if err != nil {
		return fmt.Errorf("mark conflict split resolved: update local task: %w", err)
	}
	taskRowsAffected, err := resultTask.RowsAffected()
	if err != nil {
		return fmt.Errorf("mark conflict split resolved: local task rows affected: %w", err)
	}
	if taskRowsAffected != 1 {
		return fmt.Errorf("mark conflict split resolved: %w", ErrConflictNotFound)
	}

	parsed := model.ParseVTODOFields(input.ResolvedVTODO)
	_, err = tx.ExecContext(ctx, `
INSERT INTO tasks (
	id, project_id, uid, href, etag, server_version, title, description, status, completed_at, due_date, due_at, priority, rrule, parent_id, raw_vtodo, base_vtodo, label_names, project_name, sync_status, created_at, updated_at
)
SELECT ?, t.project_id, ?, ?, ?, 1, ?, ?, ?, ?, ?, ?, ?, ?, NULL, ?, ?, ?, t.project_name, 'synced', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
FROM tasks t
WHERE t.id=(SELECT task_id FROM conflicts WHERE id=?);
`, uuid.NewString(), input.NewTaskUID, input.NewTaskHref, nullableString(strings.TrimSpace(input.NewTaskETag)),
	parsed.Title, nullableString(parsed.Description), parsed.Status, nullValue(dueAtNull(parsed.CompletedAt)), nullValue(dueDateNull(parsed.DueDate)),
	nullValue(dueAtNull(parsed.DueAt)), nullValue(priorityNull(parsed.Priority)), nullableString(parsed.RRule), input.ResolvedVTODO, input.ResolvedVTODO, nullValue(labelsNull(parsed.Categories)), input.ConflictID)
	if err != nil {
		return fmt.Errorf("mark conflict split resolved: insert split task: %w", err)
	}

	resultConflict, err := tx.ExecContext(ctx, `
UPDATE conflicts
SET resolved_at=CURRENT_TIMESTAMP, resolution='split', resolved_vtodo=?
WHERE id=? AND resolved_at IS NULL;
`, input.ResolvedVTODO, input.ConflictID)
	if err != nil {
		return fmt.Errorf("mark conflict split resolved: update conflict: %w", err)
	}
	conflictRowsAffected, err := resultConflict.RowsAffected()
	if err != nil {
		return fmt.Errorf("mark conflict split resolved: conflict rows affected: %w", err)
	}
	if conflictRowsAffected != 1 {
		return fmt.Errorf("mark conflict split resolved: %w", ErrConflictNotFound)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("mark conflict split resolved: commit transaction: %w", err)
	}
	return nil
}

func priorityNull(value *int) sql.NullInt64 {
	if value == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: int64(*value), Valid: true}
}

func dueDateNull(value *string) sql.NullString {
	if value == nil || strings.TrimSpace(*value) == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: strings.TrimSpace(*value), Valid: true}
}

func dueAtNull(value *time.Time) sql.NullTime {
	if value == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: value.UTC(), Valid: true}
}

func labelsNull(values []string) sql.NullString {
	if len(values) == 0 {
		return sql.NullString{}
	}
	return sql.NullString{String: strings.Join(values, ","), Valid: true}
}
