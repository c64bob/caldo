package db

import (
	"context"
	"path/filepath"
	"testing"
)

func TestListTaskVersionsReturnsExistingRowsOnly(t *testing.T) {
	t.Parallel()
	database, err := OpenSQLite(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	if _, err := database.Conn.ExecContext(context.Background(), `
INSERT INTO projects (id, calendar_href, display_name, sync_strategy, created_at, updated_at)
VALUES ('project-1', '/cal/work/', 'Work', 'fullscan', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
INSERT INTO tasks (
 id, project_id, uid, href, etag, server_version, title, status, raw_vtodo, base_vtodo, project_name, sync_status, created_at, updated_at
) VALUES
('task-1', 'project-1', 'uid-1', '/cal/work/task-1.ics', '"e1"', 7, 'One', 'needs-action', 'BEGIN:VTODO\nUID:uid-1\nEND:VTODO', 'BEGIN:VTODO\nUID:uid-1\nEND:VTODO', 'Work', 'synced', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('task-2', 'project-1', 'uid-2', '/cal/work/task-2.ics', '"e2"', 3, 'Two', 'needs-action', 'BEGIN:VTODO\nUID:uid-2\nEND:VTODO', 'BEGIN:VTODO\nUID:uid-2\nEND:VTODO', 'Work', 'synced', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
`); err != nil {
		t.Fatalf("seed tasks: %v", err)
	}

	rows, err := database.ListTaskVersions(context.Background(), []string{"task-1", "missing", "task-2"})
	if err != nil {
		t.Fatalf("list task versions: %v", err)
	}

	if len(rows) != 2 {
		t.Fatalf("unexpected row count: got %d", len(rows))
	}

	versions := map[string]int{}
	for _, row := range rows {
		versions[row.TaskID] = row.ServerVersion
	}
	if versions["task-1"] != 7 || versions["task-2"] != 3 {
		t.Fatalf("unexpected versions: %#v", versions)
	}
}
