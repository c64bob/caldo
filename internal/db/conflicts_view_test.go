package db

import (
	"context"
	"testing"
)

func TestListUnresolvedConflictsExcludesResolved(t *testing.T) {
	database, err := OpenSQLite(t.TempDir() + "/caldo.db")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	_, err = database.Conn.ExecContext(context.Background(), `
INSERT INTO projects (id, calendar_href, display_name, sync_strategy, created_at, updated_at) VALUES ('project-1','/p','Inbox','ctag',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP);
INSERT INTO tasks (id, project_id, uid, href, etag, title, status, raw_vtodo, sync_status, created_at, updated_at)
VALUES ('task-1','project-1','uid-1','/t','e','Task 1','needs-action','raw','conflict',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP);
INSERT INTO conflicts (id, task_id, project_id, conflict_type, created_at, base_vtodo, local_vtodo, remote_vtodo)
VALUES ('open-1','task-1','project-1','field_conflict',CURRENT_TIMESTAMP,'b','l','r');
INSERT INTO conflicts (id, task_id, project_id, conflict_type, created_at, resolved_at, resolution)
VALUES ('resolved-1','task-1','project-1','field_conflict',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,'local');
`)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	rows, err := database.ListUnresolvedConflicts(context.Background(), 10)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(rows) != 1 || rows[0].ID != "open-1" {
		t.Fatalf("unexpected rows: %+v", rows)
	}
}
