package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// SyncProject describes persisted sync metadata for one project.
type SyncProject struct {
	ID           string
	CalendarHref string
	SyncStrategy string
	SyncToken    string
	CTag         string
}

// ListSyncProjects returns all projects with sync metadata.
func (d *Database) ListSyncProjects(ctx context.Context) ([]SyncProject, error) {
	rows, err := d.Conn.QueryContext(ctx, `
SELECT id, calendar_href, sync_strategy, sync_token, ctag
FROM projects
ORDER BY created_at ASC;
`)
	if err != nil {
		return nil, fmt.Errorf("list sync projects: %w", err)
	}
	defer rows.Close()

	projects := make([]SyncProject, 0)
	for rows.Next() {
		var item SyncProject
		var syncToken sql.NullString
		var ctag sql.NullString
		if err := rows.Scan(&item.ID, &item.CalendarHref, &item.SyncStrategy, &syncToken, &ctag); err != nil {
			return nil, fmt.Errorf("list sync projects: scan row: %w", err)
		}
		item.SyncToken = strings.TrimSpace(syncToken.String)
		item.CTag = strings.TrimSpace(ctag.String)
		projects = append(projects, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list sync projects: iterate rows: %w", err)
	}

	return projects, nil
}

// UpdateProjectSyncStrategy stores the effective strategy after fallback evaluation.
func (d *Database) UpdateProjectSyncStrategy(ctx context.Context, projectID string, strategy string) error {
	d.WriteMu.Lock()
	defer d.WriteMu.Unlock()

	result, err := d.Conn.ExecContext(ctx, `
UPDATE projects
SET sync_strategy = ?,
    server_version = server_version + 1,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?;
`, strings.TrimSpace(strategy), strings.TrimSpace(projectID))
	if err != nil {
		return fmt.Errorf("update project sync strategy: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("update project sync strategy: read affected rows: %w", err)
	}
	if affected != 1 {
		return ErrProjectNotFound
	}

	return nil
}
