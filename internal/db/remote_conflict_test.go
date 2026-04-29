package db

import (
	"context"
	"path/filepath"
	"testing"
)

func TestRecordRemoteFieldConflictMarksTaskAndStoresConflict(t *testing.T) {
	database, err := OpenSQLite(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	if _, err := database.Conn.ExecContext(context.Background(), `
INSERT INTO projects (id, calendar_href, display_name, created_at, updated_at) VALUES ('project-1','/cal/work/','Work',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP);
INSERT INTO tasks (id, project_id, uid, href, etag, server_version, title, status, raw_vtodo, base_vtodo, project_name, sync_status, created_at, updated_at)
VALUES ('task-1','project-1','uid-1','/cal/work/task-1.ics','"e1"',5,'Task','needs-action','BEGIN:VTODO\nSUMMARY:a\nEND:VTODO','BEGIN:VTODO\nSUMMARY:a\nEND:VTODO','Work','pending',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP);
`); err != nil {
		t.Fatal(err)
	}

	err = database.RecordRemoteFieldConflict(context.Background(), "task-1", 5, "base", "local", "remote")
	if err != nil {
		t.Fatal(err)
	}

	assertSingleTextResult(t, database, `SELECT sync_status FROM tasks WHERE id='task-1';`, "conflict")
	assertSingleIntResult(t, database, `SELECT COUNT(*) FROM conflicts WHERE task_id='task-1' AND conflict_type='field_conflict' AND base_vtodo='base' AND local_vtodo='local' AND remote_vtodo='remote';`, 1)
}

func TestRecordRemoteEditDeleteConflictStoresNullRemoteVTODO(t *testing.T) {
	database, err := OpenSQLite(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	if _, err := database.Conn.ExecContext(context.Background(), `
INSERT INTO projects (id, calendar_href, display_name, created_at, updated_at) VALUES ('project-1','/cal/work/','Work',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP);
INSERT INTO tasks (id, project_id, uid, href, etag, server_version, title, status, raw_vtodo, base_vtodo, project_name, sync_status, created_at, updated_at)
VALUES ('task-1','project-1','uid-1','/cal/work/task-1.ics','"e1"',5,'Task','needs-action','BEGIN:VTODO\nSUMMARY:a\nEND:VTODO','BEGIN:VTODO\nSUMMARY:a\nEND:VTODO','Work','pending',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP);
`); err != nil {
		t.Fatal(err)
	}

	err = database.RecordRemoteEditDeleteConflict(context.Background(), "task-1", 5, "base", "local")
	if err != nil {
		t.Fatal(err)
	}

	assertSingleIntResult(t, database, `SELECT COUNT(*) FROM conflicts WHERE task_id='task-1' AND conflict_type='edit_delete' AND base_vtodo='base' AND local_vtodo='local' AND remote_vtodo IS NULL;`, 1)
}

func TestRecordRemoteDeleteEditConflictStoresNullLocalVTODO(t *testing.T) {
	database, err := OpenSQLite(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	if _, err := database.Conn.ExecContext(context.Background(), `
INSERT INTO projects (id, calendar_href, display_name, created_at, updated_at) VALUES ('project-1','/cal/work/','Work',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP);
INSERT INTO tasks (id, project_id, uid, href, etag, server_version, title, status, raw_vtodo, base_vtodo, project_name, sync_status, created_at, updated_at)
VALUES ('task-1','project-1','uid-1','/cal/work/task-1.ics','"e1"',5,'Task','needs-action','BEGIN:VTODO\nSUMMARY:a\nEND:VTODO','BEGIN:VTODO\nSUMMARY:a\nEND:VTODO','Work','pending',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP);
`); err != nil {
		t.Fatal(err)
	}

	err = database.RecordRemoteDeleteEditConflict(context.Background(), "task-1", 5, "base", "remote")
	if err != nil {
		t.Fatal(err)
	}

	assertSingleIntResult(t, database, `SELECT COUNT(*) FROM conflicts WHERE task_id='task-1' AND conflict_type='delete_edit' AND base_vtodo='base' AND local_vtodo IS NULL AND remote_vtodo='remote';`, 1)
}
