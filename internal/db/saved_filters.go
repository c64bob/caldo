package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

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

// ErrVersionConflict indicates optimistic locking conflict.
var ErrSavedFilterVersionConflict = errors.New("saved filter version conflict")

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

// EvaluateSavedFilter compiles the filter query using the provided upcoming window in days.
// Invalid syntax returns an empty result set without error.
func EvaluateSavedFilter(filterQuery string, upcomingDays int) (string, []any, bool, error) {
	tokens := query.LexFilter(filterQuery)
	ast, err := query.ParseFilter(tokens)
	if err != nil {
		return "", nil, false, nil
	}
	where, args, err := query.CompileFilter(ast, query.CompileOptions{UpcomingDays: upcomingDays})
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
			return SavedFilter{}, fmt.Errorf("load saved filter: not found")
		}
		return SavedFilter{}, fmt.Errorf("load saved filter: %w", err)
	}
	item.IsFavorite = favorite == 1
	return item, nil
}
