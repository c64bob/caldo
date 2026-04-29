package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path"
	"strings"

	"github.com/google/uuid"
)

var (
	// ErrUndoSnapshotNotFound indicates no undo snapshot exists for session/tab.
	ErrUndoSnapshotNotFound = errors.New("undo snapshot not found")
	// ErrUndoSnapshotExpired indicates the undo snapshot has expired.
	ErrUndoSnapshotExpired = errors.New("undo snapshot expired")
	// ErrUndoETagMismatch indicates current task etag no longer matches snapshot etag.
	ErrUndoETagMismatch = errors.New("undo etag mismatch")
	// ErrUndoActionNotSupported indicates the snapshot action is not yet undoable.
	ErrUndoActionNotSupported = errors.New("undo action not supported")
)

type PreparedTaskUndo struct {
	SnapshotID     string
	TaskID         string
	ActionType     string
	TodoHref       string
	ProjectID      string
	ExpectedETag   string
	RawVTODO       string
	PendingVersion int
}

// PrepareTaskUndo loads and validates the latest undo snapshot for a session/tab and marks task as pending.
func (d *Database) PrepareTaskUndo(ctx context.Context, sessionID, tabID string) (PreparedTaskUndo, error) {
	d.WriteMu.Lock()
	defer d.WriteMu.Unlock()

	tx, err := d.Conn.BeginTx(ctx, nil)
	if err != nil {
		return PreparedTaskUndo{}, fmt.Errorf("prepare task undo: begin transaction: %w", err)
	}
	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()

	var snapshotID, taskID, actionType, snapshotVTODO, snapshotFields string
	var etagAtSnapshot sql.NullString
	var isExpired bool
	if err := tx.QueryRowContext(ctx, `
SELECT id, task_id, action_type, snapshot_vtodo, snapshot_fields, etag_at_snapshot, expires_at <= CURRENT_TIMESTAMP
FROM undo_snapshots
WHERE session_id = ? AND tab_id = ?;
`, sessionID, tabID).Scan(&snapshotID, &taskID, &actionType, &snapshotVTODO, &snapshotFields, &etagAtSnapshot, &isExpired); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return PreparedTaskUndo{}, ErrUndoSnapshotNotFound
		}
		return PreparedTaskUndo{}, fmt.Errorf("prepare task undo: load snapshot: %w", err)
	}

	if isExpired {
		if _, err := tx.ExecContext(ctx, `DELETE FROM undo_snapshots WHERE id = ?;`, snapshotID); err != nil {
			return PreparedTaskUndo{}, fmt.Errorf("prepare task undo: delete expired snapshot: %w", err)
		}
		if err := tx.Commit(); err != nil {
			return PreparedTaskUndo{}, fmt.Errorf("prepare task undo: commit expired snapshot delete: %w", err)
		}
		tx = nil
		return PreparedTaskUndo{}, ErrUndoSnapshotExpired
	}

	switch actionType {
	case "task_updated":
		return d.prepareTaskUpdateUndo(ctx, tx, snapshotID, taskID, snapshotVTODO, snapshotFields, etagAtSnapshot)
	case "task_deleted":
		return d.prepareTaskDeletedUndo(ctx, tx, snapshotID, snapshotVTODO, snapshotFields)
	default:
		return PreparedTaskUndo{}, ErrUndoActionNotSupported
	}
}

func (d *Database) prepareTaskUpdateUndo(ctx context.Context, tx *sql.Tx, snapshotID, taskID, snapshotVTODO, snapshotFields string, etagAtSnapshot sql.NullString) (PreparedTaskUndo, error) {
	var currentETag sql.NullString
	var href string
	var version int
	if err := tx.QueryRowContext(ctx, `SELECT etag, href, server_version FROM tasks WHERE id = ?;`, taskID).Scan(&currentETag, &href, &version); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return PreparedTaskUndo{}, ErrTaskNotFound
		}
		return PreparedTaskUndo{}, fmt.Errorf("prepare task undo: load current task: %w", err)
	}

	if nullableString(currentETag.String) != nullableString(etagAtSnapshot.String) {
		if _, err := tx.ExecContext(ctx, `UPDATE tasks SET sync_status = 'conflict', updated_at = CURRENT_TIMESTAMP WHERE id = ?;`, taskID); err != nil {
			return PreparedTaskUndo{}, fmt.Errorf("prepare task undo: mark task conflict: %w", err)
		}
		if err := tx.Commit(); err != nil {
			return PreparedTaskUndo{}, fmt.Errorf("prepare task undo: commit conflict state: %w", err)
		}
		tx = nil
		return PreparedTaskUndo{}, ErrUndoETagMismatch
	}

	result, err := tx.ExecContext(ctx, `
UPDATE tasks
SET raw_vtodo = ?,
    title = json_extract(?, '$.title'),
    description = json_extract(?, '$.description'),
    status = json_extract(?, '$.status'),
    due_date = json_extract(?, '$.due_date'),
    due_at = json_extract(?, '$.due_at'),
    priority = json_extract(?, '$.priority'),
    label_names = json_extract(?, '$.label_names'),
    sync_status = 'pending',
    server_version = server_version + 1,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ? AND server_version = ?;
`, snapshotVTODO, snapshotFields, snapshotFields, snapshotFields, snapshotFields, snapshotFields, snapshotFields, snapshotFields, taskID, version)
	if err != nil {
		return PreparedTaskUndo{}, fmt.Errorf("prepare task undo: update pending task: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return PreparedTaskUndo{}, fmt.Errorf("prepare task undo: read affected rows: %w", err)
	}
	if affected != 1 {
		return PreparedTaskUndo{}, ErrTaskVersionMismatch
	}

	if err := tx.Commit(); err != nil {
		return PreparedTaskUndo{}, fmt.Errorf("prepare task undo: commit transaction: %w", err)
	}
	tx = nil

	return PreparedTaskUndo{SnapshotID: snapshotID, TaskID: taskID, ActionType: "task_updated", TodoHref: href, ExpectedETag: currentETag.String, RawVTODO: snapshotVTODO, PendingVersion: version + 1}, nil
}

func (d *Database) prepareTaskDeletedUndo(ctx context.Context, tx *sql.Tx, snapshotID, snapshotVTODO, snapshotFields string) (PreparedTaskUndo, error) {
	var projectID string
	if err := tx.QueryRowContext(ctx, `SELECT json_extract(?, '$.project_id');`, snapshotFields).Scan(&projectID); err != nil {
		return PreparedTaskUndo{}, fmt.Errorf("prepare task undo: load deleted snapshot project id: %w", err)
	}
	var calendarHref string
	var projectName string
	if err := tx.QueryRowContext(ctx, `SELECT calendar_href, display_name FROM projects WHERE id = ?;`, projectID).Scan(&calendarHref, &projectName); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return PreparedTaskUndo{}, ErrTaskNotFound
		}
		return PreparedTaskUndo{}, fmt.Errorf("prepare task undo: load deleted snapshot project: %w", err)
	}

	taskID := uuid.NewString()
	uid := extractUIDFromVTODO(snapshotVTODO)
	href := buildTaskHref(calendarHref, uid)

	result, err := tx.ExecContext(ctx, `
INSERT INTO tasks (
    id, project_id, uid, href, title, description, status, due_date, due_at, priority, label_names,
    raw_vtodo, base_vtodo, project_name, sync_status, created_at, updated_at
) VALUES (
    ?, ?, ?, ?, json_extract(?, '$.title'), json_extract(?, '$.description'), json_extract(?, '$.status'),
    json_extract(?, '$.due_date'), json_extract(?, '$.due_at'), json_extract(?, '$.priority'), json_extract(?, '$.label_names'),
    ?, ?, ?, 'pending', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
);
`, taskID, projectID, uid, href, snapshotFields, snapshotFields, snapshotFields, snapshotFields, snapshotFields, snapshotFields, snapshotFields, snapshotVTODO, snapshotVTODO, projectName)
	if err != nil {
		return PreparedTaskUndo{}, fmt.Errorf("prepare task undo: insert deleted task pending: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return PreparedTaskUndo{}, fmt.Errorf("prepare task undo: read inserted rows: %w", err)
	}
	if affected != 1 {
		return PreparedTaskUndo{}, ErrTaskVersionMismatch
	}
	if err := tx.Commit(); err != nil {
		return PreparedTaskUndo{}, fmt.Errorf("prepare task undo: commit deleted task transaction: %w", err)
	}
	tx = nil
	return PreparedTaskUndo{SnapshotID: snapshotID, TaskID: taskID, ActionType: "task_deleted", TodoHref: href, ProjectID: projectID, RawVTODO: snapshotVTODO, PendingVersion: 0}, nil
}

func extractUIDFromVTODO(rawVTODO string) string {
	normalized := strings.ReplaceAll(rawVTODO, "\\n", "\n")
	for _, line := range strings.Split(normalized, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToUpper(trimmed), "UID:") {
			uid := strings.TrimSpace(trimmed[4:])
			if uid != "" {
				return uid
			}
		}
	}
	return uuid.NewString()
}

func buildTaskHref(calendarHref, uid string) string {
	trimmed := strings.TrimSpace(calendarHref)
	if strings.HasSuffix(trimmed, "/") {
		return trimmed + uid + ".ics"
	}
	return path.Clean(trimmed + "/" + uid + ".ics")
}

// DeleteUndoSnapshot deletes a single undo snapshot by id.
func (d *Database) DeleteUndoSnapshot(ctx context.Context, snapshotID string) error {
	d.WriteMu.Lock()
	defer d.WriteMu.Unlock()
	_, err := d.Conn.ExecContext(ctx, `DELETE FROM undo_snapshots WHERE id = ?;`, snapshotID)
	if err != nil {
		return fmt.Errorf("delete undo snapshot: %w", err)
	}
	return nil
}
