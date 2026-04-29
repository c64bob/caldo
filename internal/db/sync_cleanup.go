package db

import (
	"context"
	"fmt"
	"time"
)

// SyncCleanupResult contains row counts for one cleanup run.
type SyncCleanupResult struct {
	ExpiredUndoDeleted       int64
	ResolvedConflictsDeleted int64
}

// CleanupSyncArtifacts removes expired undo snapshots and, optionally, old resolved conflicts.
func (d *Database) CleanupSyncArtifacts(ctx context.Context, now time.Time, cleanupResolvedConflicts bool) (SyncCleanupResult, error) {
	d.WriteMu.Lock()
	defer d.WriteMu.Unlock()

	tx, err := d.Conn.BeginTx(ctx, nil)
	if err != nil {
		return SyncCleanupResult{}, fmt.Errorf("cleanup sync artifacts: begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	result := SyncCleanupResult{}

	undoRes, err := tx.ExecContext(ctx, `DELETE FROM undo_snapshots WHERE expires_at < ?`, now.UTC())
	if err != nil {
		return SyncCleanupResult{}, fmt.Errorf("cleanup sync artifacts: delete expired undo snapshots: %w", err)
	}
	result.ExpiredUndoDeleted, err = undoRes.RowsAffected()
	if err != nil {
		return SyncCleanupResult{}, fmt.Errorf("cleanup sync artifacts: undo rows affected: %w", err)
	}

	if cleanupResolvedConflicts {
		conflictRes, execErr := tx.ExecContext(ctx, `
DELETE FROM conflicts
WHERE resolved_at IS NOT NULL
  AND resolved_at < DATETIME(CURRENT_TIMESTAMP, '-7 days');
`)
		if execErr != nil {
			return SyncCleanupResult{}, fmt.Errorf("cleanup sync artifacts: delete resolved conflicts: %w", execErr)
		}
		result.ResolvedConflictsDeleted, err = conflictRes.RowsAffected()
		if err != nil {
			return SyncCleanupResult{}, fmt.Errorf("cleanup sync artifacts: conflict rows affected: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return SyncCleanupResult{}, fmt.Errorf("cleanup sync artifacts: commit transaction: %w", err)
	}

	return result, nil
}
