package db

import (
	"context"
	"path/filepath"
	"testing"
)

func TestBuildSearchMatchExprRemovesHyphenFromBarewords(t *testing.T) {
	t.Parallel()

	matchExpr := buildSearchMatchExpr("on-call #Team-A @p1-high")

	const expected = "(title:oncall* OR description:oncall* OR label_names:oncall* OR project_name:oncall*) AND project_name:teama* AND label_names:p1high*"
	if matchExpr != expected {
		t.Fatalf("unexpected match expression: got %q want %q", matchExpr, expected)
	}
}

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

	result := results[0]
	if result.Description != "Prüfen" ||
		result.ProjectName != "Finanzen" ||
		result.LabelNames != "Büro,dringend" ||
		result.DueISODate != "2026-06-09" ||
		result.Priority != 1 ||
		!result.HasPriority ||
		result.SyncStatus != "error" ||
		result.ServerVersion != 5 ||
		result.RawVTODO == "" {
		t.Fatalf("search result missing row metadata: %#v", result)
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

func TestSearchActiveTasksIncludesSubtaskRelationshipMetadata(t *testing.T) {
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
	if _, err := database.Conn.Exec(`
INSERT INTO tasks (
    id, project_id, uid, href, etag, server_version, title, description, status, parent_id, raw_vtodo, base_vtodo,
    label_names, project_name, sync_status, due_date, priority, created_at, updated_at
) VALUES (
    'task-child', 'project-1', 'uid-child', '/calendars/work/task-child.ics', '"etag-child"', 1,
    'Rechnung Unteraufgabe', 'Kind', 'needs-action', 'task-active', 'BEGIN:VTODO\nUID:uid-child\nRELATED-TO;RELTYPE=PARENT:uid-active\nEND:VTODO',
    'BEGIN:VTODO\nUID:uid-child\nRELATED-TO;RELTYPE=PARENT:uid-active\nEND:VTODO', 'Büro', 'Finanzen', 'synced', '2026-06-09', NULL, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
),
(
    'task-child-done', 'project-1', 'uid-child-done', '/calendars/work/task-child-done.ics', '"etag-child-done"', 1,
    'Erledigte Rechnung Unteraufgabe', 'Kind erledigt', 'completed', 'task-active', 'BEGIN:VTODO\nUID:uid-child-done\nRELATED-TO;RELTYPE=PARENT:uid-active\nEND:VTODO',
    'BEGIN:VTODO\nUID:uid-child-done\nRELATED-TO;RELTYPE=PARENT:uid-active\nEND:VTODO', 'Büro', 'Finanzen', 'synced', '2026-06-09', NULL, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
);
`); err != nil {
		t.Fatalf("insert child task: %v", err)
	}

	results, err := database.SearchActiveTasks(context.Background(), "rechnung", 25)
	if err != nil {
		t.Fatalf("search active tasks: %v", err)
	}

	var parent SearchResult
	var child SearchResult
	for _, result := range results {
		switch result.ID {
		case "task-active":
			parent = result
		case "task-child":
			child = result
		}
	}
	if parent.ID == "" || child.ID == "" {
		t.Fatalf("expected parent and child in search results: %#v", results)
	}
	if parent.SubtaskCount != 2 || parent.OpenSubtaskCount != 1 || parent.IsSubtask {
		t.Fatalf("parent missing subtask count metadata: %#v", parent)
	}
	if child.ParentID != "task-active" ||
		child.ParentTitle != "Überweisung Rechnung" ||
		!child.IsSubtask ||
		child.SubtaskCount != 0 ||
		child.OpenSubtaskCount != 0 {
		t.Fatalf("child missing relationship metadata: %#v", child)
	}
}

func seedSearchTasks(t *testing.T, database *Database) {
	t.Helper()

	if _, err := database.Conn.Exec(`
INSERT INTO tasks (
    id, project_id, uid, href, etag, server_version, title, description, status, raw_vtodo, base_vtodo,
    label_names, project_name, sync_status, due_date, priority, created_at, updated_at
) VALUES
(
    'task-active', 'project-1', 'uid-active', '/calendars/work/task-active.ics', '"etag-active"', 5,
    'Überweisung Rechnung', 'Prüfen', 'needs-action', 'BEGIN:VTODO\nUID:uid-active\nEND:VTODO',
    'BEGIN:VTODO\nUID:uid-active\nEND:VTODO', 'Büro,dringend', 'Finanzen', 'error', '2026-06-09', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
),
(
    'task-completed', 'project-1', 'uid-completed', '/calendars/work/task-completed.ics', '"etag-completed"', 1,
    'Überfällige Rechnung', 'Archiv', 'completed', 'BEGIN:VTODO\nUID:uid-completed\nEND:VTODO',
    'BEGIN:VTODO\nUID:uid-completed\nEND:VTODO', 'Büro,erledigt', 'Finanzen', 'synced', '2026-06-08', NULL, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
);
`); err != nil {
		t.Fatalf("insert tasks: %v", err)
	}
}
