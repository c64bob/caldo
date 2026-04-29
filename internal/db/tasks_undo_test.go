package db

import (
	"context"
	"path/filepath"
	"testing"
)

func TestPrepareTaskUndoLoadsSnapshotMarksPending(t *testing.T) {
	t.Parallel()
	db := openTaskUndoTestDB(t)
	seedTaskUndoData(t, db)

	prepared, err := db.PrepareTaskUndo(context.Background(), "session-1", "tab-1")
	if err != nil {
		t.Fatalf("prepare undo: %v", err)
	}
	if prepared.TaskID != "task-1" || prepared.PendingVersion != 4 {
		t.Fatalf("unexpected prepared: %+v", prepared)
	}

	assertSingleTextResult(t, db, `SELECT sync_status FROM tasks WHERE id='task-1';`, "pending")
	assertSingleIntResult(t, db, `SELECT server_version FROM tasks WHERE id='task-1';`, 4)
	assertSingleTextResult(t, db, `SELECT title FROM tasks WHERE id='task-1';`, "before")
	assertSingleTextResult(t, db, `SELECT COALESCE(description, '') FROM tasks WHERE id='task-1';`, "before description")
	assertSingleTextResult(t, db, `SELECT status FROM tasks WHERE id='task-1';`, "completed")
	assertSingleTextResult(t, db, `SELECT COALESCE(due_date, '') FROM tasks WHERE id='task-1';`, "2026-05-01")
	assertSingleTextResult(t, db, `SELECT COALESCE(due_at, '') FROM tasks WHERE id='task-1';`, "2026-05-01T09:00:00Z")
	assertSingleIntResult(t, db, `SELECT COALESCE(priority, 0) FROM tasks WHERE id='task-1';`, 3)
	assertSingleTextResult(t, db, `SELECT COALESCE(label_names, '') FROM tasks WHERE id='task-1';`, "alpha,beta")
}

func TestPrepareTaskUndoRejectsExpiredSnapshotAndDeletesIt(t *testing.T) {
	t.Parallel()
	db := openTaskUndoTestDB(t)
	seedTaskUndoData(t, db)

	_, err := db.PrepareTaskUndo(context.Background(), "session-exp", "tab-exp")
	if err != ErrUndoSnapshotExpired {
		t.Fatalf("expected expired, got %v", err)
	}
	assertSingleIntResult(t, db, `SELECT COUNT(*) FROM undo_snapshots WHERE session_id='session-exp' AND tab_id='tab-exp';`, 0)
}

func TestPrepareTaskUndoETagMismatchMarksConflict(t *testing.T) {
	t.Parallel()
	db := openTaskUndoTestDB(t)
	seedTaskUndoData(t, db)

	_, err := db.PrepareTaskUndo(context.Background(), "session-mm", "tab-mm")
	if err != ErrUndoETagMismatch {
		t.Fatalf("expected etag mismatch, got %v", err)
	}
	assertSingleTextResult(t, db, `SELECT sync_status FROM tasks WHERE id='task-1';`, "conflict")
}

func TestPrepareTaskUndoDeletedTaskInsertsPendingCreate(t *testing.T) {
	t.Parallel()
	db := openTaskUndoTestDB(t)
	seedTaskUndoData(t, db)

	prepared, err := db.PrepareTaskUndo(context.Background(), "session-del", "tab-del")
	if err != nil {
		t.Fatalf("prepare deleted undo: %v", err)
	}
	if prepared.ActionType != "task_deleted" {
		t.Fatalf("unexpected action type: %s", prepared.ActionType)
	}
	assertSingleTextResult(t, db, `SELECT uid FROM tasks WHERE id='`+prepared.TaskID+`';`, "uid-del")
	assertSingleTextResult(t, db, `SELECT href FROM tasks WHERE id='`+prepared.TaskID+`';`, "/cal/inbox/uid-del.ics")
	assertSingleTextResult(t, db, `SELECT sync_status FROM tasks WHERE id='`+prepared.TaskID+`';`, "pending")
}

func TestPrepareTaskUndoDeletedTaskReusesPendingRowOnRetry(t *testing.T) {
	t.Parallel()
	db := openTaskUndoTestDB(t)
	seedTaskUndoData(t, db)

	first, err := db.PrepareTaskUndo(context.Background(), "session-del", "tab-del")
	if err != nil {
		t.Fatalf("first prepare deleted undo: %v", err)
	}
	if err := db.MarkTaskCreateError(context.Background(), first.TaskID); err != nil {
		t.Fatalf("mark create error: %v", err)
	}

	second, err := db.PrepareTaskUndo(context.Background(), "session-del", "tab-del")
	if err != nil {
		t.Fatalf("second prepare deleted undo: %v", err)
	}
	if second.TaskID != first.TaskID {
		t.Fatalf("expected retry to reuse task id %s, got %s", first.TaskID, second.TaskID)
	}

	assertSingleIntResult(t, db, `SELECT COUNT(*) FROM tasks WHERE project_id='project-1' AND uid='uid-del';`, 1)
	assertSingleTextResult(t, db, `SELECT sync_status FROM tasks WHERE id='`+second.TaskID+`';`, "pending")
}
func openTaskUndoTestDB(t *testing.T) *Database {
	t.Helper()
	database, err := OpenSQLite(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	return database
}

func seedTaskUndoData(t *testing.T, database *Database) {
	t.Helper()
	if _, err := database.Conn.ExecContext(context.Background(), `
INSERT INTO projects (id, calendar_href, display_name, sync_strategy, is_default, created_at, updated_at)
VALUES ('project-1', '/cal/inbox/', 'Inbox', 'fullscan', TRUE, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
INSERT INTO tasks (
    id, project_id, uid, href, etag, server_version, title, status, raw_vtodo, base_vtodo,
    project_name, sync_status, created_at, updated_at
) VALUES (
    'task-1', 'project-1', 'uid-1', '/cal/inbox/uid-1.ics', '"etag-1"', 3, 'current', 'needs-action',
    'BEGIN:VTODO\nUID:uid-1\nSUMMARY:old\nEND:VTODO', 'BEGIN:VTODO\nUID:uid-1\nSUMMARY:old\nEND:VTODO',
    'Inbox', 'synced', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
);
INSERT INTO undo_snapshots (id, session_id, tab_id, task_id, action_type, snapshot_vtodo, snapshot_fields, etag_at_snapshot, created_at, expires_at)
VALUES
('undo-1','session-1','tab-1','task-1','task_updated','BEGIN:VTODO\nUID:uid-1\nSUMMARY:before\nEND:VTODO',json_object('title','before','description','before description','status','completed','due_date','2026-05-01','due_at','2026-05-01T09:00:00Z','priority',3,'label_names','alpha,beta'),'"etag-1"',CURRENT_TIMESTAMP,DATETIME(CURRENT_TIMESTAMP,'+5 minutes')),
('undo-exp','session-exp','tab-exp','task-1','task_updated','BEGIN:VTODO\nUID:uid-1\nSUMMARY:before\nEND:VTODO',json_object('title','before','description','before description','status','completed','due_date','2026-05-01','due_at','2026-05-01T09:00:00Z','priority',3,'label_names','alpha,beta'),'"etag-1"',CURRENT_TIMESTAMP,DATETIME(CURRENT_TIMESTAMP,'-1 minutes')),
('undo-mm','session-mm','tab-mm','task-1','task_updated','BEGIN:VTODO\nUID:uid-1\nSUMMARY:before\nEND:VTODO',json_object('title','before','description','before description','status','completed','due_date','2026-05-01','due_at','2026-05-01T09:00:00Z','priority',3,'label_names','alpha,beta'),'"etag-old"',CURRENT_TIMESTAMP,DATETIME(CURRENT_TIMESTAMP,'+5 minutes')),
('undo-del','session-del','tab-del','task-gone','task_deleted','BEGIN:VTODO\nUID:uid-del\nSUMMARY:deleted\nEND:VTODO',json_object('project_id','project-1','title','deleted','description','deleted desc','status','needs-action','label_names','home'),'"etag-del"',CURRENT_TIMESTAMP,DATETIME(CURRENT_TIMESTAMP,'+5 minutes'));
`); err != nil {
		t.Fatalf("seed undo data: %v", err)
	}
}
