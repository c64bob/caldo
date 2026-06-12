package db

import (
	"context"
	"fmt"
	"strings"

	"caldo/internal/model"
)

// LabelOption contains one selectable label for task editing and quick add.
type LabelOption struct {
	Name string
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
