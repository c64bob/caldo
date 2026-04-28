package db

import (
	"context"
	"path/filepath"
	"testing"
)

func TestTasksFTSTriggersKeepIndexInSync(t *testing.T) {
	t.Parallel()

	database, err := OpenSQLite(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() {
		if err := database.Close(); err != nil {
			t.Fatalf("close sqlite: %v", err)
		}
	})

	ctx := context.Background()
	seedFTSProject(t, database)

	if _, err := database.Conn.ExecContext(ctx, `
INSERT INTO tasks (
    id, project_id, uid, href, etag, server_version, title, description, status, raw_vtodo, base_vtodo,
    label_names, project_name, sync_status, created_at, updated_at
) VALUES (
    'task-fts-1', 'project-1', 'uid-fts-1', '/calendars/work/task-fts-1.ics', '"etag-1"', 1,
    'Old title', 'Old description', 'needs-action', 'BEGIN:VTODO\nUID:uid-fts-1\nEND:VTODO',
    'BEGIN:VTODO\nUID:uid-fts-1\nEND:VTODO', 'old-label', 'Work', 'synced', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
);
`); err != nil {
		t.Fatalf("insert task: %v", err)
	}

	assertSingleIntResult(t, database, `SELECT COUNT(*) FROM tasks_fts WHERE rowid = (SELECT rowid FROM tasks WHERE id = 'task-fts-1');`, 1)

	if _, err := database.Conn.ExecContext(ctx, `
UPDATE tasks
SET title = 'New title',
    description = 'New description',
    label_names = 'new-label',
    project_name = 'Renamed Work',
    updated_at = CURRENT_TIMESTAMP
WHERE id = 'task-fts-1';
`); err != nil {
		t.Fatalf("update task: %v", err)
	}

	assertSingleIntResult(t, database, `SELECT COUNT(*) FROM tasks_fts WHERE tasks_fts MATCH 'old*';`, 0)
	assertSingleIntResult(t, database, `SELECT COUNT(*) FROM tasks_fts WHERE tasks_fts MATCH 'new*';`, 1)

	if _, err := database.Conn.ExecContext(ctx, `DELETE FROM tasks WHERE id = 'task-fts-1';`); err != nil {
		t.Fatalf("delete task: %v", err)
	}

	assertSingleIntResult(t, database, `SELECT COUNT(*) FROM tasks_fts;`, 0)
}

func TestTasksFTSSupportsDiacriticsPrefixAndDefaultCompletedExclusion(t *testing.T) {
	t.Parallel()

	database, err := OpenSQLite(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() {
		if err := database.Close(); err != nil {
			t.Fatalf("close sqlite: %v", err)
		}
	})

	ctx := context.Background()
	seedFTSProject(t, database)

	if _, err := database.Conn.ExecContext(ctx, `
INSERT INTO tasks (
    id, project_id, uid, href, etag, server_version, title, description, status, raw_vtodo, base_vtodo,
    label_names, project_name, sync_status, created_at, updated_at
) VALUES
(
    'task-active', 'project-1', 'uid-active', '/calendars/work/task-active.ics', '"etag-active"', 1,
    'Überweisung Rechnung', 'Prüfen', 'needs-action', 'BEGIN:VTODO\nUID:uid-active\nEND:VTODO',
    'BEGIN:VTODO\nUID:uid-active\nEND:VTODO', 'Büro dringend', 'Finänzen', 'synced', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
),
(
    'task-completed', 'project-1', 'uid-completed', '/calendars/work/task-completed.ics', '"etag-completed"', 1,
    'Überfällige Rechnung', 'Archiv', 'completed', 'BEGIN:VTODO\nUID:uid-completed\nEND:VTODO',
    'BEGIN:VTODO\nUID:uid-completed\nEND:VTODO', 'Büro erledigt', 'Finänzen', 'synced', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
);
`); err != nil {
		t.Fatalf("insert tasks: %v", err)
	}

	assertSingleIntResult(t, database, `SELECT COUNT(*) FROM tasks_fts WHERE tasks_fts MATCH 'uber*';`, 2)
	assertSingleIntResult(t, database, `SELECT COUNT(*) FROM tasks_fts WHERE tasks_fts MATCH 'finanz*';`, 2)
	assertSingleIntResult(t, database, `SELECT COUNT(*) FROM tasks_fts WHERE tasks_fts MATCH 'buro*';`, 2)
	assertSingleIntResult(t, database, `
SELECT COUNT(*)
FROM tasks_fts f
JOIN tasks t ON t.rowid = f.rowid
WHERE f.tasks_fts MATCH 'rech*'
  AND t.status != 'completed';
`, 1)
}

func seedFTSProject(t *testing.T, database *Database) {
	t.Helper()

	if _, err := database.Conn.Exec(`
INSERT INTO projects (
    id, calendar_href, display_name, sync_strategy, server_version, created_at, updated_at
) VALUES (
    'project-1', '/calendars/work', 'Work', 'fullscan', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
);
`); err != nil {
		t.Fatalf("insert project: %v", err)
	}
}
