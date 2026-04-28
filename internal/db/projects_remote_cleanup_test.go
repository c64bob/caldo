package db

import (
	"context"
	"path/filepath"
	"testing"
)

func TestCleanupRemoteDeletedCalendarsRemovesProjectDataAndReturnsWarnings(t *testing.T) {
	t.Parallel()

	database, err := OpenSQLite(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	seedRemoteCleanupProjects(t, database)

	if _, err := database.Conn.ExecContext(context.Background(), `
CREATE VIRTUAL TABLE tasks_fts USING fts5(title, description, label_names, project_name);
INSERT INTO tasks_fts(rowid, title, description, label_names, project_name)
SELECT rowid, title, description, label_names, project_name FROM tasks WHERE project_id = 'project-deleted';
`); err != nil {
		t.Fatalf("seed fts entries: %v", err)
	}

	cleanups, err := database.CleanupRemoteDeletedCalendars(context.Background(), []string{"/cal/keep/"})
	if err != nil {
		t.Fatalf("cleanup remote deleted calendars: %v", err)
	}
	if len(cleanups) != 1 {
		t.Fatalf("unexpected cleanup count: got %d want %d", len(cleanups), 1)
	}

	cleanup := cleanups[0]
	if cleanup.ProjectID != "project-deleted" || cleanup.CalendarHref != "/cal/deleted/" {
		t.Fatalf("unexpected cleanup project: %#v", cleanup)
	}
	if !cleanup.HadPendingTasks {
		t.Fatal("expected pending task warning flag")
	}
	if !cleanup.WasDefaultProject {
		t.Fatal("expected default project flag")
	}

	assertSingleIntResult(t, database, `SELECT COUNT(*) FROM projects WHERE id = 'project-deleted';`, 0)
	assertSingleIntResult(t, database, `SELECT COUNT(*) FROM tasks WHERE project_id = 'project-deleted';`, 0)
	assertSingleIntResult(t, database, `SELECT COUNT(*) FROM undo_snapshots WHERE task_id IN ('task-deleted-1','task-deleted-2');`, 0)
	assertSingleIntResult(t, database, `SELECT COUNT(*) FROM conflicts WHERE project_id = 'project-deleted' OR task_id IN ('task-deleted-1','task-deleted-2');`, 0)
	assertSingleIntResult(t, database, `SELECT COUNT(*) FROM tasks_fts;`, 0)

	assertSingleIntResult(t, database, `SELECT COUNT(*) FROM projects WHERE id = 'project-keep';`, 1)
	assertSingleIntResult(t, database, `SELECT COUNT(*) FROM tasks WHERE project_id = 'project-keep';`, 1)

	assertSingleIntResult(t, database, `SELECT COUNT(*) FROM settings WHERE id = 'default' AND default_project_id IS NULL;`, 1)
}

func TestCleanupRemoteDeletedCalendarsKeepsAllWhenRemotesExist(t *testing.T) {
	t.Parallel()

	database, err := OpenSQLite(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	seedRemoteCleanupProjects(t, database)

	cleanups, err := database.CleanupRemoteDeletedCalendars(context.Background(), []string{"/cal/deleted/", "/cal/keep/", "/cal/keep/"})
	if err != nil {
		t.Fatalf("cleanup remote deleted calendars: %v", err)
	}
	if len(cleanups) != 0 {
		t.Fatalf("expected no cleanup rows, got %d", len(cleanups))
	}

	assertSingleIntResult(t, database, `SELECT COUNT(*) FROM projects;`, 2)
	assertSingleIntResult(t, database, `SELECT COUNT(*) FROM tasks;`, 3)
}

func seedRemoteCleanupProjects(t *testing.T, database *Database) {
	t.Helper()

	if _, err := database.Conn.ExecContext(context.Background(), `
INSERT INTO projects (id, calendar_href, display_name, sync_strategy, created_at, updated_at)
VALUES
	('project-deleted', '/cal/deleted/', 'Deleted', 'fullscan', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
	('project-keep', '/cal/keep/', 'Keep', 'fullscan', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);

INSERT INTO tasks (
	id, project_id, uid, href, etag, title, description, status, raw_vtodo, label_names, project_name, sync_status, created_at, updated_at
) VALUES
	('task-deleted-1', 'project-deleted', 'uid-1', '/cal/deleted/1.ics', 'etag-1', 'Task 1', 'Desc 1', 'needs-action', 'BEGIN:VTODO\nUID:uid-1\nEND:VTODO', 'home', 'Deleted', 'pending', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
	('task-deleted-2', 'project-deleted', 'uid-2', '/cal/deleted/2.ics', 'etag-2', 'Task 2', 'Desc 2', 'needs-action', 'BEGIN:VTODO\nUID:uid-2\nEND:VTODO', 'work', 'Deleted', 'synced', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
	('task-keep-1', 'project-keep', 'uid-3', '/cal/keep/1.ics', 'etag-3', 'Task 3', 'Desc 3', 'needs-action', 'BEGIN:VTODO\nUID:uid-3\nEND:VTODO', 'life', 'Keep', 'synced', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);

INSERT INTO undo_snapshots (
	id, session_id, tab_id, task_id, action_type, snapshot_vtodo, snapshot_fields, created_at, expires_at
) VALUES
	('undo-deleted', 'session-1', 'tab-1', 'task-deleted-1', 'task_updated', 'BEGIN:VTODO\nUID:uid-1\nEND:VTODO', '{}', CURRENT_TIMESTAMP, DATETIME(CURRENT_TIMESTAMP, '+5 minutes')),
	('undo-keep', 'session-1', 'tab-2', 'task-keep-1', 'task_updated', 'BEGIN:VTODO\nUID:uid-3\nEND:VTODO', '{}', CURRENT_TIMESTAMP, DATETIME(CURRENT_TIMESTAMP, '+5 minutes'));

INSERT INTO conflicts (
	id, task_id, project_id, conflict_type, created_at, base_vtodo, remote_vtodo
) VALUES
	('conflict-deleted', 'task-deleted-2', 'project-deleted', 'field_conflict', CURRENT_TIMESTAMP, 'BEGIN:VTODO\nUID:uid-2\nEND:VTODO', 'BEGIN:VTODO\nUID:uid-2\nSUMMARY:remote\nEND:VTODO'),
	('conflict-keep', 'task-keep-1', 'project-keep', 'field_conflict', CURRENT_TIMESTAMP, 'BEGIN:VTODO\nUID:uid-3\nEND:VTODO', 'BEGIN:VTODO\nUID:uid-3\nSUMMARY:remote\nEND:VTODO');

UPDATE settings SET default_project_id = 'project-deleted' WHERE id = 'default';
`); err != nil {
		t.Fatalf("seed remote cleanup data: %v", err)
	}
}
