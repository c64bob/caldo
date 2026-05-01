package db

import (
	"context"
	"fmt"
)

// SaveSyncInterval updates sync settings in the singleton row.
func (d *Database) SaveSyncInterval(ctx context.Context, syncIntervalMinutes int) error {
	d.WriteMu.Lock()
	defer d.WriteMu.Unlock()

	result, err := d.Conn.ExecContext(ctx, `
UPDATE settings
SET sync_interval_minutes = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = 'default';
`, syncIntervalMinutes)
	if err != nil {
		return fmt.Errorf("update sync interval: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read affected rows: %w", err)
	}
	if affected != 1 {
		return fmt.Errorf("update sync interval: expected 1 row affected, got %d", affected)
	}

	return nil
}

// SaveUISettings updates UI settings in the singleton row.
func (d *Database) SaveUISettings(ctx context.Context, showCompleted bool, upcomingDays int, uiLanguage, darkMode string) error {
	d.WriteMu.Lock()
	defer d.WriteMu.Unlock()

	result, err := d.Conn.ExecContext(ctx, `
UPDATE settings
SET show_completed = ?,
    upcoming_days = ?,
    ui_language = ?,
    dark_mode = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = 'default';
`, showCompleted, upcomingDays, uiLanguage, darkMode)
	if err != nil {
		return fmt.Errorf("update ui settings: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read affected rows: %w", err)
	}
	if affected != 1 {
		return fmt.Errorf("update ui settings: expected 1 row affected, got %d", affected)
	}

	return nil
}
