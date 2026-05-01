package db

import (
	"context"
	"fmt"
)

// AppSettings contains editable settings in normal operation.
type AppSettings struct {
	SyncIntervalMinutes int
	UpcomingDays        int
	ShowCompleted       bool
	UILanguage          string
	DarkMode            string
}

// LoadAppSettings reads editable settings from the singleton row.
func (d *Database) LoadAppSettings(ctx context.Context) (AppSettings, error) {
	var s AppSettings
	err := d.Conn.QueryRowContext(ctx, `SELECT sync_interval_minutes, upcoming_days, show_completed, ui_language, dark_mode FROM settings WHERE id='default'`).Scan(
		&s.SyncIntervalMinutes,
		&s.UpcomingDays,
		&s.ShowCompleted,
		&s.UILanguage,
		&s.DarkMode,
	)
	if err != nil {
		return AppSettings{}, fmt.Errorf("load app settings: %w", err)
	}
	return s, nil
}
