package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"caldo/internal/model"
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
	ConflictID    string
	Resolution    string
	ResolvedVTODO string
	NewETag       string
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
	if _, err := tx.ExecContext(ctx, `
UPDATE tasks
SET raw_vtodo=?, title=?, description=?, status=?, due_date=?, due_at=?, priority=?, label_names=?, etag=?, sync_status='synced', server_version=server_version+1, updated_at=CURRENT_TIMESTAMP
WHERE id=(SELECT task_id FROM conflicts WHERE id=?);
`, input.ResolvedVTODO, parsed.Title, nullableString(parsed.Description), parsed.Status, nullValue(dueDateNull(parsed.DueDate)), nullValue(dueAtNull(parsed.DueAt)), nullValue(priorityNull(parsed.Priority)), nullValue(labelsNull(parsed.Categories)), nullableString(strings.TrimSpace(input.NewETag)), input.ConflictID); err != nil {
		return fmt.Errorf("mark conflict resolved: update task: %w", err)
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
