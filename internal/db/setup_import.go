package db

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"

	"github.com/google/uuid"
)

// SetupProject represents a selected project during setup import.
type SetupProject struct {
	ID           string
	CalendarHref string
	DisplayName  string
}

// ImportedTask stores normalized task payload for setup initial import writes.
type ImportedTask struct {
	UID         string
	Href        string
	ETag        string
	Title       string
	Description string
	Status      string
	CompletedAt *string
	DueDate     *string
	DueAt       *string
	Priority    *int
	RRule       string
	ParentUID   string
	RawVTODO    string
	BaseVTODO   string
	LabelNames  []string
	ProjectName string
}

// LoadSetupImportProjects returns selected projects configured during setup.
func (d *Database) LoadSetupImportProjects(ctx context.Context) ([]SetupProject, error) {
	rows, err := d.Conn.QueryContext(ctx, `
SELECT id, calendar_href, display_name
FROM projects
ORDER BY display_name COLLATE NOCASE, id;
`)
	if err != nil {
		return nil, fmt.Errorf("load setup import projects: query projects: %w", err)
	}
	defer rows.Close()

	projects := make([]SetupProject, 0)
	for rows.Next() {
		var project SetupProject
		if err := rows.Scan(&project.ID, &project.CalendarHref, &project.DisplayName); err != nil {
			return nil, fmt.Errorf("load setup import projects: scan project: %w", err)
		}
		projects = append(projects, project)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("load setup import projects: iterate projects: %w", err)
	}

	return projects, nil
}

// ReplaceSetupProjectTasks writes imported tasks as synced records for one project.
func (d *Database) ReplaceSetupProjectTasks(ctx context.Context, projectID string, tasks []ImportedTask) error {
	if strings.TrimSpace(projectID) == "" {
		return fmt.Errorf("replace setup project tasks: project id is required")
	}

	d.WriteMu.Lock()
	defer d.WriteMu.Unlock()

	tx, err := d.Conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("replace setup project tasks: begin transaction: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `DELETE FROM tasks WHERE project_id = ?;`, projectID); err != nil {
		return fmt.Errorf("replace setup project tasks: clear tasks: %w", err)
	}

	idByUID := make(map[string]string, len(tasks))
	for _, task := range tasks {
		taskID := uuid.NewString()
		idByUID[task.UID] = taskID
		if err := insertImportedTask(ctx, tx, taskID, projectID, "", task); err != nil {
			return err
		}
	}

	parentUIDByUID := make(map[string]string, len(tasks))
	for _, task := range tasks {
		parentUIDByUID[task.UID] = strings.TrimSpace(task.ParentUID)
	}

	for _, task := range tasks {
		parentUID := strings.TrimSpace(task.ParentUID)
		if parentUID == "" {
			continue
		}
		if strings.TrimSpace(parentUIDByUID[parentUID]) != "" {
			continue
		}
		childID, okChild := idByUID[task.UID]
		parentID, okParent := idByUID[parentUID]
		if !okChild || !okParent {
			continue
		}
		if _, err := tx.ExecContext(ctx, `UPDATE tasks SET parent_id = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?;`, parentID, childID); err != nil {
			return fmt.Errorf("replace setup project tasks: update parent_id: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("replace setup project tasks: commit transaction: %w", err)
	}
	return nil
}

func insertImportedTask(ctx context.Context, tx *sql.Tx, taskID string, projectID string, parentID string, task ImportedTask) error {
	labelNames := append([]string(nil), task.LabelNames...)
	sort.Slice(labelNames, func(i, j int) bool {
		return strings.ToLower(labelNames[i]) < strings.ToLower(labelNames[j])
	})
	denormalizedLabels := strings.Join(labelNames, " ")

	if _, err := tx.ExecContext(ctx, `
INSERT INTO tasks (
    id, project_id, uid, href, etag, title, description, status, completed_at,
    due_date, due_at, priority, rrule, parent_id, raw_vtodo, base_vtodo,
    label_names, project_name, sync_status, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'synced', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
`, taskID, projectID, task.UID, task.Href, nullableString(task.ETag), task.Title, nullableString(task.Description), task.Status,
		nullableStringPointer(task.CompletedAt), nullableStringPointer(task.DueDate), nullableStringPointer(task.DueAt), task.Priority,
		nullableString(task.RRule), nullableString(parentID), task.RawVTODO, task.BaseVTODO, nullableString(denormalizedLabels), nullableString(task.ProjectName)); err != nil {
		return fmt.Errorf("replace setup project tasks: insert task: %w", err)
	}
	return nil
}

func nullableString(value string) any {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return trimmed
}

func nullableStringPointer(value *string) any {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return trimmed
}
