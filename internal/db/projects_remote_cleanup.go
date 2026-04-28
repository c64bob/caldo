package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// RemoteDeletedProjectCleanup describes one local project removed because the remote calendar no longer exists.
type RemoteDeletedProjectCleanup struct {
	ProjectID         string
	CalendarHref      string
	HadPendingTasks   bool
	WasDefaultProject bool
}

// CleanupRemoteDeletedCalendars removes local projects whose calendar href is no longer present remotely.
func (d *Database) CleanupRemoteDeletedCalendars(ctx context.Context, remoteCalendarHrefs []string) ([]RemoteDeletedProjectCleanup, error) {
	normalizedRemote := normalizeCalendarHrefs(remoteCalendarHrefs)

	d.WriteMu.Lock()
	defer d.WriteMu.Unlock()

	tx, err := d.Conn.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("cleanup remote deleted calendars: begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	localProjects, err := loadMissingRemoteProjects(ctx, tx, normalizedRemote)
	if err != nil {
		return nil, fmt.Errorf("cleanup remote deleted calendars: load missing projects: %w", err)
	}
	if len(localProjects) == 0 {
		if err := tx.Commit(); err != nil {
			return nil, fmt.Errorf("cleanup remote deleted calendars: commit empty cleanup: %w", err)
		}
		return nil, nil
	}

	for i := range localProjects {
		if err := cleanupSingleDeletedProject(ctx, tx, localProjects[i].ProjectID); err != nil {
			return nil, fmt.Errorf("cleanup remote deleted calendars: cleanup project %q: %w", localProjects[i].ProjectID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("cleanup remote deleted calendars: commit cleanup: %w", err)
	}

	return localProjects, nil
}

func normalizeCalendarHrefs(hrefs []string) []string {
	normalized := make([]string, 0, len(hrefs))
	seen := make(map[string]struct{}, len(hrefs))
	for _, href := range hrefs {
		trimmed := strings.TrimSpace(href)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		normalized = append(normalized, trimmed)
	}
	return normalized
}

func loadMissingRemoteProjects(ctx context.Context, tx *sql.Tx, remoteHrefs []string) ([]RemoteDeletedProjectCleanup, error) {
	query := `
SELECT p.id, p.calendar_href,
       EXISTS(SELECT 1 FROM tasks t WHERE t.project_id = p.id AND t.sync_status = 'pending') AS has_pending,
       EXISTS(SELECT 1 FROM settings s WHERE s.id = 'default' AND s.default_project_id = p.id) AS is_default
FROM projects p`
	args := make([]any, 0, len(remoteHrefs))
	if len(remoteHrefs) > 0 {
		query += "\nWHERE p.calendar_href NOT IN (" + placeholders(len(remoteHrefs)) + ")"
		for _, href := range remoteHrefs {
			args = append(args, href)
		}
	}
	query += ";"

	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cleanups := make([]RemoteDeletedProjectCleanup, 0)
	for rows.Next() {
		var cleanup RemoteDeletedProjectCleanup
		if err := rows.Scan(&cleanup.ProjectID, &cleanup.CalendarHref, &cleanup.HadPendingTasks, &cleanup.WasDefaultProject); err != nil {
			return nil, err
		}
		cleanups = append(cleanups, cleanup)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return cleanups, nil
}

func cleanupSingleDeletedProject(ctx context.Context, tx *sql.Tx, projectID string) error {
	if _, err := tx.ExecContext(ctx, `
DELETE FROM undo_snapshots
WHERE task_id IN (
	SELECT id FROM tasks WHERE project_id = ?
)
   OR (
	action_type = 'task_deleted'
	AND json_extract(snapshot_fields, '$.project_id') = ?
);
`, projectID, projectID); err != nil {
		return fmt.Errorf("delete undo snapshots: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
DELETE FROM conflicts
WHERE project_id = ?
   OR task_id IN (SELECT id FROM tasks WHERE project_id = ?);
`, projectID, projectID); err != nil {
		return fmt.Errorf("delete conflicts: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
DELETE FROM tasks
WHERE project_id = ?;
`, projectID); err != nil {
		return fmt.Errorf("delete tasks: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
DELETE FROM projects
WHERE id = ?;
`, projectID); err != nil {
		return fmt.Errorf("delete project: %w", err)
	}

	return nil
}

func placeholders(count int) string {
	if count < 1 {
		return ""
	}
	parts := make([]string, count)
	for i := range parts {
		parts[i] = "?"
	}
	return strings.Join(parts, ",")
}
