package db

import (
	"context"
	"database/sql"
	"fmt"
)

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
