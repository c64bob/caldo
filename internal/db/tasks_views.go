package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// DatedTaskViewRow contains fields rendered in date-based system views.
type DatedTaskViewRow struct {
	ID                   string
	ProjectID            string
	Title                string
	Description          string
	Status               string
	ProjectName          string
	DueISODate           string
	Priority             int
	HasPriority          bool
	LabelNames           string
	SyncStatus           string
	ServerVersion        int
	ParentID             string
	ParentTitle          string
	IsSubtask            bool
	SubtaskCount         int
	OpenSubtaskCount     int
	UnresolvedConflictID string
	RawVTODO             string
}

// ListTodayTasks returns tasks due today plus overdue tasks.
func (d *Database) ListTodayTasks(ctx context.Context, referenceDate time.Time, limit int) ([]DatedTaskViewRow, error) {
	return d.listDateScopedTasks(ctx, `
	AND due_iso_date <= date(?)`, referenceDate, limit, 1)
}

// ListUpcomingTasks returns tasks due in the configured upcoming window, excluding today.
func (d *Database) ListUpcomingTasks(ctx context.Context, referenceDate time.Time, limit int) ([]DatedTaskViewRow, error) {
	return d.listDateScopedTasks(ctx, `
	AND due_iso_date > date(?)
	AND due_iso_date <= date(?, '+' || cfg.upcoming_days || ' days')`, referenceDate, limit, 2)
}

// ListOverdueTasks returns tasks that are overdue.
func (d *Database) ListOverdueTasks(ctx context.Context, referenceDate time.Time, limit int) ([]DatedTaskViewRow, error) {
	return d.listDateScopedTasks(ctx, `
	AND due_iso_date < date(?)`, referenceDate, limit, 1)
}

// ListFavoriteTasks returns active favorite tasks.
func (d *Database) ListFavoriteTasks(ctx context.Context, limit int) ([]DatedTaskViewRow, error) {
	return d.listSimpleSystemTasks(ctx, `
	AND (LOWER(COALESCE(t.label_names, '')) LIKE '%starred%')
	AND (cfg.show_completed = 1 OR t.status != 'completed')`, limit)
}

// ListNoDateTasks returns active tasks without a due date.
func (d *Database) ListNoDateTasks(ctx context.Context, limit int) ([]DatedTaskViewRow, error) {
	return d.listSimpleSystemTasks(ctx, `
	AND due_iso_date IS NULL
	AND (cfg.show_completed = 1 OR t.status != 'completed')`, limit)
}

// ListCompletedTasks returns completed tasks when the visibility setting is enabled.
func (d *Database) ListCompletedTasks(ctx context.Context, limit int) ([]DatedTaskViewRow, error) {
	return d.listSimpleSystemTasks(ctx, `
	AND cfg.show_completed = 1
	AND t.status = 'completed'`, limit)
}

// ListProjectTasks returns tasks for one project, hiding completed tasks unless enabled.
func (d *Database) ListProjectTasks(ctx context.Context, projectID string, limit int) ([]DatedTaskViewRow, error) {
	if limit <= 0 {
		limit = 200
	}

	rows, err := d.Conn.QueryContext(ctx, `
WITH cfg AS (
	SELECT show_completed
	FROM settings
	WHERE id = 'default'
),
scoped_tasks AS (
	SELECT
		t.id,
		t.project_id,
		t.title,
		COALESCE(t.description, '') AS description,
		t.status,
		COALESCE(t.project_name, '') AS project_name,
		COALESCE(
			date(t.due_at),
			date(substr(t.due_at, 1, 19)),
			date(substr(t.due_at, 1, 10)),
			date(t.due_date)
		) AS due_iso_date,
		t.priority,
		t.updated_at,
		t.label_names,
		t.sync_status,
		t.server_version,
		t.parent_id,
		COALESCE(parent.title, '') AS parent_title,
		(SELECT COUNT(1) FROM tasks child WHERE child.parent_id = t.id) AS subtask_count,
		(SELECT COUNT(1) FROM tasks child WHERE child.parent_id = t.id AND LOWER(child.status) != 'completed') AS open_subtask_count,
		COALESCE((SELECT c.id FROM conflicts c WHERE c.task_id = t.id AND c.resolved_at IS NULL ORDER BY c.created_at DESC LIMIT 1), '') AS unresolved_conflict_id,
		COALESCE(t.raw_vtodo, '') AS raw_vtodo
	FROM tasks t
	LEFT JOIN tasks parent ON parent.id = t.parent_id
	WHERE t.project_id = ?
)
SELECT
	t.id,
	t.project_id,
	t.title,
	t.description,
	t.status,
	t.project_name,
	COALESCE(t.due_iso_date, ''),
	COALESCE(t.priority, 0),
	t.priority IS NOT NULL,
	COALESCE(t.label_names, ''),
	t.sync_status,
	t.server_version,
	COALESCE(t.parent_id, ''),
	t.parent_title,
	t.parent_id IS NOT NULL,
	t.subtask_count,
	t.open_subtask_count,
	t.unresolved_conflict_id,
	t.raw_vtodo
FROM scoped_tasks t
CROSS JOIN cfg
WHERE cfg.show_completed = 1 OR LOWER(t.status) != 'completed'
ORDER BY
	CASE WHEN t.due_iso_date IS NULL THEN 1 ELSE 0 END,
	t.due_iso_date ASC,
	COALESCE(t.parent_id, t.id),
	CASE WHEN t.parent_id IS NULL THEN 0 ELSE 1 END,
	t.updated_at DESC
LIMIT ?;`, projectID, limit)
	if err != nil {
		return nil, fmt.Errorf("list project tasks: %w", err)
	}
	defer rows.Close()

	results := make([]DatedTaskViewRow, 0, limit)
	for rows.Next() {
		var row DatedTaskViewRow
		if err := rows.Scan(&row.ID, &row.ProjectID, &row.Title, &row.Description, &row.Status, &row.ProjectName, &row.DueISODate, &row.Priority, &row.HasPriority, &row.LabelNames, &row.SyncStatus, &row.ServerVersion, &row.ParentID, &row.ParentTitle, &row.IsSubtask, &row.SubtaskCount, &row.OpenSubtaskCount, &row.UnresolvedConflictID, &row.RawVTODO); err != nil {
			return nil, fmt.Errorf("list project tasks: scan row: %w", err)
		}
		results = append(results, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list project tasks: iterate rows: %w", err)
	}

	return results, nil
}

// ListLabelTasks returns tasks for one label, hiding completed tasks unless enabled.
func (d *Database) ListLabelTasks(ctx context.Context, labelID string, limit int) ([]DatedTaskViewRow, error) {
	if limit <= 0 {
		limit = 200
	}

	rows, err := d.Conn.QueryContext(ctx, `
WITH cfg AS (
	SELECT show_completed
	FROM settings
	WHERE id = 'default'
),
scoped_tasks AS (
	SELECT
		t.id,
		t.project_id,
		t.title,
		COALESCE(t.description, '') AS description,
		t.status,
		COALESCE(t.project_name, '') AS project_name,
		COALESCE(
			date(t.due_at),
			date(substr(t.due_at, 1, 19)),
			date(substr(t.due_at, 1, 10)),
			date(t.due_date)
		) AS due_iso_date,
		t.priority,
		t.updated_at,
		t.label_names,
		t.sync_status,
		t.server_version,
		t.parent_id,
		COALESCE(parent.title, '') AS parent_title,
		(SELECT COUNT(1) FROM tasks child WHERE child.parent_id = t.id) AS subtask_count,
		(SELECT COUNT(1) FROM tasks child WHERE child.parent_id = t.id AND LOWER(child.status) != 'completed') AS open_subtask_count,
		COALESCE((SELECT c.id FROM conflicts c WHERE c.task_id = t.id AND c.resolved_at IS NULL ORDER BY c.created_at DESC LIMIT 1), '') AS unresolved_conflict_id,
		COALESCE(t.raw_vtodo, '') AS raw_vtodo
	FROM tasks t
	JOIN task_labels tl ON tl.task_id = t.id AND tl.label_id = ?
	LEFT JOIN tasks parent ON parent.id = t.parent_id
)
SELECT
	t.id,
	t.project_id,
	t.title,
	t.description,
	t.status,
	t.project_name,
	COALESCE(t.due_iso_date, ''),
	COALESCE(t.priority, 0),
	t.priority IS NOT NULL,
	COALESCE(t.label_names, ''),
	t.sync_status,
	t.server_version,
	COALESCE(t.parent_id, ''),
	t.parent_title,
	t.parent_id IS NOT NULL,
	t.subtask_count,
	t.open_subtask_count,
	t.unresolved_conflict_id,
	t.raw_vtodo
FROM scoped_tasks t
CROSS JOIN cfg
WHERE cfg.show_completed = 1 OR LOWER(t.status) != 'completed'
ORDER BY
	CASE WHEN t.due_iso_date IS NULL THEN 1 ELSE 0 END,
	t.due_iso_date ASC,
	COALESCE(t.parent_id, t.id),
	CASE WHEN t.parent_id IS NULL THEN 0 ELSE 1 END,
	t.updated_at DESC
LIMIT ?;`, labelID, limit)
	if err != nil {
		return nil, fmt.Errorf("list label tasks: %w", err)
	}
	defer rows.Close()

	results := make([]DatedTaskViewRow, 0, limit)
	for rows.Next() {
		var row DatedTaskViewRow
		if err := rows.Scan(&row.ID, &row.ProjectID, &row.Title, &row.Description, &row.Status, &row.ProjectName, &row.DueISODate, &row.Priority, &row.HasPriority, &row.LabelNames, &row.SyncStatus, &row.ServerVersion, &row.ParentID, &row.ParentTitle, &row.IsSubtask, &row.SubtaskCount, &row.OpenSubtaskCount, &row.UnresolvedConflictID, &row.RawVTODO); err != nil {
			return nil, fmt.Errorf("list label tasks: scan row: %w", err)
		}
		results = append(results, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list label tasks: iterate rows: %w", err)
	}

	return results, nil
}

// LoadTaskView returns one task row for fragment rendering.
func (d *Database) LoadTaskView(ctx context.Context, taskID string) (DatedTaskViewRow, error) {
	var row DatedTaskViewRow
	err := d.Conn.QueryRowContext(ctx, `
SELECT
	t.id,
	t.project_id,
	t.title,
	COALESCE(t.description, ''),
	t.status,
	COALESCE(t.project_name, ''),
	COALESCE(
		date(t.due_at),
		date(substr(t.due_at, 1, 19)),
		date(substr(t.due_at, 1, 10)),
		date(t.due_date),
		''
	),
	COALESCE(t.priority, 0),
	t.priority IS NOT NULL,
	COALESCE(t.label_names, ''),
	t.sync_status,
	t.server_version,
	COALESCE(t.parent_id, ''),
	COALESCE(parent.title, ''),
	t.parent_id IS NOT NULL,
	(SELECT COUNT(1) FROM tasks child WHERE child.parent_id = t.id),
	(SELECT COUNT(1) FROM tasks child WHERE child.parent_id = t.id AND child.status != 'completed'),
	COALESCE((SELECT c.id FROM conflicts c WHERE c.task_id = t.id AND c.resolved_at IS NULL ORDER BY c.created_at DESC LIMIT 1), ''),
	COALESCE(t.raw_vtodo, '')
FROM tasks t
LEFT JOIN tasks parent ON parent.id = t.parent_id
WHERE t.id = ?;
`, taskID).Scan(&row.ID, &row.ProjectID, &row.Title, &row.Description, &row.Status, &row.ProjectName, &row.DueISODate, &row.Priority, &row.HasPriority, &row.LabelNames, &row.SyncStatus, &row.ServerVersion, &row.ParentID, &row.ParentTitle, &row.IsSubtask, &row.SubtaskCount, &row.OpenSubtaskCount, &row.UnresolvedConflictID, &row.RawVTODO)
	if errors.Is(err, sql.ErrNoRows) {
		return DatedTaskViewRow{}, ErrTaskNotFound
	}
	if err != nil {
		return DatedTaskViewRow{}, fmt.Errorf("load task view: %w", err)
	}
	return row, nil
}

func (d *Database) listSimpleSystemTasks(ctx context.Context, whereSQL string, limit int) ([]DatedTaskViewRow, error) {
	if limit <= 0 {
		limit = 200
	}

	rows, err := d.Conn.QueryContext(ctx, `
WITH cfg AS (
	SELECT show_completed
	FROM settings
	WHERE id = 'default'
),
scoped_tasks AS (
	SELECT
		t.id,
		t.project_id,
		t.title,
		COALESCE(t.description, '') AS description,
		t.status,
		COALESCE(t.project_name, '') AS project_name,
		COALESCE(
			date(t.due_at),
			date(substr(t.due_at, 1, 19)),
			date(substr(t.due_at, 1, 10)),
			date(t.due_date)
		) AS due_iso_date,
		t.priority,
		t.updated_at,
		t.label_names,
		t.sync_status,
		t.server_version,
		t.parent_id,
		COALESCE(parent.title, '') AS parent_title,
		(SELECT COUNT(1) FROM tasks child WHERE child.parent_id = t.id) AS subtask_count,
		(SELECT COUNT(1) FROM tasks child WHERE child.parent_id = t.id AND child.status != 'completed') AS open_subtask_count,
		COALESCE((SELECT c.id FROM conflicts c WHERE c.task_id = t.id AND c.resolved_at IS NULL ORDER BY c.created_at DESC LIMIT 1), '') AS unresolved_conflict_id,
		COALESCE(t.raw_vtodo, '') AS raw_vtodo
	FROM tasks t
	LEFT JOIN tasks parent ON parent.id = t.parent_id
)
SELECT
	t.id,
	t.project_id,
	t.title,
	t.description,
	t.status,
	t.project_name,
	COALESCE(t.due_iso_date, ''),
	COALESCE(t.priority, 0),
	t.priority IS NOT NULL,
	COALESCE(t.label_names, ''),
	t.sync_status,
	t.server_version,
	COALESCE(t.parent_id, ''),
	t.parent_title,
	t.parent_id IS NOT NULL,
	t.subtask_count,
	t.open_subtask_count,
	t.unresolved_conflict_id,
	t.raw_vtodo
FROM scoped_tasks t
CROSS JOIN cfg
WHERE 1=1
`+whereSQL+`
ORDER BY COALESCE(t.parent_id, t.id), CASE WHEN t.parent_id IS NULL THEN 0 ELSE 1 END, t.updated_at DESC
LIMIT ?;`, limit)
	if err != nil {
		return nil, fmt.Errorf("list simple system tasks: %w", err)
	}
	defer rows.Close()

	results := make([]DatedTaskViewRow, 0, limit)
	for rows.Next() {
		var row DatedTaskViewRow
		if err := rows.Scan(&row.ID, &row.ProjectID, &row.Title, &row.Description, &row.Status, &row.ProjectName, &row.DueISODate, &row.Priority, &row.HasPriority, &row.LabelNames, &row.SyncStatus, &row.ServerVersion, &row.ParentID, &row.ParentTitle, &row.IsSubtask, &row.SubtaskCount, &row.OpenSubtaskCount, &row.UnresolvedConflictID, &row.RawVTODO); err != nil {
			return nil, fmt.Errorf("list simple system tasks: scan row: %w", err)
		}
		results = append(results, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list simple system tasks: iterate rows: %w", err)
	}

	return results, nil
}

func (d *Database) listDateScopedTasks(ctx context.Context, dateFilterSQL string, referenceDate time.Time, limit int, dateArgs int) ([]DatedTaskViewRow, error) {
	if limit <= 0 {
		limit = 200
	}

	reference := referenceDate.UTC().Format("2006-01-02")
	args := make([]any, 0, dateArgs+1)
	for i := 0; i < dateArgs; i++ {
		args = append(args, reference)
	}
	args = append(args, limit)

	rows, err := d.Conn.QueryContext(ctx, `
WITH cfg AS (
	SELECT show_completed, upcoming_days
	FROM settings
	WHERE id = 'default'
),
scoped_tasks AS (
	SELECT
		t.id,
		t.project_id,
		t.title,
		COALESCE(t.description, '') AS description,
		t.status,
		COALESCE(t.project_name, '') AS project_name,
		COALESCE(
			date(t.due_at),
			date(substr(t.due_at, 1, 19)),
			date(substr(t.due_at, 1, 10)),
			date(t.due_date)
		) AS due_iso_date,
		t.priority,
		t.updated_at,
		t.label_names,
		t.sync_status,
		t.server_version,
		t.parent_id,
		COALESCE(parent.title, '') AS parent_title,
		(SELECT COUNT(1) FROM tasks child WHERE child.parent_id = t.id) AS subtask_count,
		(SELECT COUNT(1) FROM tasks child WHERE child.parent_id = t.id AND child.status != 'completed') AS open_subtask_count,
		COALESCE((SELECT c.id FROM conflicts c WHERE c.task_id = t.id AND c.resolved_at IS NULL ORDER BY c.created_at DESC LIMIT 1), '') AS unresolved_conflict_id,
		COALESCE(t.raw_vtodo, '') AS raw_vtodo
	FROM tasks t
	LEFT JOIN tasks parent ON parent.id = t.parent_id
)
SELECT
	t.id,
	t.project_id,
	t.title,
	t.description,
	t.status,
	t.project_name,
	COALESCE(t.due_iso_date, ''),
	COALESCE(t.priority, 0),
	t.priority IS NOT NULL,
	COALESCE(t.label_names, ''),
	t.sync_status,
	t.server_version,
	COALESCE(t.parent_id, ''),
	t.parent_title,
	t.parent_id IS NOT NULL,
	t.subtask_count,
	t.open_subtask_count,
	t.unresolved_conflict_id,
	t.raw_vtodo
FROM scoped_tasks t
CROSS JOIN cfg
WHERE
	t.due_iso_date IS NOT NULL
	`+dateFilterSQL+`
	AND (cfg.show_completed = 1 OR t.status != 'completed')
ORDER BY due_iso_date ASC, COALESCE(t.parent_id, t.id), CASE WHEN t.parent_id IS NULL THEN 0 ELSE 1 END, t.updated_at DESC
LIMIT ?;
`, args...)
	if err != nil {
		return nil, fmt.Errorf("list date scoped tasks: %w", err)
	}
	defer rows.Close()

	results := make([]DatedTaskViewRow, 0, limit)
	for rows.Next() {
		var row DatedTaskViewRow
		if err := rows.Scan(&row.ID, &row.ProjectID, &row.Title, &row.Description, &row.Status, &row.ProjectName, &row.DueISODate, &row.Priority, &row.HasPriority, &row.LabelNames, &row.SyncStatus, &row.ServerVersion, &row.ParentID, &row.ParentTitle, &row.IsSubtask, &row.SubtaskCount, &row.OpenSubtaskCount, &row.UnresolvedConflictID, &row.RawVTODO); err != nil {
			return nil, fmt.Errorf("list date scoped tasks: scan row: %w", err)
		}
		results = append(results, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list date scoped tasks: iterate rows: %w", err)
	}

	return results, nil
}
