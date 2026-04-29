package db

import (
	"context"
	"path/filepath"
	"testing"
)

func TestMarkConflictResolvedChecksExpectedVersion(t *testing.T) {
	t.Parallel()
	database, err := OpenSQLite(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })

	_, err = database.Conn.ExecContext(context.Background(), `
INSERT INTO projects (id, calendar_href, display_name, sync_strategy, created_at, updated_at)
VALUES ('project-1','/p','Inbox','ctag',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP);
INSERT INTO tasks (id, project_id, uid, href, etag, title, status, raw_vtodo, sync_status, server_version, created_at, updated_at)
VALUES ('task-1','project-1','uid-1','/t','e','Task 1','needs-action','BEGIN:VTODO\r\nSUMMARY:Task 1\r\nSTATUS:NEEDS-ACTION\r\nEND:VTODO\r\n','conflict',2,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP);
INSERT INTO conflicts (id, task_id, project_id, conflict_type, created_at, base_vtodo, local_vtodo, remote_vtodo)
VALUES ('open-1','task-1','project-1','field_conflict',CURRENT_TIMESTAMP,'b','BEGIN:VTODO\r\nSUMMARY:Task 1\r\nSTATUS:NEEDS-ACTION\r\nEND:VTODO\r\n','BEGIN:VTODO\r\nSUMMARY:Task 1 remote\r\nSTATUS:NEEDS-ACTION\r\nEND:VTODO\r\n');
`)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := database.Conn.ExecContext(context.Background(), `UPDATE tasks SET server_version=3 WHERE id='task-1'`); err != nil {
		t.Fatal(err)
	}

	err = database.MarkConflictResolved(context.Background(), ResolveConflictInput{
		ConflictID:      "open-1",
		Resolution:      "remote",
		ResolvedVTODO:   "BEGIN:VTODO\r\nSUMMARY:Resolved\r\nSTATUS:NEEDS-ACTION\r\nEND:VTODO\r\n",
		NewETag:         "\"etag-new\"",
		ExpectedVersion: 2,
	})
	if err == nil {
		t.Fatal("expected optimistic locking error")
	}

	var unresolvedCount int
	if err := database.Conn.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM conflicts WHERE id='open-1' AND resolved_at IS NULL`).Scan(&unresolvedCount); err != nil {
		t.Fatal(err)
	}
	if unresolvedCount != 1 {
		t.Fatalf("expected unresolved conflict, got %d", unresolvedCount)
	}
}
