package db

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

const minSyncIntervalMinutes = 5

// SaveSyncInterval updates sync settings in the singleton row.
func (d *Database) SaveSyncInterval(ctx context.Context, syncIntervalMinutes int) error {
	if syncIntervalMinutes < minSyncIntervalMinutes {
		return fmt.Errorf("update sync interval: interval must be at least %d minutes", minSyncIntervalMinutes)
	}

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

// SaveSettingsCalendars stores selected CalDAV calendars as projects and updates the default project.
func (d *Database) SaveSettingsCalendars(ctx context.Context, selected []SelectedCalendar, defaultHref string, strategy string) error {
	if len(selected) == 0 {
		return fmt.Errorf("save settings calendars: at least one calendar is required")
	}
	if strings.TrimSpace(defaultHref) == "" {
		return fmt.Errorf("save settings calendars: default project is required")
	}

	selectedByHref := make(map[string]SelectedCalendar, len(selected))
	for _, calendar := range selected {
		href := strings.TrimSpace(calendar.Href)
		if href == "" {
			return fmt.Errorf("save settings calendars: calendar href is required")
		}
		displayName := strings.TrimSpace(calendar.DisplayName)
		if displayName == "" {
			displayName = href
		}
		calendar.Href = href
		calendar.DisplayName = displayName
		selectedByHref[href] = calendar
	}
	defaultCalendar, ok := selectedByHref[strings.TrimSpace(defaultHref)]
	if !ok {
		return fmt.Errorf("save settings calendars: default project must be selected")
	}

	syncStrategy := strings.TrimSpace(strategy)
	if syncStrategy == "" {
		syncStrategy = "fullscan"
	}

	d.WriteMu.Lock()
	defer d.WriteMu.Unlock()

	tx, err := d.Conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("save settings calendars: begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	projectIDsByHref := make(map[string]string, len(selectedByHref))
	rows, err := tx.QueryContext(ctx, `SELECT id, calendar_href FROM projects;`)
	if err != nil {
		return fmt.Errorf("save settings calendars: load existing projects: %w", err)
	}
	for rows.Next() {
		var id, href string
		if err := rows.Scan(&id, &href); err != nil {
			_ = rows.Close()
			return fmt.Errorf("save settings calendars: scan existing project: %w", err)
		}
		projectIDsByHref[href] = id
	}
	if err := rows.Close(); err != nil {
		return fmt.Errorf("save settings calendars: close project rows: %w", err)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("save settings calendars: iterate existing projects: %w", err)
	}

	for _, calendar := range selectedByHref {
		if projectID, exists := projectIDsByHref[calendar.Href]; exists {
			if _, err := tx.ExecContext(ctx, `
UPDATE projects
SET display_name = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?;
`, calendar.DisplayName, projectID); err != nil {
				return fmt.Errorf("save settings calendars: update existing project: %w", err)
			}
			if _, err := tx.ExecContext(ctx, `
UPDATE tasks
SET project_name = ?
WHERE project_id = ?;
`, calendar.DisplayName, projectID); err != nil {
				return fmt.Errorf("save settings calendars: update task project names: %w", err)
			}
			continue
		}

		projectID := uuid.NewString()
		projectIDsByHref[calendar.Href] = projectID
		if _, err := tx.ExecContext(ctx, `
INSERT INTO projects (
    id, calendar_href, display_name, sync_strategy, is_default, created_at, updated_at
) VALUES (?, ?, ?, ?, FALSE, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
`, projectID, calendar.Href, calendar.DisplayName, syncStrategy); err != nil {
			return fmt.Errorf("save settings calendars: insert project: %w", err)
		}
	}

	selectedHrefs := make(map[string]struct{}, len(selectedByHref))
	for href := range selectedByHref {
		selectedHrefs[href] = struct{}{}
	}
	existingRows, err := tx.QueryContext(ctx, `
SELECT p.id, p.calendar_href, COUNT(t.id)
FROM projects p
LEFT JOIN tasks t ON t.project_id = p.id
GROUP BY p.id, p.calendar_href;
`)
	if err != nil {
		return fmt.Errorf("save settings calendars: load removable projects: %w", err)
	}
	for existingRows.Next() {
		var projectID, href string
		var taskCount int
		if err := existingRows.Scan(&projectID, &href, &taskCount); err != nil {
			_ = existingRows.Close()
			return fmt.Errorf("save settings calendars: scan removable project: %w", err)
		}
		if _, selected := selectedHrefs[href]; selected || taskCount > 0 {
			continue
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM projects WHERE id = ?;`, projectID); err != nil {
			_ = existingRows.Close()
			return fmt.Errorf("save settings calendars: remove unselected empty project: %w", err)
		}
		delete(projectIDsByHref, href)
	}
	if err := existingRows.Close(); err != nil {
		return fmt.Errorf("save settings calendars: close removable project rows: %w", err)
	}
	if err := existingRows.Err(); err != nil {
		return fmt.Errorf("save settings calendars: iterate removable projects: %w", err)
	}

	defaultProjectID := projectIDsByHref[defaultCalendar.Href]
	if strings.TrimSpace(defaultProjectID) == "" {
		return fmt.Errorf("save settings calendars: default project not found")
	}

	if _, err := tx.ExecContext(ctx, `UPDATE projects SET is_default = CASE WHEN id = ? THEN TRUE ELSE FALSE END, updated_at = CURRENT_TIMESTAMP;`, defaultProjectID); err != nil {
		return fmt.Errorf("save settings calendars: update project defaults: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
UPDATE settings
SET default_project_id = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = 'default';
`, defaultProjectID); err != nil {
		return fmt.Errorf("save settings calendars: update settings: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("save settings calendars: commit transaction: %w", err)
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
