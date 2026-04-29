package db

import (
	"context"
	"fmt"
	"strings"
)

// TaskVersionRow represents the current server version for one task id.
type TaskVersionRow struct {
	TaskID        string
	ServerVersion int
}

// ListTaskVersions returns current task versions for provided task IDs.
func (d *Database) ListTaskVersions(ctx context.Context, taskIDs []string) ([]TaskVersionRow, error) {
	if len(taskIDs) == 0 {
		return []TaskVersionRow{}, nil
	}

	placeholders := make([]string, len(taskIDs))
	args := make([]any, len(taskIDs))
	for i, taskID := range taskIDs {
		placeholders[i] = "?"
		args[i] = taskID
	}

	query := `SELECT id, server_version FROM tasks WHERE id IN (` + strings.Join(placeholders, ",") + `);`
	rows, err := d.Conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list task versions: %w", err)
	}
	defer rows.Close()

	results := make([]TaskVersionRow, 0, len(taskIDs))
	for rows.Next() {
		var row TaskVersionRow
		if err := rows.Scan(&row.TaskID, &row.ServerVersion); err != nil {
			return nil, fmt.Errorf("list task versions: scan row: %w", err)
		}
		results = append(results, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list task versions: iterate rows: %w", err)
	}

	return results, nil
}
