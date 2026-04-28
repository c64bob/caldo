package db

import (
	"context"
	"fmt"
	"strings"
	"unicode"
)

// SearchResult contains one task row returned by global search.
type SearchResult struct {
	ID          string
	Title       string
	Description string
	ProjectName string
	LabelNames  string
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
SELECT t.id, t.title, COALESCE(t.description, ''), COALESCE(t.project_name, ''), COALESCE(t.label_names, '')
FROM tasks_fts f
JOIN tasks t ON t.rowid = f.rowid
WHERE f.tasks_fts MATCH ?
  AND t.status != 'completed'
ORDER BY bm25(tasks_fts), t.updated_at DESC
LIMIT ?;
`, matchExpr, limit)
	if err != nil {
		return nil, fmt.Errorf("search active tasks: %w", err)
	}
	defer rows.Close()

	results := make([]SearchResult, 0, limit)
	for rows.Next() {
		var item SearchResult
		if err := rows.Scan(&item.ID, &item.Title, &item.Description, &item.ProjectName, &item.LabelNames); err != nil {
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
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-' {
			builder.WriteRune(r)
		}
	}
	return builder.String()
}
