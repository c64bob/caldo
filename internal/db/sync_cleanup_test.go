package db

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestCleanupSyncArtifactsDeletesExpiredUndoAndOldResolvedConflicts(t *testing.T) {
	t.Parallel()
	database := openCleanupSyncTestDB(t)

	now := time.Now().UTC()
	_, err := database.Conn.ExecContext(context.Background(), `
INSERT INTO projects (id, calendar_href, display_name, sync_strategy, created_at, updated_at)
VALUES ('project-1', '/cal/p1/', 'Project 1', 'fullscan', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);

INSERT INTO undo_snapshots (id, session_id, tab_id, task_id, action_type, snapshot_vtodo, snapshot_fields, created_at, expires_at)
VALUES
	('undo-expired', 's1', 't1', 'task-1', 'task_updated', 'BEGIN:VTODO\nEND:VTODO', '{}', CURRENT_TIMESTAMP, ?),
	('undo-active', 's1', 't2', 'task-2', 'task_updated', 'BEGIN:VTODO\nEND:VTODO', '{}', CURRENT_TIMESTAMP, ?);

INSERT INTO conflicts (id, task_id, project_id, conflict_type, created_at, resolved_at)
VALUES
	('conflict-old-resolved', 'task-1', 'project-1', 'field_conflict', CURRENT_TIMESTAMP, DATETIME(CURRENT_TIMESTAMP, '-8 days')) ,
	('conflict-recent-resolved', 'task-2', 'project-1', 'field_conflict', CURRENT_TIMESTAMP, DATETIME(CURRENT_TIMESTAMP, '-2 days')),
	('conflict-unresolved', 'task-3', 'project-1', 'field_conflict', CURRENT_TIMESTAMP, NULL);
`, now.Add(-time.Minute), now.Add(time.Minute), now.Add(-time.Minute), now.Add(time.Minute))
	if err != nil {
		t.Fatalf("seed cleanup data: %v", err)
	}

	result, err := database.CleanupSyncArtifacts(context.Background(), now, true)
	if err != nil {
		t.Fatalf("cleanup sync artifacts: %v", err)
	}
	if result.ExpiredUndoDeleted != 1 || result.ResolvedConflictsDeleted != 1 {
		t.Fatalf("unexpected cleanup result: %#v", result)
	}

	assertSingleIntResult(t, database, `SELECT COUNT(*) FROM undo_snapshots;`, 1)
	assertSingleIntResult(t, database, `SELECT COUNT(*) FROM conflicts;`, 2)
	assertSingleIntResult(t, database, `SELECT COUNT(*) FROM conflicts WHERE id = 'conflict-unresolved';`, 1)
}

func TestCleanupSyncArtifactsSkipsResolvedConflictCleanupWhenDisabled(t *testing.T) {
	t.Parallel()
	database := openCleanupSyncTestDB(t)
	now := time.Now().UTC()

	if _, err := database.Conn.ExecContext(context.Background(), `
INSERT INTO projects (id, calendar_href, display_name, sync_strategy, created_at, updated_at)
VALUES ('project-1', '/cal/p1/', 'Project 1', 'fullscan', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
INSERT INTO conflicts (id, task_id, project_id, conflict_type, created_at, resolved_at)
VALUES ('conflict-old-resolved', 'task-1', 'project-1', 'field_conflict', CURRENT_TIMESTAMP, DATETIME(CURRENT_TIMESTAMP, '-10 days'));
`); err != nil {
		t.Fatalf("seed conflict: %v", err)
	}

	result, err := database.CleanupSyncArtifacts(context.Background(), now, false)
	if err != nil {
		t.Fatalf("cleanup sync artifacts: %v", err)
	}
	if result.ResolvedConflictsDeleted != 0 {
		t.Fatalf("unexpected resolved cleanup count: %#v", result)
	}
	assertSingleIntResult(t, database, `SELECT COUNT(*) FROM conflicts WHERE id = 'conflict-old-resolved';`, 1)
}

func openCleanupSyncTestDB(t *testing.T) *Database {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "caldo.db")
	database, err := OpenSQLite(dbPath)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() {
		if err := database.Close(); err != nil {
			t.Fatalf("close sqlite: %v", err)
		}
	})
	return database
}
