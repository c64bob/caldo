package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"caldo/internal/model"
)

// ErrLabelNotFound is returned when a requested user label does not exist.
var ErrLabelNotFound = errors.New("label not found")

// ErrLabelStillInUse indicates a direct label row mutation would desynchronize task labels.
var ErrLabelStillInUse = errors.New("label still in use")

// LabelDetail contains one user label with task counters.
type LabelDetail struct {
	ID            string
	Name          string
	OpenTaskCount int
	TaskCount     int
}

// LabelOption contains one selectable label for task editing and quick add.
type LabelOption struct {
	Name string
}

// LabelMutationTask contains the task fields needed for label write-through operations.
type LabelMutationTask struct {
	ID            string
	ProjectID     string
	ProjectName   string
	Href          string
	ETag          string
	ServerVersion int
	RawVTODO      string
}

// LoadLabelDetail returns one user label and its task counters.
func (d *Database) LoadLabelDetail(ctx context.Context, labelID string) (LabelDetail, error) {
	var detail LabelDetail
	err := d.Conn.QueryRowContext(ctx, `
SELECT
	l.id,
	l.name,
	COUNT(CASE WHEN t.id IS NOT NULL AND LOWER(t.status) != 'completed' THEN 1 END),
	COUNT(t.id)
FROM labels l
LEFT JOIN task_labels tl ON tl.label_id = l.id
LEFT JOIN tasks t ON t.id = tl.task_id
WHERE l.id = ? AND LOWER(l.name) != LOWER(?)
GROUP BY l.id, l.name;
`, labelID, model.ReservedFavoriteCategory).Scan(&detail.ID, &detail.Name, &detail.OpenTaskCount, &detail.TaskCount)
	if errors.Is(err, sql.ErrNoRows) {
		return LabelDetail{}, ErrLabelNotFound
	}
	if err != nil {
		return LabelDetail{}, fmt.Errorf("load label detail: %w", err)
	}

	return detail, nil
}

// ListLabelMutationTasks returns all tasks currently attached to a user label.
func (d *Database) ListLabelMutationTasks(ctx context.Context, labelID string) ([]LabelMutationTask, error) {
	trimmedLabelID := strings.TrimSpace(labelID)
	if trimmedLabelID == "" {
		return nil, fmt.Errorf("list label mutation tasks: label id is required")
	}

	rows, err := d.Conn.QueryContext(ctx, `
SELECT
	t.id,
	t.project_id,
	COALESCE(t.project_name, p.display_name, ''),
	t.href,
	COALESCE(t.etag, ''),
	t.server_version,
	COALESCE(t.raw_vtodo, '')
FROM tasks t
JOIN task_labels tl ON tl.task_id = t.id
JOIN labels l ON l.id = tl.label_id
JOIN projects p ON p.id = t.project_id
WHERE l.id = ? AND LOWER(l.name) != LOWER(?)
ORDER BY t.updated_at ASC, t.id ASC;
`, trimmedLabelID, model.ReservedFavoriteCategory)
	if err != nil {
		return nil, fmt.Errorf("list label mutation tasks: %w", err)
	}
	defer rows.Close()

	tasks := make([]LabelMutationTask, 0)
	for rows.Next() {
		var task LabelMutationTask
		if err := rows.Scan(&task.ID, &task.ProjectID, &task.ProjectName, &task.Href, &task.ETag, &task.ServerVersion, &task.RawVTODO); err != nil {
			return nil, fmt.Errorf("list label mutation tasks: scan task: %w", err)
		}
		tasks = append(tasks, task)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list label mutation tasks: iterate tasks: %w", err)
	}

	return tasks, nil
}

// RenameUnusedLabel renames a label row that has no task assignments.
func (d *Database) RenameUnusedLabel(ctx context.Context, labelID string, name string) error {
	trimmedLabelID := strings.TrimSpace(labelID)
	trimmedName := strings.TrimSpace(name)
	if trimmedLabelID == "" {
		return fmt.Errorf("rename unused label: label id is required")
	}
	if trimmedName == "" {
		return fmt.Errorf("rename unused label: label name is required")
	}

	d.WriteMu.Lock()
	defer d.WriteMu.Unlock()

	tx, err := d.Conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("rename unused label: begin transaction: %w", err)
	}
	defer tx.Rollback()

	if err := ensureLabelUnused(ctx, tx, trimmedLabelID); err != nil {
		return err
	}

	var existingID string
	err = tx.QueryRowContext(ctx, `
SELECT id
FROM labels
WHERE name = ? COLLATE NOCASE
LIMIT 1;
`, trimmedName).Scan(&existingID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("rename unused label: query existing label name: %w", err)
	}
	if err == nil && existingID != trimmedLabelID {
		if _, err := tx.ExecContext(ctx, `DELETE FROM labels WHERE id = ?;`, trimmedLabelID); err != nil {
			return fmt.Errorf("rename unused label: remove duplicate unused label: %w", err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("rename unused label: commit duplicate removal: %w", err)
		}
		return nil
	}

	result, err := tx.ExecContext(ctx, `
UPDATE labels
SET name = ?
WHERE id = ?;
`, trimmedName, trimmedLabelID)
	if err != nil {
		return fmt.Errorf("rename unused label: update label: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rename unused label: read affected rows: %w", err)
	}
	if affected != 1 {
		return ErrLabelNotFound
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("rename unused label: commit transaction: %w", err)
	}
	return nil
}

// DeleteUnusedLabel deletes a label row that has no task assignments.
func (d *Database) DeleteUnusedLabel(ctx context.Context, labelID string) error {
	trimmedLabelID := strings.TrimSpace(labelID)
	if trimmedLabelID == "" {
		return fmt.Errorf("delete unused label: label id is required")
	}

	d.WriteMu.Lock()
	defer d.WriteMu.Unlock()

	tx, err := d.Conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("delete unused label: begin transaction: %w", err)
	}
	defer tx.Rollback()

	if err := ensureLabelUnused(ctx, tx, trimmedLabelID); err != nil {
		return err
	}

	result, err := tx.ExecContext(ctx, `DELETE FROM labels WHERE id = ?;`, trimmedLabelID)
	if err != nil {
		return fmt.Errorf("delete unused label: delete label: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete unused label: read affected rows: %w", err)
	}
	if affected != 1 {
		return ErrLabelNotFound
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("delete unused label: commit transaction: %w", err)
	}
	return nil
}

// RenameLabelRow renames a label row after callers have updated all affected task labels.
func (d *Database) RenameLabelRow(ctx context.Context, labelID string, name string) error {
	trimmedLabelID := strings.TrimSpace(labelID)
	trimmedName := strings.TrimSpace(name)
	if trimmedLabelID == "" {
		return fmt.Errorf("rename label row: label id is required")
	}
	if trimmedName == "" {
		return fmt.Errorf("rename label row: label name is required")
	}

	d.WriteMu.Lock()
	defer d.WriteMu.Unlock()

	result, err := d.Conn.ExecContext(ctx, `
UPDATE labels
SET name = ?
WHERE id = ? AND LOWER(name) != LOWER(?);
`, trimmedName, trimmedLabelID, model.ReservedFavoriteCategory)
	if err != nil {
		return fmt.Errorf("rename label row: update label: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rename label row: read affected rows: %w", err)
	}
	if affected != 1 {
		return ErrLabelNotFound
	}
	return nil
}

// ListLabelOptions returns existing user labels ordered by display name.
func (d *Database) ListLabelOptions(ctx context.Context) ([]LabelOption, error) {
	rows, err := d.Conn.QueryContext(ctx, `
SELECT name
FROM labels
WHERE LOWER(name) != LOWER(?)
ORDER BY name COLLATE NOCASE ASC;
`, model.ReservedFavoriteCategory)
	if err != nil {
		return nil, fmt.Errorf("list label options: %w", err)
	}
	defer rows.Close()

	labels := make([]LabelOption, 0)
	seen := make(map[string]struct{})
	for rows.Next() {
		var label LabelOption
		if err := rows.Scan(&label.Name); err != nil {
			return nil, fmt.Errorf("list label options: scan label: %w", err)
		}
		name := strings.TrimSpace(label.Name)
		if name == "" {
			continue
		}
		key := strings.ToLower(name)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		labels = append(labels, LabelOption{Name: name})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list label options: iterate labels: %w", err)
	}

	return labels, nil
}

func ensureLabelUnused(ctx context.Context, tx *sql.Tx, labelID string) error {
	var assignmentCount int
	if err := tx.QueryRowContext(ctx, `
SELECT COUNT(tl.task_id)
FROM labels l
LEFT JOIN task_labels tl ON tl.label_id = l.id
WHERE l.id = ? AND LOWER(l.name) != LOWER(?)
GROUP BY l.id;
`, labelID, model.ReservedFavoriteCategory).Scan(&assignmentCount); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrLabelNotFound
		}
		return fmt.Errorf("check label usage: %w", err)
	}
	if assignmentCount > 0 {
		return ErrLabelStillInUse
	}
	return nil
}
