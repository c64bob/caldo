package db

import (
	"context"
	"fmt"
	"time"
)

// NavigationSnapshot contains factual counts and entries for the app shell.
type NavigationSnapshot struct {
	TodayCount     int
	UpcomingCount  int
	FavoriteCount  int
	OverdueCount   int
	NoDateCount    int
	CompletedCount int
	ConflictCount  int
	Projects       []NavigationListItem
	Labels         []NavigationListItem
	SavedFilters   []NavigationListItem
}

// NavigationListItem contains one counted navigation entry.
type NavigationListItem struct {
	ID            string
	Name          string
	OpenTaskCount int
	TaskCount     int
	ServerVersion int
}

// LoadNavigationSnapshot returns factual navigation counts for the app shell.
func (d *Database) LoadNavigationSnapshot(ctx context.Context, referenceDate time.Time) (NavigationSnapshot, error) {
	reference := referenceDate.UTC().Format("2006-01-02")
	var snapshot NavigationSnapshot

	if err := d.Conn.QueryRowContext(ctx, `
WITH cfg AS (
	SELECT show_completed, upcoming_days
	FROM settings
	WHERE id = 'default'
),
scoped_tasks AS (
	SELECT
		t.status,
		LOWER(COALESCE(t.label_names, '')) AS label_names,
		COALESCE(
			date(t.due_at),
			date(substr(t.due_at, 1, 19)),
			date(substr(t.due_at, 1, 10)),
			date(t.due_date)
		) AS due_iso_date
	FROM tasks t
)
SELECT
	COALESCE(SUM(CASE WHEN t.due_iso_date <= date(?) AND (cfg.show_completed = 1 OR t.status != 'completed') THEN 1 ELSE 0 END), 0),
	COALESCE(SUM(CASE WHEN t.due_iso_date > date(?) AND t.due_iso_date <= date(?, '+' || cfg.upcoming_days || ' days') AND (cfg.show_completed = 1 OR t.status != 'completed') THEN 1 ELSE 0 END), 0),
	COALESCE(SUM(CASE WHEN t.label_names LIKE '%starred%' AND (cfg.show_completed = 1 OR t.status != 'completed') THEN 1 ELSE 0 END), 0),
	COALESCE(SUM(CASE WHEN t.due_iso_date < date(?) AND (cfg.show_completed = 1 OR t.status != 'completed') THEN 1 ELSE 0 END), 0),
	COALESCE(SUM(CASE WHEN t.due_iso_date IS NULL AND (cfg.show_completed = 1 OR t.status != 'completed') THEN 1 ELSE 0 END), 0),
	COALESCE(SUM(CASE WHEN cfg.show_completed = 1 AND t.status = 'completed' THEN 1 ELSE 0 END), 0)
FROM scoped_tasks t
CROSS JOIN cfg;
`, reference, reference, reference, reference).Scan(
		&snapshot.TodayCount,
		&snapshot.UpcomingCount,
		&snapshot.FavoriteCount,
		&snapshot.OverdueCount,
		&snapshot.NoDateCount,
		&snapshot.CompletedCount,
	); err != nil {
		return NavigationSnapshot{}, fmt.Errorf("load navigation snapshot: count system lists: %w", err)
	}

	conflictCount, err := d.countUnresolvedConflicts(ctx)
	if err != nil {
		return NavigationSnapshot{}, err
	}
	snapshot.ConflictCount = conflictCount

	projects, err := d.listNavigationProjects(ctx)
	if err != nil {
		return NavigationSnapshot{}, err
	}
	snapshot.Projects = projects

	labels, err := d.listNavigationLabels(ctx)
	if err != nil {
		return NavigationSnapshot{}, err
	}
	snapshot.Labels = labels

	filters, err := d.listNavigationSavedFilters(ctx)
	if err != nil {
		return NavigationSnapshot{}, err
	}
	snapshot.SavedFilters = filters

	return snapshot, nil
}

func (d *Database) countUnresolvedConflicts(ctx context.Context) (int, error) {
	var count int
	if err := d.Conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM conflicts WHERE resolved_at IS NULL;`).Scan(&count); err != nil {
		return 0, fmt.Errorf("load navigation snapshot: count conflicts: %w", err)
	}
	return count, nil
}

func (d *Database) listNavigationProjects(ctx context.Context) ([]NavigationListItem, error) {
	rows, err := d.Conn.QueryContext(ctx, `
SELECT
	p.id,
	p.display_name,
	COUNT(CASE WHEN LOWER(t.status) != 'completed' THEN 1 END),
	COUNT(t.id),
	p.server_version
FROM projects p
LEFT JOIN tasks t ON t.project_id = p.id
GROUP BY p.id, p.display_name, p.is_default, p.server_version
ORDER BY p.is_default DESC, p.display_name COLLATE NOCASE ASC, p.id ASC;
`)
	if err != nil {
		return nil, fmt.Errorf("load navigation snapshot: list projects: %w", err)
	}
	defer rows.Close()

	items := make([]NavigationListItem, 0)
	for rows.Next() {
		var item NavigationListItem
		if err := rows.Scan(&item.ID, &item.Name, &item.OpenTaskCount, &item.TaskCount, &item.ServerVersion); err != nil {
			return nil, fmt.Errorf("load navigation snapshot: scan project: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("load navigation snapshot: iterate projects: %w", err)
	}
	return items, nil
}

func (d *Database) listNavigationLabels(ctx context.Context) ([]NavigationListItem, error) {
	rows, err := d.Conn.QueryContext(ctx, `
SELECT l.id, l.name, COUNT(t.id)
FROM labels l
LEFT JOIN task_labels tl ON tl.label_id = l.id
LEFT JOIN tasks t ON t.id = tl.task_id AND t.status != 'completed'
GROUP BY l.id, l.name
ORDER BY l.name COLLATE NOCASE ASC;
`)
	if err != nil {
		return nil, fmt.Errorf("load navigation snapshot: list labels: %w", err)
	}
	defer rows.Close()

	items := make([]NavigationListItem, 0)
	for rows.Next() {
		var item NavigationListItem
		if err := rows.Scan(&item.ID, &item.Name, &item.OpenTaskCount); err != nil {
			return nil, fmt.Errorf("load navigation snapshot: scan label: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("load navigation snapshot: iterate labels: %w", err)
	}
	return items, nil
}

func (d *Database) listNavigationSavedFilters(ctx context.Context) ([]NavigationListItem, error) {
	rows, err := d.Conn.QueryContext(ctx, `SELECT id, name FROM saved_filters WHERE is_favorite = 1 ORDER BY name COLLATE NOCASE ASC;`)
	if err != nil {
		return nil, fmt.Errorf("load navigation snapshot: list saved filters: %w", err)
	}
	defer rows.Close()

	items := make([]NavigationListItem, 0)
	for rows.Next() {
		var item NavigationListItem
		if err := rows.Scan(&item.ID, &item.Name); err != nil {
			return nil, fmt.Errorf("load navigation snapshot: scan saved filter: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("load navigation snapshot: iterate saved filters: %w", err)
	}
	return items, nil
}
