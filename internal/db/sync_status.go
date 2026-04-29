package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// SyncStatus stores current manual sync state for UI and SSE.
type SyncStatus struct {
	State         string
	LastStartedAt sql.NullTime
	LastFinished  sql.NullTime
	LastSuccessAt sql.NullTime
	LastErrorCode sql.NullString
}

// LoadSyncStatus returns current persisted sync status.
func (d *Database) LoadSyncStatus(ctx context.Context) (SyncStatus, error) {
	var status SyncStatus
	err := d.Conn.QueryRowContext(ctx, `SELECT sync_state, sync_last_started_at, sync_last_finished_at, sync_last_success_at, sync_last_error_code FROM settings WHERE id = 'default'`).Scan(
		&status.State,
		&status.LastStartedAt,
		&status.LastFinished,
		&status.LastSuccessAt,
		&status.LastErrorCode,
	)
	if err != nil {
		return SyncStatus{}, fmt.Errorf("load sync status: %w", err)
	}
	return status, nil
}

// TryStartManualSync marks sync as running unless already running.
func (d *Database) TryStartManualSync(ctx context.Context) (bool, error) {
	d.WriteMu.Lock()
	defer d.WriteMu.Unlock()

	res, err := d.Conn.ExecContext(ctx, `UPDATE settings SET sync_state='running', sync_last_started_at=?, sync_last_error_code=NULL, updated_at=CURRENT_TIMESTAMP WHERE id='default' AND sync_state!='running'`, time.Now().UTC())
	if err != nil {
		return false, fmt.Errorf("start manual sync: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("start manual sync rows affected: %w", err)
	}
	return rows == 1, nil
}

// FinishManualSyncSuccess marks a manual sync as successful.
func (d *Database) FinishManualSyncSuccess(ctx context.Context) error {
	d.WriteMu.Lock()
	defer d.WriteMu.Unlock()
	now := time.Now().UTC()
	if _, err := d.Conn.ExecContext(ctx, `UPDATE settings SET sync_state='idle', sync_last_finished_at=?, sync_last_success_at=?, sync_last_error_code=NULL, updated_at=CURRENT_TIMESTAMP WHERE id='default'`, now, now); err != nil {
		return fmt.Errorf("finish manual sync success: %w", err)
	}
	return nil
}

// FinishManualSyncError marks a manual sync as failed with an error code.
func (d *Database) FinishManualSyncError(ctx context.Context, errorCode string) error {
	d.WriteMu.Lock()
	defer d.WriteMu.Unlock()
	now := time.Now().UTC()
	if _, err := d.Conn.ExecContext(ctx, `UPDATE settings SET sync_state='idle', sync_last_finished_at=?, sync_last_error_code=?, updated_at=CURRENT_TIMESTAMP WHERE id='default'`, now, errorCode); err != nil {
		return fmt.Errorf("finish manual sync error: %w", err)
	}
	return nil
}
