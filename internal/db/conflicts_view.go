package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// ConflictListRow represents one unresolved conflict in the global conflict list.
type ConflictListRow struct {
	ID           string
	TaskID       sql.NullString
	ProjectID    sql.NullString
	ProjectName  string
	ConflictType string
	CreatedAt    time.Time
	TaskTitle    string
}

// ConflictDetail contains all data needed to render one conflict detail page.
type ConflictDetail struct {
	ID           string
	TaskID       sql.NullString
	ProjectID    sql.NullString
	ProjectName  string
	ConflictType string
	CreatedAt    time.Time
	BaseVTODO    sql.NullString
	LocalVTODO   sql.NullString
	RemoteVTODO  sql.NullString
}

// ListUnresolvedConflicts returns unresolved conflicts ordered newest first.
func (d *Database) ListUnresolvedConflicts(ctx context.Context) ([]ConflictListRow, error) {
	rows, err := d.Conn.QueryContext(ctx, `
SELECT c.id,
       c.task_id,
       c.project_id,
       COALESCE(p.display_name, ''),
       c.conflict_type,
       c.created_at,
       COALESCE(t.title, '(gelöschte Aufgabe)')
FROM conflicts c
LEFT JOIN projects p ON p.id = c.project_id
LEFT JOIN tasks t ON t.id = c.task_id
WHERE c.resolved_at IS NULL
ORDER BY c.created_at DESC
;
`)
	if err != nil {
		return nil, fmt.Errorf("list unresolved conflicts: %w", err)
	}
	defer rows.Close()

	results := make([]ConflictListRow, 0)
	for rows.Next() {
		var row ConflictListRow
		if err := rows.Scan(&row.ID, &row.TaskID, &row.ProjectID, &row.ProjectName, &row.ConflictType, &row.CreatedAt, &row.TaskTitle); err != nil {
			return nil, fmt.Errorf("list unresolved conflicts: scan row: %w", err)
		}
		results = append(results, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list unresolved conflicts: iterate rows: %w", err)
	}

	return results, nil
}

// GetUnresolvedConflictByID returns one unresolved conflict by id.
func (d *Database) GetUnresolvedConflictByID(ctx context.Context, id string) (ConflictDetail, error) {
	var detail ConflictDetail
	row := d.Conn.QueryRowContext(ctx, `
SELECT c.id,
       c.task_id,
       c.project_id,
       COALESCE(p.display_name, ''),
       c.conflict_type,
       c.created_at,
       c.base_vtodo,
       c.local_vtodo,
       c.remote_vtodo
FROM conflicts c
LEFT JOIN projects p ON p.id = c.project_id
WHERE c.id = ? AND c.resolved_at IS NULL;
`, id)
	if err := row.Scan(&detail.ID, &detail.TaskID, &detail.ProjectID, &detail.ProjectName, &detail.ConflictType, &detail.CreatedAt, &detail.BaseVTODO, &detail.LocalVTODO, &detail.RemoteVTODO); err != nil {
		return ConflictDetail{}, fmt.Errorf("get unresolved conflict by id: %w", err)
	}

	return detail, nil
}
