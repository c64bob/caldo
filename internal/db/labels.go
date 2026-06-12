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
