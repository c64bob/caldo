package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"caldo/internal/model"
	"github.com/google/uuid"
)

// ApplyWebDAVSyncProject applies one WebDAV Sync result for a project.
func (d *Database) ApplyWebDAVSyncProject(ctx context.Context, projectID string, changed []ImportedTask, deletedHrefs []string, syncToken string, baseline bool) (FullScanApplyResult, error) {
	if strings.TrimSpace(projectID) == "" {
		return FullScanApplyResult{}, fmt.Errorf("apply webdav sync project: project id is required")
	}
	if strings.TrimSpace(syncToken) == "" {
		return FullScanApplyResult{}, fmt.Errorf("apply webdav sync project: sync token is required")
	}

	d.WriteMu.Lock()
	defer d.WriteMu.Unlock()

	tx, err := d.Conn.BeginTx(ctx, nil)
	if err != nil {
		return FullScanApplyResult{}, fmt.Errorf("apply webdav sync project: begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	projectName, err := loadProjectDisplayName(ctx, tx, projectID)
	if err != nil {
		return FullScanApplyResult{}, err
	}

	localTasks, err := loadFullScanLocalTasks(ctx, tx, projectID)
	if err != nil {
		return FullScanApplyResult{}, fmt.Errorf("apply webdav sync project: load local tasks: %w", err)
	}
	localByUID := make(map[string]fullScanLocalTask, len(localTasks))
	localByHref := make(map[string]fullScanLocalTask, len(localTasks))
	for _, task := range localTasks {
		if strings.TrimSpace(task.UID) != "" {
			if _, exists := localByUID[task.UID]; !exists {
				localByUID[task.UID] = task
			}
		}
		if strings.TrimSpace(task.Href) != "" {
			localByHref[strings.TrimSpace(task.Href)] = task
		}
	}

	remoteTasks := dedupeImportedTasks(changed, projectName)
	parentUIDByUID := make(map[string]string, len(remoteTasks))
	remoteUIDs := make(map[string]struct{}, len(remoteTasks))
	touchedLocalIDs := make(map[string]struct{}, len(remoteTasks))
	var result FullScanApplyResult
	for _, remoteTask := range remoteTasks {
		parentUIDByUID[remoteTask.UID] = strings.TrimSpace(remoteTask.ParentUID)
		remoteUIDs[remoteTask.UID] = struct{}{}
		touchedID, err := applyWebDAVChangedTask(ctx, tx, projectID, localByUID, localByHref, remoteTask, &result)
		if err != nil {
			return FullScanApplyResult{}, err
		}
		if touchedID != "" {
			touchedLocalIDs[touchedID] = struct{}{}
		}
	}

	deleteTasks := make([]fullScanLocalTask, 0, len(deletedHrefs)+len(localTasks))
	deletedSeen := make(map[string]struct{}, len(deletedHrefs))
	for _, href := range deletedHrefs {
		localTask, exists := localByHref[strings.TrimSpace(href)]
		if !exists {
			continue
		}
		if _, seen := deletedSeen[localTask.ID]; seen {
			continue
		}
		deletedSeen[localTask.ID] = struct{}{}
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
	if baseline {
		for _, localTask := range localTasks {
			if _, ok := remoteUIDs[localTask.UID]; ok {
				continue
			}
			if _, ok := touchedLocalIDs[localTask.ID]; ok {
				continue
			}
			if _, ok := deletedSeen[localTask.ID]; ok {
				continue
			}
			deletedSeen[localTask.ID] = struct{}{}
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
	if _, err := tx.ExecContext(ctx, `
UPDATE projects
SET sync_token = ?,
    sync_strategy = 'webdav_sync',
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?;
`, strings.TrimSpace(syncToken), strings.TrimSpace(projectID)); err != nil {
		return FullScanApplyResult{}, fmt.Errorf("apply webdav sync project: update sync metadata: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return FullScanApplyResult{}, fmt.Errorf("apply webdav sync project: commit transaction: %w", err)
	}
	return result, nil
}

func applyWebDAVChangedTask(ctx context.Context, tx *sql.Tx, projectID string, localByUID map[string]fullScanLocalTask, localByHref map[string]fullScanLocalTask, remoteTask ImportedTask, result *FullScanApplyResult) (string, error) {
	localTask, exists := localByUID[remoteTask.UID]
	if !exists {
		if hrefMatch, ok := localByHref[strings.TrimSpace(remoteTask.Href)]; ok {
			return hrefMatch.ID, recordWebDAVUIDConflict(ctx, tx, hrefMatch, remoteTask, result)
		}
		taskID := uuid.NewString()
		if err := insertImportedTask(ctx, tx, taskID, projectID, "", remoteTask); err != nil {
			return "", fmt.Errorf("apply webdav sync project: insert remote task: %w", err)
		}
		if err := syncTaskLabels(ctx, tx, taskID, importedLabelsNull(remoteTask.LabelNames)); err != nil {
			return "", fmt.Errorf("apply webdav sync project: sync inserted labels: %w", err)
		}
		result.Inserted++
		return "", nil
	}

	if localTask.SyncStatus == "conflict" {
		return localTask.ID, nil
	}
	if !remoteTaskChanged(localTask, remoteTask) {
		return localTask.ID, nil
	}
	if sameVTODO(localTask.RawVTODO, remoteTask.RawVTODO) || localTaskIsClean(localTask) {
		if err := updateTaskFromImported(ctx, tx, localTask, remoteTask); err != nil {
			return "", err
		}
		result.Updated++
		return localTask.ID, nil
	}

	merge := model.MergeVTODOFields(localTask.BaseVTODO, localTask.RawVTODO, remoteTask.RawVTODO)
	if merge.Merged && !merge.Conflict {
		mergedTask := importedTaskWithRaw(remoteTask, merge.MergedVTODO)
		if err := updateTaskFromImported(ctx, tx, localTask, mergedTask); err != nil {
			return "", err
		}
		result.Updated++
		return localTask.ID, nil
	}

	if err := recordFullScanTaskConflict(ctx, tx, localTask, conflictTypeFieldConflict, localTask.BaseVTODO, localTask.RawVTODO, remoteTask.RawVTODO, remoteTask.ETag); err != nil {
		return "", err
	}
	result.Conflicts++
	return localTask.ID, nil
}

func recordWebDAVUIDConflict(ctx context.Context, tx *sql.Tx, localTask fullScanLocalTask, remoteTask ImportedTask, result *FullScanApplyResult) error {
	if localTask.SyncStatus == "conflict" {
		return nil
	}
	if err := recordFullScanTaskConflict(ctx, tx, localTask, conflictTypeFieldConflict, localTask.BaseVTODO, localTask.RawVTODO, remoteTask.RawVTODO, remoteTask.ETag); err != nil {
		return err
	}
	result.Conflicts++
	return nil
}
