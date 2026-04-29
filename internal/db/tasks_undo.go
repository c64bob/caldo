package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

var (
	// ErrUndoSnapshotNotFound indicates no undo snapshot exists for session/tab.
	ErrUndoSnapshotNotFound = errors.New("undo snapshot not found")
	// ErrUndoSnapshotExpired indicates the undo snapshot has expired.
	ErrUndoSnapshotExpired = errors.New("undo snapshot expired")
	// ErrUndoETagMismatch indicates current task etag no longer matches snapshot etag.
	ErrUndoETagMismatch = errors.New("undo etag mismatch")
	// ErrUndoActionNotSupported indicates the snapshot action is not yet undoable.
	ErrUndoActionNotSupported = errors.New("undo action not supported")
)

type PreparedTaskUndo struct {
	TaskID         string
	ActionType     string
	TodoHref       string
	ExpectedETag   string
	RawVTODO       string
	PendingVersion int
}

// PrepareTaskUndo loads and validates the latest undo snapshot for a session/tab and marks task as pending.
func (d *Database) PrepareTaskUndo(ctx context.Context, sessionID, tabID string) (PreparedTaskUndo, error) {
	d.WriteMu.Lock()
	defer d.WriteMu.Unlock()

	tx, err := d.Conn.BeginTx(ctx, nil)
	if err != nil {
		return PreparedTaskUndo{}, fmt.Errorf("prepare task undo: begin transaction: %w", err)
	}
	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()

	var snapshotID, taskID, actionType, snapshotVTODO string
	var etagAtSnapshot sql.NullString
	var isExpired bool
	if err := tx.QueryRowContext(ctx, `
SELECT id, task_id, action_type, snapshot_vtodo, etag_at_snapshot, expires_at <= CURRENT_TIMESTAMP
FROM undo_snapshots
WHERE session_id = ? AND tab_id = ?;
`, sessionID, tabID).Scan(&snapshotID, &taskID, &actionType, &snapshotVTODO, &etagAtSnapshot, &isExpired); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return PreparedTaskUndo{}, ErrUndoSnapshotNotFound
		}
		return PreparedTaskUndo{}, fmt.Errorf("prepare task undo: load snapshot: %w", err)
	}

	if isExpired {
		if _, err := tx.ExecContext(ctx, `DELETE FROM undo_snapshots WHERE id = ?;`, snapshotID); err != nil {
			return PreparedTaskUndo{}, fmt.Errorf("prepare task undo: delete expired snapshot: %w", err)
		}
		if err := tx.Commit(); err != nil {
			return PreparedTaskUndo{}, fmt.Errorf("prepare task undo: commit expired snapshot delete: %w", err)
		}
		tx = nil
		return PreparedTaskUndo{}, ErrUndoSnapshotExpired
	}

	if actionType != "task_updated" {
		return PreparedTaskUndo{}, ErrUndoActionNotSupported
	}

	var currentETag sql.NullString
	var href string
	var version int
	if err := tx.QueryRowContext(ctx, `SELECT etag, href, server_version FROM tasks WHERE id = ?;`, taskID).Scan(&currentETag, &href, &version); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return PreparedTaskUndo{}, ErrTaskNotFound
		}
		return PreparedTaskUndo{}, fmt.Errorf("prepare task undo: load current task: %w", err)
	}

	if nullableString(currentETag.String) != nullableString(etagAtSnapshot.String) {
		if _, err := tx.ExecContext(ctx, `UPDATE tasks SET sync_status = 'conflict', updated_at = CURRENT_TIMESTAMP WHERE id = ?;`, taskID); err != nil {
			return PreparedTaskUndo{}, fmt.Errorf("prepare task undo: mark task conflict: %w", err)
		}
		if err := tx.Commit(); err != nil {
			return PreparedTaskUndo{}, fmt.Errorf("prepare task undo: commit conflict state: %w", err)
		}
		tx = nil
		return PreparedTaskUndo{}, ErrUndoETagMismatch
	}

	result, err := tx.ExecContext(ctx, `
UPDATE tasks
SET raw_vtodo = ?,
    sync_status = 'pending',
    server_version = server_version + 1,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ? AND server_version = ?;
`, snapshotVTODO, taskID, version)
	if err != nil {
		return PreparedTaskUndo{}, fmt.Errorf("prepare task undo: update pending task: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return PreparedTaskUndo{}, fmt.Errorf("prepare task undo: read affected rows: %w", err)
	}
	if affected != 1 {
		return PreparedTaskUndo{}, ErrTaskVersionMismatch
	}

	if err := tx.Commit(); err != nil {
		return PreparedTaskUndo{}, fmt.Errorf("prepare task undo: commit transaction: %w", err)
	}
	tx = nil

	return PreparedTaskUndo{TaskID: taskID, ActionType: actionType, TodoHref: href, ExpectedETag: currentETag.String, RawVTODO: snapshotVTODO, PendingVersion: version + 1}, nil
}

// DeleteUndoSnapshot deletes a single undo snapshot by session/tab.
func (d *Database) DeleteUndoSnapshot(ctx context.Context, sessionID, tabID string) error {
	d.WriteMu.Lock()
	defer d.WriteMu.Unlock()
	_, err := d.Conn.ExecContext(ctx, `DELETE FROM undo_snapshots WHERE session_id = ? AND tab_id = ?;`, sessionID, tabID)
	if err != nil {
		return fmt.Errorf("delete undo snapshot: %w", err)
	}
	return nil
}
