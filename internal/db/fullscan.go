package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"caldo/internal/model"
	"github.com/google/uuid"
)

// FullScanApplyResult summarizes one full-scan DB mutation chunk.
type FullScanApplyResult struct {
	Inserted  int
	Updated   int
	Deleted   int
	Conflicts int
}

type fullScanLocalTask struct {
	ID            string
	UID           string
	Href          string
	ETag          string
	ServerVersion int
	RawVTODO      string
	BaseVTODO     string
	SyncStatus    string
}

// ApplyFullScanProject applies normalized remote scan results for one project in a single DB chunk.
func (d *Database) ApplyFullScanProject(ctx context.Context, projectID string, tasks []ImportedTask) (FullScanApplyResult, error) {
	if strings.TrimSpace(projectID) == "" {
		return FullScanApplyResult{}, fmt.Errorf("apply fullscan project: project id is required")
	}

	d.WriteMu.Lock()
	defer d.WriteMu.Unlock()

	tx, err := d.Conn.BeginTx(ctx, nil)
	if err != nil {
		return FullScanApplyResult{}, fmt.Errorf("apply fullscan project: begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	projectName, err := loadProjectDisplayName(ctx, tx, projectID)
	if err != nil {
		return FullScanApplyResult{}, err
	}

	localTasks, err := loadFullScanLocalTasks(ctx, tx, projectID)
	if err != nil {
		return FullScanApplyResult{}, fmt.Errorf("apply fullscan project: load local tasks: %w", err)
	}
	localByUID := make(map[string]fullScanLocalTask, len(localTasks))
	for _, task := range localTasks {
		if strings.TrimSpace(task.UID) == "" {
			continue
		}
		if _, exists := localByUID[task.UID]; exists {
			continue
		}
		localByUID[task.UID] = task
	}

	remoteTasks := dedupeImportedTasks(tasks, projectName)
	parentUIDByUID := make(map[string]string, len(remoteTasks))
	for _, task := range remoteTasks {
		parentUIDByUID[task.UID] = strings.TrimSpace(task.ParentUID)
	}

	var result FullScanApplyResult
	seenLocalIDs := make(map[string]struct{}, len(remoteTasks))
	for _, remoteTask := range remoteTasks {
		localTask, exists := localByUID[remoteTask.UID]
		if !exists {
			taskID := uuid.NewString()
			if err := insertImportedTask(ctx, tx, taskID, projectID, "", remoteTask); err != nil {
				return FullScanApplyResult{}, fmt.Errorf("apply fullscan project: insert remote task: %w", err)
			}
			if err := syncTaskLabels(ctx, tx, taskID, importedLabelsNull(remoteTask.LabelNames)); err != nil {
				return FullScanApplyResult{}, fmt.Errorf("apply fullscan project: sync inserted labels: %w", err)
			}
			result.Inserted++
			continue
		}

		seenLocalIDs[localTask.ID] = struct{}{}
		if localTask.SyncStatus == "conflict" {
			continue
		}
		if !remoteTaskChanged(localTask, remoteTask) {
			continue
		}
		if sameVTODO(localTask.RawVTODO, remoteTask.RawVTODO) || localTaskIsClean(localTask) {
			if err := updateTaskFromImported(ctx, tx, localTask, remoteTask); err != nil {
				return FullScanApplyResult{}, err
			}
			result.Updated++
			continue
		}

		merge := model.MergeVTODOFields(localTask.BaseVTODO, localTask.RawVTODO, remoteTask.RawVTODO)
		if merge.Merged && !merge.Conflict {
			mergedTask := importedTaskWithRaw(remoteTask, merge.MergedVTODO)
			if err := updateTaskFromImported(ctx, tx, localTask, mergedTask); err != nil {
				return FullScanApplyResult{}, err
			}
			result.Updated++
			continue
		}

		if err := recordFullScanTaskConflict(ctx, tx, localTask, conflictTypeFieldConflict, localTask.BaseVTODO, localTask.RawVTODO, remoteTask.RawVTODO, remoteTask.ETag); err != nil {
			return FullScanApplyResult{}, err
		}
		result.Conflicts++
	}

	deleteTasks := make([]fullScanLocalTask, 0)
	for _, localTask := range localTasks {
		if _, seen := seenLocalIDs[localTask.ID]; seen {
			continue
		}
		if localTask.SyncStatus == "conflict" {
			continue
		}
		if localTaskIsClean(localTask) {
			deleteTasks = append(deleteTasks, localTask)
			continue
		}
		if err := recordFullScanTaskConflict(ctx, tx, localTask, conflictTypeEditDelete, localTask.BaseVTODO, localTask.RawVTODO, "", ""); err != nil {
			return FullScanApplyResult{}, err
		}
		result.Conflicts++
	}
	if len(deleteTasks) > 0 {
		deleted, err := deleteFullScanTasks(ctx, tx, deleteTasks)
		if err != nil {
			return FullScanApplyResult{}, err
		}
		result.Deleted = deleted
	}

	if err := applyFullScanParentLinks(ctx, tx, projectID, parentUIDByUID); err != nil {
		return FullScanApplyResult{}, err
	}

	if err := tx.Commit(); err != nil {
		return FullScanApplyResult{}, fmt.Errorf("apply fullscan project: commit transaction: %w", err)
	}
	return result, nil
}

func loadProjectDisplayName(ctx context.Context, tx *sql.Tx, projectID string) (string, error) {
	var displayName string
	err := tx.QueryRowContext(ctx, `SELECT display_name FROM projects WHERE id = ?;`, strings.TrimSpace(projectID)).Scan(&displayName)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrProjectNotFound
		}
		return "", fmt.Errorf("apply fullscan project: load project: %w", err)
	}
	return displayName, nil
}

func loadFullScanLocalTasks(ctx context.Context, tx *sql.Tx, projectID string) ([]fullScanLocalTask, error) {
	rows, err := tx.QueryContext(ctx, `
SELECT id, uid, COALESCE(href, ''), COALESCE(etag, ''), server_version, raw_vtodo, COALESCE(base_vtodo, ''), sync_status
FROM tasks
WHERE project_id = ?
ORDER BY created_at ASC, id ASC;
`, strings.TrimSpace(projectID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tasks := make([]fullScanLocalTask, 0)
	for rows.Next() {
		var task fullScanLocalTask
		if err := rows.Scan(&task.ID, &task.UID, &task.Href, &task.ETag, &task.ServerVersion, &task.RawVTODO, &task.BaseVTODO, &task.SyncStatus); err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return tasks, nil
}

func dedupeImportedTasks(tasks []ImportedTask, projectName string) []ImportedTask {
	results := make([]ImportedTask, 0, len(tasks))
	seen := make(map[string]struct{}, len(tasks))
	for _, task := range tasks {
		task.UID = strings.TrimSpace(task.UID)
		if task.UID == "" {
			continue
		}
		if _, exists := seen[task.UID]; exists {
			continue
		}
		seen[task.UID] = struct{}{}
		if strings.TrimSpace(task.ProjectName) == "" {
			task.ProjectName = projectName
		}
		if strings.TrimSpace(task.Title) == "" {
			task.Title = task.UID
		}
		if strings.TrimSpace(task.Status) == "" {
			task.Status = "needs-action"
		}
		if strings.TrimSpace(task.BaseVTODO) == "" {
			task.BaseVTODO = task.RawVTODO
		}
		results = append(results, task)
	}
	return results
}

func importedTaskWithRaw(task ImportedTask, rawVTODO string) ImportedTask {
	parsed := model.ParseVTODOFields(rawVTODO)
	if parsed.UID != "" {
		task.UID = parsed.UID
	}
	title := parsed.Title
	if title == "" {
		title = task.UID
	}
	categories, priority, err := model.NormalizeFavoritePriorityFields(parsed.Categories, parsed.Priority)
	if err != nil {
		categories = parsed.Categories
		priority = parsed.Priority
	}
	labels, favorite := model.CategoriesToLabelsAndFavorite(categories)
	if favorite {
		labels = append(labels, model.ReservedFavoriteCategory)
	}

	task.Title = title
	task.Description = parsed.Description
	task.Status = parsed.Status
	task.CompletedAt = fullScanTimePointer(parsed.CompletedAt)
	task.DueDate = parsed.DueDate
	task.DueAt = fullScanTimePointer(parsed.DueAt)
	task.Priority = priority
	task.RRule = parsed.RRule
	task.ParentUID = parsed.ParentUID
	task.RawVTODO = rawVTODO
	task.BaseVTODO = rawVTODO
	task.LabelNames = labels
	return task
}

func fullScanTimePointer(v *time.Time) *string {
	if v == nil {
		return nil
	}
	formatted := v.UTC().Format(time.RFC3339)
	return &formatted
}

func remoteTaskChanged(local fullScanLocalTask, remote ImportedTask) bool {
	return !sameVTODO(local.BaseVTODO, remote.RawVTODO) ||
		strings.TrimSpace(local.ETag) != strings.TrimSpace(remote.ETag) ||
		strings.TrimSpace(local.Href) != strings.TrimSpace(remote.Href)
}

func sameVTODO(left string, right string) bool {
	return strings.TrimSpace(left) == strings.TrimSpace(right)
}

func localTaskIsClean(task fullScanLocalTask) bool {
	if task.SyncStatus == "pending" || task.SyncStatus == "conflict" {
		return false
	}
	return sameVTODO(task.RawVTODO, task.BaseVTODO)
}

func updateTaskFromImported(ctx context.Context, tx *sql.Tx, local fullScanLocalTask, task ImportedTask) error {
	result, err := tx.ExecContext(ctx, `
UPDATE tasks
SET href = ?,
    etag = ?,
    title = ?,
    description = ?,
    status = ?,
    completed_at = ?,
    due_date = ?,
    due_at = ?,
    priority = ?,
    rrule = ?,
    raw_vtodo = ?,
    base_vtodo = ?,
    label_names = ?,
    project_name = ?,
    sync_status = 'synced',
    server_version = server_version + 1,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ? AND server_version = ?;
`, task.Href, nullableString(task.ETag), task.Title, nullableString(task.Description), task.Status,
		nullableStringPointer(task.CompletedAt), nullableStringPointer(task.DueDate), nullableStringPointer(task.DueAt),
		task.Priority, nullableString(task.RRule), task.RawVTODO, task.RawVTODO, nullableString(denormalizedLabelNames(task.LabelNames)),
		nullableString(task.ProjectName), local.ID, local.ServerVersion)
	if err != nil {
		return fmt.Errorf("apply fullscan project: update task: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("apply fullscan project: update task rows affected: %w", err)
	}
	if affected != 1 {
		return ErrTaskVersionMismatch
	}
	if err := syncTaskLabels(ctx, tx, local.ID, importedLabelsNull(task.LabelNames)); err != nil {
		return fmt.Errorf("apply fullscan project: sync updated labels: %w", err)
	}
	return nil
}

func recordFullScanTaskConflict(ctx context.Context, tx *sql.Tx, task fullScanLocalTask, conflictType string, baseVTODO string, localVTODO string, remoteVTODO string, remoteETag string) error {
	result, err := tx.ExecContext(ctx, `
UPDATE tasks
SET etag = COALESCE(NULLIF(?, ''), etag),
    sync_status = 'conflict',
    updated_at = CURRENT_TIMESTAMP
WHERE id = ? AND server_version = ?;
`, strings.TrimSpace(remoteETag), task.ID, task.ServerVersion)
	if err != nil {
		return fmt.Errorf("apply fullscan project: mark conflict: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("apply fullscan project: mark conflict rows affected: %w", err)
	}
	if affected != 1 {
		return ErrTaskVersionMismatch
	}

	if _, err := tx.ExecContext(ctx, `
INSERT INTO conflicts (id, task_id, project_id, conflict_type, created_at, base_vtodo, local_vtodo, remote_vtodo)
SELECT ?, t.id, t.project_id, ?, CURRENT_TIMESTAMP, NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, '')
FROM tasks t
WHERE t.id = ?;
`, uuid.NewString(), conflictType, baseVTODO, localVTODO, remoteVTODO, task.ID); err != nil {
		return fmt.Errorf("apply fullscan project: insert conflict: %w", err)
	}
	return nil
}

func deleteFullScanTasks(ctx context.Context, tx *sql.Tx, tasks []fullScanLocalTask) (int, error) {
	if len(tasks) == 0 {
		return 0, nil
	}

	args := make([]any, 0, len(tasks))
	for _, task := range tasks {
		args = append(args, task.ID)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE tasks SET parent_id = NULL WHERE parent_id IN (`+placeholders(len(tasks))+`);`, args...); err != nil { // #nosec G202 -- placeholders are generated from slice length and task IDs are passed as query args.
		return 0, fmt.Errorf("apply fullscan project: clear deleted parent links: %w", err)
	}

	deleted := 0
	for _, task := range tasks {
		result, err := tx.ExecContext(ctx, `DELETE FROM tasks WHERE id = ? AND server_version = ?;`, task.ID, task.ServerVersion)
		if err != nil {
			return 0, fmt.Errorf("apply fullscan project: delete remote-missing task: %w", err)
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return 0, fmt.Errorf("apply fullscan project: delete rows affected: %w", err)
		}
		if affected != 1 {
			return 0, ErrTaskVersionMismatch
		}
		deleted++
	}
	return deleted, nil
}

func applyFullScanParentLinks(ctx context.Context, tx *sql.Tx, projectID string, parentUIDByUID map[string]string) error {
	if len(parentUIDByUID) == 0 {
		return nil
	}

	uidToID, err := loadProjectTaskIDsByUID(ctx, tx, projectID)
	if err != nil {
		return fmt.Errorf("apply fullscan project: load task ids: %w", err)
	}
	for uid, parentUID := range parentUIDByUID {
		taskID, ok := uidToID[uid]
		if !ok {
			continue
		}

		parentID := ""
		if parentUID != "" && strings.TrimSpace(parentUIDByUID[parentUID]) == "" {
			parentID = uidToID[parentUID]
		}
		if _, err := tx.ExecContext(ctx, `
UPDATE tasks
SET parent_id = ?,
    updated_at = CASE
        WHEN COALESCE(parent_id, '') != ? THEN CURRENT_TIMESTAMP
        ELSE updated_at
    END
WHERE id = ? AND sync_status != 'conflict';
`, nullableString(parentID), parentID, taskID); err != nil {
			return fmt.Errorf("apply fullscan project: update parent link: %w", err)
		}
	}
	return nil
}

func loadProjectTaskIDsByUID(ctx context.Context, tx *sql.Tx, projectID string) (map[string]string, error) {
	rows, err := tx.QueryContext(ctx, `
SELECT uid, id
FROM tasks
WHERE project_id = ?;
`, strings.TrimSpace(projectID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids := make(map[string]string)
	for rows.Next() {
		var uid string
		var id string
		if err := rows.Scan(&uid, &id); err != nil {
			return nil, err
		}
		if strings.TrimSpace(uid) == "" {
			continue
		}
		if _, exists := ids[uid]; exists {
			continue
		}
		ids[uid] = id
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return ids, nil
}

func denormalizedLabelNames(labels []string) string {
	labelNames := append([]string(nil), labels...)
	sort.Slice(labelNames, func(i, j int) bool {
		left := strings.ToLower(labelNames[i])
		right := strings.ToLower(labelNames[j])
		if left == right {
			return labelNames[i] < labelNames[j]
		}
		return left < right
	})
	return strings.Join(labelNames, " ")
}

func importedLabelsNull(labels []string) sql.NullString {
	if len(labels) == 0 {
		return sql.NullString{}
	}
	return sql.NullString{String: strings.Join(labels, ","), Valid: true}
}
