package db

import (
	"context"
	"fmt"
	"time"
)

// DatedTaskViewRow contains fields rendered in date-based system views.
type DatedTaskViewRow struct {
	ID          string
	Title       string
	Status      string
	ProjectName string
	DueISODate  string
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
	AND (LOWER(COALESCE(t.label_names, '')) LIKE '%starred%')`, limit)
}

// ListNoDateTasks returns active tasks without a due date.
func (d *Database) ListNoDateTasks(ctx context.Context, limit int) ([]DatedTaskViewRow, error) {
	return d.listSimpleSystemTasks(ctx, `
	AND due_iso_date IS NULL`, limit)
}

// ListCompletedTasks returns completed tasks when the visibility setting is enabled.
func (d *Database) ListCompletedTasks(ctx context.Context, limit int) ([]DatedTaskViewRow, error) {
	return d.listSimpleSystemTasks(ctx, `
	AND cfg.show_completed = 1
	AND t.status = 'completed'`, limit)
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
		t.title,
		t.status,
		COALESCE(t.project_name, '') AS project_name,
		COALESCE(
			date(t.due_at),
			date(substr(t.due_at, 1, 19)),
			date(substr(t.due_at, 1, 10)),
			date(t.due_date)
		) AS due_iso_date,
		t.updated_at,
		t.label_names
	FROM tasks t
)
SELECT t.id, t.title, t.status, t.project_name, COALESCE(t.due_iso_date, '')
FROM scoped_tasks t
CROSS JOIN cfg
WHERE 1=1
`+whereSQL+`
ORDER BY t.updated_at DESC
LIMIT ?;`, limit)
	if err != nil {
		return nil, fmt.Errorf("list simple system tasks: %w", err)
	}
	defer rows.Close()

	results := make([]DatedTaskViewRow, 0, limit)
	for rows.Next() {
		var row DatedTaskViewRow
		if err := rows.Scan(&row.ID, &row.Title, &row.Status, &row.ProjectName, &row.DueISODate); err != nil {
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
		t.title,
		t.status,
		COALESCE(t.project_name, '') AS project_name,
		COALESCE(
			date(t.due_at),
			date(substr(t.due_at, 1, 19)),
			date(substr(t.due_at, 1, 10)),
			date(t.due_date)
		) AS due_iso_date,
		t.updated_at
	FROM tasks t
)
SELECT
	t.id,
	t.title,
	t.status,
	t.project_name,
	t.due_iso_date
FROM scoped_tasks t
CROSS JOIN cfg
WHERE
	t.due_iso_date IS NOT NULL
	`+dateFilterSQL+`
	AND (cfg.show_completed = 1 OR t.status != 'completed')
ORDER BY due_iso_date ASC, t.updated_at DESC
LIMIT ?;
`, args...)
	if err != nil {
		return nil, fmt.Errorf("list date scoped tasks: %w", err)
	}
	defer rows.Close()

	results := make([]DatedTaskViewRow, 0, limit)
	for rows.Next() {
		var row DatedTaskViewRow
		if err := rows.Scan(&row.ID, &row.Title, &row.Status, &row.ProjectName, &row.DueISODate); err != nil {
			return nil, fmt.Errorf("list date scoped tasks: scan row: %w", err)
		}
		results = append(results, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list date scoped tasks: iterate rows: %w", err)
	}

	return results, nil
}
