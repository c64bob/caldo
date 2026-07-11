package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"caldo/internal/query"
	"github.com/google/uuid"
)

// SavedFilter stores a persisted user filter.
type SavedFilter struct {
	ID            string
	Name          string
	Query         string
	IsFavorite    bool
	ServerVersion int
}

var (
	// ErrSavedFilterNotFound indicates that a saved filter does not exist.
	ErrSavedFilterNotFound = errors.New("saved filter not found")
	// ErrSavedFilterVersionConflict indicates optimistic locking conflict.
	ErrSavedFilterVersionConflict = errors.New("saved filter version conflict")
)

// ListSavedFilters returns all saved filters ordered by name.
func (d *Database) ListSavedFilters(ctx context.Context) ([]SavedFilter, error) {
	rows, err := d.Conn.QueryContext(ctx, `SELECT id, name, query, is_favorite, server_version FROM saved_filters ORDER BY name COLLATE NOCASE ASC;`)
	if err != nil {
		return nil, fmt.Errorf("list saved filters: %w", err)
	}
	defer rows.Close()

	result := make([]SavedFilter, 0)
	for rows.Next() {
		var item SavedFilter
		var favorite int
		if err := rows.Scan(&item.ID, &item.Name, &item.Query, &favorite, &item.ServerVersion); err != nil {
			return nil, fmt.Errorf("list saved filters: scan row: %w", err)
		}
		item.IsFavorite = favorite == 1
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list saved filters: iterate rows: %w", err)
	}
	return result, nil
}

// LoadSavedFilter returns one saved filter by id.
func (d *Database) LoadSavedFilter(ctx context.Context, id string) (SavedFilter, error) {
	return d.loadSavedFilterByID(ctx, id)
}

// CreateSavedFilter creates a saved filter with name and query.
func (d *Database) CreateSavedFilter(ctx context.Context, name, filterQuery string, favorite bool) (SavedFilter, error) {
	d.WriteMu.Lock()
	defer d.WriteMu.Unlock()

	id := uuid.NewString()
	fav := 0
	if favorite {
		fav = 1
	}

	if _, err := d.Conn.ExecContext(ctx, `INSERT INTO saved_filters (id, name, query, is_favorite) VALUES (?, ?, ?, ?);`, id, strings.TrimSpace(name), strings.TrimSpace(filterQuery), fav); err != nil {
		return SavedFilter{}, fmt.Errorf("create saved filter: %w", err)
	}

	return d.loadSavedFilterByID(ctx, id)
}

// UpdateSavedFilter updates a saved filter using optimistic locking.
func (d *Database) UpdateSavedFilter(ctx context.Context, id, name, filterQuery string, favorite bool, expectedVersion int) (SavedFilter, error) {
	d.WriteMu.Lock()
	defer d.WriteMu.Unlock()

	fav := 0
	if favorite {
		fav = 1
	}

	res, err := d.Conn.ExecContext(ctx, `
UPDATE saved_filters
SET name=?, query=?, is_favorite=?, server_version=server_version+1, updated_at=CURRENT_TIMESTAMP
WHERE id=? AND server_version=?;
`, strings.TrimSpace(name), strings.TrimSpace(filterQuery), fav, id, expectedVersion)
	if err != nil {
		return SavedFilter{}, fmt.Errorf("update saved filter: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return SavedFilter{}, fmt.Errorf("update saved filter: rows affected: %w", err)
	}
	if affected == 0 {
		return SavedFilter{}, ErrSavedFilterVersionConflict
	}

	return d.loadSavedFilterByID(ctx, id)
}

// DeleteSavedFilter deletes a saved filter using optimistic locking.
func (d *Database) DeleteSavedFilter(ctx context.Context, id string, expectedVersion int) error {
	d.WriteMu.Lock()
	defer d.WriteMu.Unlock()

	res, err := d.Conn.ExecContext(ctx, `DELETE FROM saved_filters WHERE id=? AND server_version=?;`, id, expectedVersion)
	if err != nil {
		return fmt.Errorf("delete saved filter: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete saved filter: rows affected: %w", err)
	}
	if affected == 0 {
		return ErrSavedFilterVersionConflict
	}
	return nil
}

// ListSavedFilterTasks returns tasks matching a saved filter query.
func (d *Database) ListSavedFilterTasks(ctx context.Context, filterID string, referenceDate time.Time, limit int) (SavedFilter, []DatedTaskViewRow, bool, error) {
	if limit <= 0 {
		limit = 200
	}

	filter, err := d.LoadSavedFilter(ctx, filterID)
	if err != nil {
		return SavedFilter{}, nil, false, err
	}

	var upcomingDays int
	if err := d.Conn.QueryRowContext(ctx, `SELECT upcoming_days FROM settings WHERE id = 'default';`).Scan(&upcomingDays); err != nil {
		return SavedFilter{}, nil, false, fmt.Errorf("list saved filter tasks: load settings: %w", err)
	}

	whereSQL, args, ok, err := EvaluateSavedFilterAt(filter.Query, query.CompileOptions{Now: referenceDate, UpcomingDays: upcomingDays})
	if err != nil {
		return SavedFilter{}, nil, false, fmt.Errorf("list saved filter tasks: evaluate query: %w", err)
	}
	if !ok {
		return filter, []DatedTaskViewRow{}, false, nil
	}

	queryArgs := make([]any, 0, len(args)+1)
	queryArgs = append(queryArgs, args...)
	queryArgs = append(queryArgs, limit)

	// #nosec G202 -- whereSQL is emitted by the filter compiler and user values are passed as query args.
	rows, err := d.Conn.QueryContext(ctx, `
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
	COALESCE(t.raw_vtodo, ''),
	COALESCE(t.created_at, '')
FROM tasks t
LEFT JOIN tasks parent ON parent.id = t.parent_id
WHERE t.id IN (SELECT id FROM tasks WHERE `+whereSQL+`)
ORDER BY COALESCE(t.parent_id, t.id), CASE WHEN t.parent_id IS NULL THEN 0 ELSE 1 END, t.updated_at DESC
LIMIT ?;
`, queryArgs...)
	if err != nil {
		return SavedFilter{}, nil, false, fmt.Errorf("list saved filter tasks: query rows: %w", err)
	}
	defer rows.Close()

	results := make([]DatedTaskViewRow, 0, limit)
	for rows.Next() {
		var row DatedTaskViewRow
		if err := rows.Scan(&row.ID, &row.ProjectID, &row.Title, &row.Description, &row.Status, &row.ProjectName, &row.DueISODate, &row.Priority, &row.HasPriority, &row.LabelNames, &row.SyncStatus, &row.ServerVersion, &row.ParentID, &row.ParentTitle, &row.IsSubtask, &row.SubtaskCount, &row.OpenSubtaskCount, &row.UnresolvedConflictID, &row.RawVTODO, &row.CreatedAt); err != nil {
			return SavedFilter{}, nil, false, fmt.Errorf("list saved filter tasks: scan row: %w", err)
		}
		results = append(results, row)
	}
	if err := rows.Err(); err != nil {
		return SavedFilter{}, nil, false, fmt.Errorf("list saved filter tasks: iterate rows: %w", err)
	}

	return filter, results, true, nil
}

// EvaluateSavedFilter compiles the filter query using the provided upcoming window in days.
// Invalid syntax returns an empty result set without error.
func EvaluateSavedFilter(filterQuery string, upcomingDays int) (string, []any, bool, error) {
	return EvaluateSavedFilterAt(filterQuery, query.CompileOptions{UpcomingDays: upcomingDays})
}

// EvaluateSavedFilterAt compiles the filter query with the supplied compiler options.
// Invalid syntax returns an empty result set without error.
func EvaluateSavedFilterAt(filterQuery string, opts query.CompileOptions) (string, []any, bool, error) {
	tokens := query.LexFilter(filterQuery)
	ast, err := query.ParseFilter(tokens)
	if err != nil {
		return "", nil, false, nil
	}
	where, args, err := query.CompileFilter(ast, opts)
	if err != nil {
		return "", nil, false, nil
	}
	return where, args, true, nil
}

func (d *Database) loadSavedFilterByID(ctx context.Context, id string) (SavedFilter, error) {
	var item SavedFilter
	var favorite int
	if err := d.Conn.QueryRowContext(ctx, `SELECT id, name, query, is_favorite, server_version FROM saved_filters WHERE id=?;`, id).Scan(&item.ID, &item.Name, &item.Query, &favorite, &item.ServerVersion); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return SavedFilter{}, ErrSavedFilterNotFound
		}
		return SavedFilter{}, fmt.Errorf("load saved filter: %w", err)
	}
	item.IsFavorite = favorite == 1
	return item, nil
}
