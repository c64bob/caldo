package db

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

const (
	conflictTypeFieldConflict = "field_conflict"
	conflictTypeEditDelete    = "edit_delete"
	conflictTypeDeleteEdit    = "delete_edit"
)

// RecordRemoteFieldConflict stores a remote sync conflict and blocks the affected task.
func (d *Database) RecordRemoteFieldConflict(ctx context.Context, taskID string, expectedVersion int, baseVTODO string, localVTODO string, remoteVTODO string) error {
	return d.recordRemoteConflict(ctx, taskID, expectedVersion, conflictTypeFieldConflict, baseVTODO, localVTODO, remoteVTODO)
}

// RecordRemoteEditDeleteConflict stores a conflict where local edits meet a remote deletion.
func (d *Database) RecordRemoteEditDeleteConflict(ctx context.Context, taskID string, expectedVersion int, baseVTODO string, localVTODO string) error {
	return d.recordRemoteConflict(ctx, taskID, expectedVersion, conflictTypeEditDelete, baseVTODO, localVTODO, "")
}

// RecordRemoteDeleteEditConflict stores a conflict where a local deletion meets a remote edit.
func (d *Database) RecordRemoteDeleteEditConflict(ctx context.Context, baseVTODO string, remoteVTODO string) error {
	d.WriteMu.Lock()
	defer d.WriteMu.Unlock()

	if _, err := d.Conn.ExecContext(ctx, `
INSERT INTO conflicts (id, task_id, project_id, conflict_type, created_at, base_vtodo, local_vtodo, remote_vtodo)
VALUES (?, NULL, NULL, ?, CURRENT_TIMESTAMP, ?, NULL, ?);
`, uuid.NewString(), conflictTypeDeleteEdit, baseVTODO, remoteVTODO); err != nil {
		return fmt.Errorf("record remote field conflict: insert conflict: %w", err)
	}

	return nil
}

func (d *Database) recordRemoteConflict(ctx context.Context, taskID string, expectedVersion int, conflictType string, baseVTODO string, localVTODO string, remoteVTODO string) error {
	d.WriteMu.Lock()
	defer d.WriteMu.Unlock()

	tx, err := d.Conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("record remote field conflict: begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	result, err := tx.ExecContext(ctx, `UPDATE tasks SET sync_status='conflict', updated_at=CURRENT_TIMESTAMP WHERE id=? AND server_version=?;`, taskID, expectedVersion)
	if err != nil {
		return fmt.Errorf("record remote field conflict: update task: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("record remote field conflict: read affected rows: %w", err)
	}
	if affected != 1 {
		return ErrTaskVersionMismatch
	}

	if _, err := tx.ExecContext(ctx, `
INSERT INTO conflicts (id, task_id, project_id, conflict_type, created_at, base_vtodo, local_vtodo, remote_vtodo)
SELECT ?, t.id, t.project_id, ?, CURRENT_TIMESTAMP, ?, NULLIF(?, ''), NULLIF(?, '')
FROM tasks t WHERE t.id = ?;
`, uuid.NewString(), conflictType, baseVTODO, localVTODO, remoteVTODO, taskID); err != nil {
		return fmt.Errorf("record remote field conflict: insert conflict: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("record remote field conflict: commit transaction: %w", err)
	}
	return nil
}
