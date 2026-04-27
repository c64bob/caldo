package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// ErrSetupPrerequisitesNotMet indicates setup completion prerequisites are not satisfied.
var ErrSetupPrerequisitesNotMet = errors.New("setup prerequisites not met")

// SetupStatus contains persisted setup gate state from the settings singleton.
type SetupStatus struct {
	Complete bool
	Step     string
}

// LoadSetupStatus returns the persisted setup status from the settings singleton.
func (d *Database) LoadSetupStatus(ctx context.Context) (SetupStatus, error) {
	var status SetupStatus
	if err := d.Conn.QueryRowContext(ctx, `
SELECT setup_complete, setup_step
FROM settings
WHERE id = 'default';
`).Scan(&status.Complete, &status.Step); err != nil {
		if err == sql.ErrNoRows {
			return SetupStatus{}, fmt.Errorf("query setup status: settings singleton missing")
		}
		return SetupStatus{}, fmt.Errorf("query setup status: %w", err)
	}

	return status, nil
}

// SaveSetupStep persists the current setup step in the settings singleton.
func (d *Database) SaveSetupStep(ctx context.Context, step string) error {
	d.WriteMu.Lock()
	defer d.WriteMu.Unlock()

	result, err := d.Conn.ExecContext(ctx, `
UPDATE settings
SET setup_step = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = 'default';
`, step)
	if err != nil {
		return fmt.Errorf("update setup step: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read affected rows: %w", err)
	}
	if affected != 1 {
		return fmt.Errorf("update setup step: expected 1 row affected, got %d", affected)
	}

	return nil
}

// CompleteSetup marks setup as complete after validating setup prerequisites.
func (d *Database) CompleteSetup(ctx context.Context) error {
	d.WriteMu.Lock()
	defer d.WriteMu.Unlock()

	tx, err := d.Conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("complete setup: begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	var setupStep string
	var defaultProjectID sql.NullString
	if err := tx.QueryRowContext(ctx, `
SELECT setup_step, default_project_id
FROM settings
WHERE id = 'default';
`).Scan(&setupStep, &defaultProjectID); err != nil {
		return fmt.Errorf("complete setup: load settings: %w", err)
	}

	if setupStep != "import" {
		return fmt.Errorf("complete setup: %w", ErrSetupPrerequisitesNotMet)
	}
	if !defaultProjectID.Valid || defaultProjectID.String == "" {
		return fmt.Errorf("complete setup: %w", ErrSetupPrerequisitesNotMet)
	}

	var projectCount int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM projects;`).Scan(&projectCount); err != nil {
		return fmt.Errorf("complete setup: count projects: %w", err)
	}
	if projectCount < 1 {
		return fmt.Errorf("complete setup: %w", ErrSetupPrerequisitesNotMet)
	}

	var defaultProjectCount int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM projects WHERE id = ?;`, defaultProjectID.String).Scan(&defaultProjectCount); err != nil {
		return fmt.Errorf("complete setup: count default project: %w", err)
	}
	if defaultProjectCount != 1 {
		return fmt.Errorf("complete setup: %w", ErrSetupPrerequisitesNotMet)
	}

	var unsyncedTaskCount int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM tasks WHERE sync_status != 'synced';`).Scan(&unsyncedTaskCount); err != nil {
		return fmt.Errorf("complete setup: count unsynced tasks: %w", err)
	}
	if unsyncedTaskCount > 0 {
		return fmt.Errorf("complete setup: %w", ErrSetupPrerequisitesNotMet)
	}

	result, err := tx.ExecContext(ctx, `
UPDATE settings
SET setup_step = 'complete',
    setup_complete = TRUE,
    updated_at = CURRENT_TIMESTAMP
WHERE id = 'default';
`)
	if err != nil {
		return fmt.Errorf("complete setup: update settings: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("complete setup: read affected rows: %w", err)
	}
	if affected != 1 {
		return fmt.Errorf("complete setup: expected 1 row affected, got %d", affected)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("complete setup: commit transaction: %w", err)
	}

	return nil
}
