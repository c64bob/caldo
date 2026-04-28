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
	AND COALESCE(date(t.due_at), date(t.due_date)) <= date(?)`, referenceDate, limit, 1)
}

// ListUpcomingTasks returns tasks due in the configured upcoming window, excluding today.
func (d *Database) ListUpcomingTasks(ctx context.Context, referenceDate time.Time, limit int) ([]DatedTaskViewRow, error) {
	return d.listDateScopedTasks(ctx, `
	AND COALESCE(date(t.due_at), date(t.due_date)) > date(?)
	AND COALESCE(date(t.due_at), date(t.due_date)) <= date(?, '+' || cfg.upcoming_days || ' days')`, referenceDate, limit, 2)
}

// ListOverdueTasks returns tasks that are overdue.
func (d *Database) ListOverdueTasks(ctx context.Context, referenceDate time.Time, limit int) ([]DatedTaskViewRow, error) {
	return d.listDateScopedTasks(ctx, `
	AND COALESCE(date(t.due_at), date(t.due_date)) < date(?)`, referenceDate, limit, 1)
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
)
SELECT
	t.id,
	t.title,
	t.status,
	COALESCE(t.project_name, ''),
	COALESCE(date(t.due_at), date(t.due_date)) AS due_iso_date
FROM tasks t
CROSS JOIN cfg
WHERE
	COALESCE(date(t.due_at), date(t.due_date)) IS NOT NULL
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
