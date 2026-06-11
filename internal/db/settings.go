package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// AppSettings contains editable settings in normal operation.
type AppSettings struct {
	SyncIntervalMinutes int
	UpcomingDays        int
	ShowCompleted       bool
	UILanguage          string
	DarkMode            string
	CalDAVURL           string
	CalDAVUsername      string
	CalDAVConfigured    bool
	Projects            []SettingsProject
}

// SettingsProject describes one project/calendar mapping shown in settings.
type SettingsProject struct {
	ID            string
	CalendarHref  string
	DisplayName   string
	SyncStrategy  string
	IsDefault     bool
	OpenTaskCount int
	TaskCount     int
}

// UIPreferences contains the persisted UI language and dark-mode setting.
type UIPreferences struct {
	UILanguage string
	DarkMode   string
}

// LoadUIPreferences reads persisted presentation preferences from settings.
func (d *Database) LoadUIPreferences(ctx context.Context) (UIPreferences, error) {
	var preferences UIPreferences
	err := d.Conn.QueryRowContext(ctx, `
SELECT ui_language, dark_mode
FROM settings
WHERE id='default';
`).Scan(&preferences.UILanguage, &preferences.DarkMode)
	if err != nil {
		return UIPreferences{}, fmt.Errorf("load ui preferences: %w", err)
	}
	return preferences, nil
}

// LoadAppSettings reads editable settings from the singleton row.
func (d *Database) LoadAppSettings(ctx context.Context) (AppSettings, error) {
	var s AppSettings
	var caldavURL, caldavUsername sql.NullString
	var caldavPasswordConfigured bool
	err := d.Conn.QueryRowContext(ctx, `
SELECT
	sync_interval_minutes,
	upcoming_days,
	show_completed,
	ui_language,
	dark_mode,
	caldav_url,
	caldav_username,
	caldav_password_enc IS NOT NULL
FROM settings
WHERE id='default';
`).Scan(
		&s.SyncIntervalMinutes,
		&s.UpcomingDays,
		&s.ShowCompleted,
		&s.UILanguage,
		&s.DarkMode,
		&caldavURL,
		&caldavUsername,
		&caldavPasswordConfigured,
	)
	if err != nil {
		return AppSettings{}, fmt.Errorf("load app settings: %w", err)
	}
	s.CalDAVURL = strings.TrimSpace(caldavURL.String)
	s.CalDAVUsername = strings.TrimSpace(caldavUsername.String)
	s.CalDAVConfigured = caldavPasswordConfigured && s.CalDAVURL != "" && s.CalDAVUsername != ""

	projects, err := d.loadSettingsProjects(ctx)
	if err != nil {
		return AppSettings{}, err
	}
	s.Projects = projects

	return s, nil
}

func (d *Database) loadSettingsProjects(ctx context.Context) ([]SettingsProject, error) {
	rows, err := d.Conn.QueryContext(ctx, `
SELECT
	p.id,
	p.calendar_href,
	p.display_name,
	p.sync_strategy,
	p.is_default,
	COUNT(CASE WHEN t.status != 'completed' THEN 1 END),
	COUNT(t.id)
FROM projects p
LEFT JOIN tasks t ON t.project_id = p.id
GROUP BY p.id, p.calendar_href, p.display_name, p.sync_strategy, p.is_default
ORDER BY p.is_default DESC, p.display_name COLLATE NOCASE ASC;
`)
	if err != nil {
		return nil, fmt.Errorf("load app settings: list projects: %w", err)
	}
	defer rows.Close()

	projects := make([]SettingsProject, 0)
	for rows.Next() {
		var project SettingsProject
		if err := rows.Scan(
			&project.ID,
			&project.CalendarHref,
			&project.DisplayName,
			&project.SyncStrategy,
			&project.IsDefault,
			&project.OpenTaskCount,
			&project.TaskCount,
		); err != nil {
			return nil, fmt.Errorf("load app settings: scan project: %w", err)
		}
		projects = append(projects, project)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("load app settings: iterate projects: %w", err)
	}

	return projects, nil
}
