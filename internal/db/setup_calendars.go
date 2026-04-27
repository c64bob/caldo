package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

// SelectedCalendar represents a calendar chosen in setup for initial project mapping.
type SelectedCalendar struct {
	Href        string
	DisplayName string
}

// SaveSetupCalendars stores selected calendars as projects, sets the default project, and advances setup to import.
func (d *Database) SaveSetupCalendars(ctx context.Context, selected []SelectedCalendar, defaultHref string, strategy string) error {
	if len(selected) == 0 {
		return fmt.Errorf("save setup calendars: at least one calendar is required")
	}
	if defaultHref == "" {
		return fmt.Errorf("save setup calendars: default project is required")
	}

	selectedByHref := make(map[string]SelectedCalendar, len(selected))
	for _, calendar := range selected {
		if calendar.Href == "" {
			return fmt.Errorf("save setup calendars: calendar href is required")
		}
		selectedByHref[calendar.Href] = calendar
	}
	defaultCalendar, ok := selectedByHref[defaultHref]
	if !ok {
		return fmt.Errorf("save setup calendars: default project must be selected")
	}

	d.WriteMu.Lock()
	defer d.WriteMu.Unlock()

	tx, err := d.Conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("save setup calendars: begin transaction: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `DELETE FROM projects;`); err != nil {
		return fmt.Errorf("save setup calendars: clear projects: %w", err)
	}

	projectIDs := make(map[string]string, len(selectedByHref))
	for _, calendar := range selectedByHref {
		projectID := uuid.NewString()
		projectIDs[calendar.Href] = projectID

		if _, err := tx.ExecContext(ctx, `
INSERT INTO projects (
    id, calendar_href, display_name, sync_strategy, is_default, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
`, projectID, calendar.Href, calendar.DisplayName, strategy, calendar.Href == defaultCalendar.Href); err != nil {
			return fmt.Errorf("save setup calendars: insert project: %w", err)
		}
	}

	defaultProjectID := projectIDs[defaultCalendar.Href]
	if _, err := tx.ExecContext(ctx, `
UPDATE settings
SET default_project_id = ?,
    setup_step = 'import',
    updated_at = CURRENT_TIMESTAMP
WHERE id = 'default';
`, defaultProjectID); err != nil {
		return fmt.Errorf("save setup calendars: update settings: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("save setup calendars: commit transaction: %w", err)
	}

	return nil
}

// LoadCalDAVServerCapabilities loads globally detected capability flags from settings.
func (d *Database) LoadCalDAVServerCapabilities(ctx context.Context) (CalDAVServerCapabilities, error) {
	var rawPayload sql.NullString
	if err := d.Conn.QueryRowContext(ctx, `
SELECT caldav_server_capabilities
FROM settings
WHERE id = 'default';
`).Scan(&rawPayload); err != nil {
		return CalDAVServerCapabilities{}, fmt.Errorf("query caldav server capabilities: %w", err)
	}

	if !rawPayload.Valid || rawPayload.String == "" {
		return CalDAVServerCapabilities{FullScan: true}, nil
	}

	var capabilities CalDAVServerCapabilities
	if err := json.Unmarshal([]byte(rawPayload.String), &capabilities); err != nil {
		return CalDAVServerCapabilities{}, fmt.Errorf("unmarshal caldav server capabilities: %w", err)
	}

	if !capabilities.WebDAVSync && !capabilities.CTag && !capabilities.FullScan {
		capabilities.FullScan = true
	}

	return capabilities, nil
}
