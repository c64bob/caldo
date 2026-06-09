package db

import (
	"context"
	"fmt"
	"strings"
	"unicode"
)

// SearchResult contains one task row returned by global search.
type SearchResult struct {
	ID            string
	ProjectID     string
	Title         string
	Description   string
	Status        string
	ProjectName   string
	LabelNames    string
	DueISODate    string
	Priority      int
	HasPriority   bool
	SyncStatus    string
	ServerVersion int
	ParentID      string
	ParentTitle   string
	IsSubtask     bool
	SubtaskCount  int
	RawVTODO      string
}

// SearchActiveTasks returns active tasks matching text tokens plus optional #project and @label filters.
func (d *Database) SearchActiveTasks(ctx context.Context, rawQuery string, limit int) ([]SearchResult, error) {
	if limit <= 0 {
		limit = 25
	}

	matchExpr := buildSearchMatchExpr(rawQuery)
	if matchExpr == "" {
		return []SearchResult{}, nil
	}

	rows, err := d.Conn.QueryContext(ctx, `
SELECT
	t.id,
	t.project_id,
	t.title,
	COALESCE(t.description, ''),
	t.status,
	COALESCE(t.project_name, ''),
	COALESCE(t.label_names, ''),
	COALESCE(
		date(t.due_at),
		date(substr(t.due_at, 1, 19)),
		date(substr(t.due_at, 1, 10)),
		date(t.due_date),
		''
	),
	COALESCE(t.priority, 0),
	t.priority IS NOT NULL,
	t.sync_status,
	t.server_version,
	COALESCE(t.parent_id, ''),
	COALESCE(parent.title, ''),
	t.parent_id IS NOT NULL,
	(SELECT COUNT(1) FROM tasks child WHERE child.parent_id = t.id),
	COALESCE(t.raw_vtodo, '')
FROM tasks_fts f
JOIN tasks t ON t.rowid = f.rowid
LEFT JOIN tasks parent ON parent.id = t.parent_id
WHERE f.tasks_fts MATCH ?
  AND t.status != 'completed'
ORDER BY bm25(tasks_fts), COALESCE(t.parent_id, t.id), CASE WHEN t.parent_id IS NULL THEN 0 ELSE 1 END, t.updated_at DESC
LIMIT ?;
`, matchExpr, limit)
	if err != nil {
		return nil, fmt.Errorf("search active tasks: %w", err)
	}
	defer rows.Close()

	results := make([]SearchResult, 0, limit)
	for rows.Next() {
		var item SearchResult
		if err := rows.Scan(&item.ID, &item.ProjectID, &item.Title, &item.Description, &item.Status, &item.ProjectName, &item.LabelNames, &item.DueISODate, &item.Priority, &item.HasPriority, &item.SyncStatus, &item.ServerVersion, &item.ParentID, &item.ParentTitle, &item.IsSubtask, &item.SubtaskCount, &item.RawVTODO); err != nil {
			return nil, fmt.Errorf("search active tasks: scan row: %w", err)
		}
		results = append(results, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("search active tasks: iterate rows: %w", err)
	}

	return results, nil
}

func buildSearchMatchExpr(rawQuery string) string {
	parts := make([]string, 0)
	for _, token := range strings.Fields(rawQuery) {
		if token == "" {
			continue
		}

		field := ""
		if strings.HasPrefix(token, "#") {
			field = "project_name"
			token = strings.TrimPrefix(token, "#")
		} else if strings.HasPrefix(token, "@") {
			field = "label_names"
			token = strings.TrimPrefix(token, "@")
		}

		normalized := normalizeSearchToken(token)
		if normalized == "" {
			continue
		}

		if field == "" {
			parts = append(parts, fmt.Sprintf("(title:%s* OR description:%s* OR label_names:%s* OR project_name:%s*)", normalized, normalized, normalized, normalized))
			continue
		}

		parts = append(parts, fmt.Sprintf("%s:%s*", field, normalized))
	}

	return strings.Join(parts, " AND ")
}

func normalizeSearchToken(token string) string {
	builder := strings.Builder{}
	for _, r := range strings.ToLower(token) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			builder.WriteRune(r)
		}
	}
	return builder.String()
}
