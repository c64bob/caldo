package db

import (
	"context"
	"path/filepath"
	"testing"
)

func TestSearchActiveTasksMatchesTextProjectAndLabelTokens(t *testing.T) {
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

	seedFTSProject(t, database)
	seedSearchTasks(t, database)

	results, err := database.SearchActiveTasks(context.Background(), "rechnung #finanzen @buRo", 25)
	if err != nil {
		t.Fatalf("search active tasks: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("unexpected result count: got %d want %d", len(results), 1)
	}
	if results[0].ID != "task-active" {
		t.Fatalf("unexpected result id: got %q want %q", results[0].ID, "task-active")
	}
}

func TestSearchActiveTasksExcludesCompletedByDefault(t *testing.T) {
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

	seedFTSProject(t, database)
	seedSearchTasks(t, database)

	results, err := database.SearchActiveTasks(context.Background(), "archiv", 25)
	if err != nil {
		t.Fatalf("search active tasks: %v", err)
	}

	if len(results) != 0 {
		t.Fatalf("unexpected result count: got %d want %d", len(results), 0)
	}
}

func seedSearchTasks(t *testing.T, database *Database) {
	t.Helper()

	if _, err := database.Conn.Exec(`
INSERT INTO tasks (
    id, project_id, uid, href, etag, server_version, title, description, status, raw_vtodo, base_vtodo,
    label_names, project_name, sync_status, created_at, updated_at
) VALUES
(
    'task-active', 'project-1', 'uid-active', '/calendars/work/task-active.ics', '"etag-active"', 1,
    'Überweisung Rechnung', 'Prüfen', 'needs-action', 'BEGIN:VTODO\nUID:uid-active\nEND:VTODO',
    'BEGIN:VTODO\nUID:uid-active\nEND:VTODO', 'Büro dringend', 'Finanzen', 'synced', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
),
(
    'task-completed', 'project-1', 'uid-completed', '/calendars/work/task-completed.ics', '"etag-completed"', 1,
    'Überfällige Rechnung', 'Archiv', 'completed', 'BEGIN:VTODO\nUID:uid-completed\nEND:VTODO',
    'BEGIN:VTODO\nUID:uid-completed\nEND:VTODO', 'Büro erledigt', 'Finanzen', 'synced', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
);
`); err != nil {
		t.Fatalf("insert tasks: %v", err)
	}
}
