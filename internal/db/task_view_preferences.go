package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"caldo/internal/model"
)

// LoadTaskViewPreference returns a saved preference or the scope default.
func (d *Database) LoadTaskViewPreference(ctx context.Context, viewKind, viewID string) (model.TaskViewPreference, error) {
	preference := model.DefaultTaskViewPreference(viewKind, viewID)
	err := d.Conn.QueryRowContext(ctx, `
SELECT sort_by, sort_order, group_by
FROM task_view_preferences
WHERE view_kind = ? AND view_id = ?;
`, preference.ViewKind, preference.ViewID).Scan(&preference.SortBy, &preference.SortOrder, &preference.GroupBy)
	if errors.Is(err, sql.ErrNoRows) {
		if validateErr := model.ValidateTaskViewPreference(preference); validateErr != nil {
			return model.TaskViewPreference{}, validateErr
		}
		return preference, nil
	}
	if err != nil {
		return model.TaskViewPreference{}, fmt.Errorf("load task view preference: %w", err)
	}
	if err := model.ValidateTaskViewPreference(preference); err != nil {
		return model.TaskViewPreference{}, fmt.Errorf("load task view preference: %w", err)
	}
	return preference, nil
}

// SaveTaskViewPreference persists display-only ordering for one task-list scope.
func (d *Database) SaveTaskViewPreference(ctx context.Context, preference model.TaskViewPreference) error {
	preference.ViewKind = strings.TrimSpace(preference.ViewKind)
	preference.ViewID = strings.TrimSpace(preference.ViewID)
	if err := model.ValidateTaskViewPreference(preference); err != nil {
		return err
	}

	d.WriteMu.Lock()
	defer d.WriteMu.Unlock()

	_, err := d.Conn.ExecContext(ctx, `
INSERT INTO task_view_preferences (view_kind, view_id, sort_by, sort_order, group_by)
VALUES (?, ?, ?, ?, ?)
ON CONFLICT(view_kind, view_id) DO UPDATE SET
    sort_by = excluded.sort_by,
    sort_order = excluded.sort_order,
    group_by = excluded.group_by,
    updated_at = CURRENT_TIMESTAMP;
`, preference.ViewKind, preference.ViewID, preference.SortBy, preference.SortOrder, preference.GroupBy)
	if err != nil {
		return fmt.Errorf("save task view preference: %w", err)
	}
	return nil
}

// DeleteTaskViewPreference restores one task-list scope to its default display.
func (d *Database) DeleteTaskViewPreference(ctx context.Context, viewKind, viewID string) error {
	preference := model.DefaultTaskViewPreference(viewKind, viewID)
	if err := model.ValidateTaskViewPreference(preference); err != nil {
		return err
	}

	d.WriteMu.Lock()
	defer d.WriteMu.Unlock()

	if _, err := d.Conn.ExecContext(ctx, `DELETE FROM task_view_preferences WHERE view_kind = ? AND view_id = ?;`, preference.ViewKind, preference.ViewID); err != nil {
		return fmt.Errorf("delete task view preference: %w", err)
	}
	return nil
}
