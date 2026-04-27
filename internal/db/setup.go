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
