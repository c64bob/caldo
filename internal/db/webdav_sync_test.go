package db

import (
	"context"
	"path/filepath"
	"testing"
)

func TestApplyWebDAVSyncProjectAppliesIncrementalChanges(t *testing.T) {
	t.Parallel()

	database, err := OpenSQLite(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	if _, err := database.Conn.ExecContext(context.Background(), `
INSERT INTO projects (id, calendar_href, display_name, sync_strategy, sync_token, created_at, updated_at)
VALUES ('project-1', '/cal/work/', 'Work', 'webdav_sync', 'token-1', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);

INSERT INTO tasks (
	id, project_id, uid, href, etag, server_version, title, status, raw_vtodo, base_vtodo, project_name, sync_status, created_at, updated_at
) VALUES
	('task-unchanged', 'project-1', 'uid-unchanged', '/cal/work/uid-unchanged.ics', '"etag-keep"', 1, 'Keep', 'needs-action', 'BEGIN:VTODO\nUID:uid-unchanged\nSUMMARY:Keep\nEND:VTODO', 'BEGIN:VTODO\nUID:uid-unchanged\nSUMMARY:Keep\nEND:VTODO', 'Work', 'synced', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
	('task-clean', 'project-1', 'uid-clean', '/cal/work/uid-clean.ics', '"etag-old"', 2, 'Old', 'needs-action', 'BEGIN:VTODO\nUID:uid-clean\nSUMMARY:Old\nEND:VTODO', 'BEGIN:VTODO\nUID:uid-clean\nSUMMARY:Old\nEND:VTODO', 'Work', 'synced', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
	('task-delete-clean', 'project-1', 'uid-delete-clean', '/cal/work/uid-delete-clean.ics', '"etag-del"', 3, 'Delete Clean', 'needs-action', 'BEGIN:VTODO\nUID:uid-delete-clean\nSUMMARY:Delete Clean\nEND:VTODO', 'BEGIN:VTODO\nUID:uid-delete-clean\nSUMMARY:Delete Clean\nEND:VTODO', 'Work', 'synced', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
	('task-delete-dirty', 'project-1', 'uid-delete-dirty', '/cal/work/uid-delete-dirty.ics', '"etag-dirty"', 4, 'Delete Dirty Local', 'needs-action', 'BEGIN:VTODO\nUID:uid-delete-dirty\nSUMMARY:Delete Dirty Local\nEND:VTODO', 'BEGIN:VTODO\nUID:uid-delete-dirty\nSUMMARY:Delete Dirty Base\nEND:VTODO', 'Work', 'pending', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
`); err != nil {
		t.Fatalf("seed data: %v", err)
	}

	result, err := database.ApplyWebDAVSyncProject(context.Background(), "project-1", []ImportedTask{
		importedTask("uid-clean", "/cal/work/uid-clean.ics", `"etag-new"`, "Remote"),
		importedTask("uid-new", "/cal/work/uid-new.ics", `"etag-new-task"`, "New Remote"),
	}, []string{
		"/cal/work/uid-delete-clean.ics",
		"/cal/work/uid-delete-dirty.ics",
	}, "token-2", false)
	if err != nil {
		t.Fatalf("apply webdav sync: %v", err)
	}

	if result.Inserted != 1 || result.Updated != 1 || result.Deleted != 1 || result.Conflicts != 1 {
		t.Fatalf("unexpected result: %#v", result)
	}
	assertSingleTextResult(t, database, `SELECT sync_token FROM projects WHERE id='project-1';`, "token-2")
	assertSingleTextResult(t, database, `SELECT sync_strategy FROM projects WHERE id='project-1';`, "webdav_sync")
	assertSingleTextResult(t, database, `SELECT title FROM tasks WHERE uid='uid-unchanged';`, "Keep")
	assertSingleTextResult(t, database, `SELECT title FROM tasks WHERE uid='uid-clean';`, "Remote")
	assertSingleIntResult(t, database, `SELECT COUNT(*) FROM tasks WHERE uid='uid-new' AND sync_status='synced';`, 1)
	assertSingleIntResult(t, database, `SELECT COUNT(*) FROM tasks WHERE uid='uid-delete-clean';`, 0)
	assertSingleTextResult(t, database, `SELECT sync_status FROM tasks WHERE uid='uid-delete-dirty';`, "conflict")
	assertSingleIntResult(t, database, `SELECT COUNT(*) FROM conflicts WHERE task_id='task-delete-dirty' AND conflict_type='edit_delete' AND remote_vtodo IS NULL;`, 1)
}

func TestApplyWebDAVSyncProjectMatchesAbsoluteDeletedHref(t *testing.T) {
	t.Parallel()

	database, err := OpenSQLite(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	if _, err := database.Conn.ExecContext(context.Background(), `
INSERT INTO projects (id, calendar_href, display_name, sync_strategy, sync_token, created_at, updated_at)
VALUES ('project-1', '/cal/work/', 'Work', 'webdav_sync', 'token-1', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);

INSERT INTO tasks (
	id, project_id, uid, href, etag, server_version, title, status, raw_vtodo, base_vtodo, project_name, sync_status, created_at, updated_at
) VALUES
	('task-delete-clean', 'project-1', 'uid-delete-clean', '/cal/work/uid-delete-clean.ics', '"etag-del"', 3, 'Delete Clean', 'needs-action', 'BEGIN:VTODO\nUID:uid-delete-clean\nSUMMARY:Delete Clean\nEND:VTODO', 'BEGIN:VTODO\nUID:uid-delete-clean\nSUMMARY:Delete Clean\nEND:VTODO', 'Work', 'synced', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
`); err != nil {
		t.Fatalf("seed data: %v", err)
	}

	result, err := database.ApplyWebDAVSyncProject(context.Background(), "project-1", nil, []string{
		"https://nextcloud.example.test/cal/work/uid-delete-clean.ics",
	}, "token-2", false)
	if err != nil {
		t.Fatalf("apply webdav sync: %v", err)
	}

	if result.Deleted != 1 {
		t.Fatalf("unexpected result: %#v", result)
	}
	assertSingleIntResult(t, database, `SELECT COUNT(*) FROM tasks WHERE uid='uid-delete-clean';`, 0)
}

func TestApplyWebDAVSyncProjectRecordsConflictWhenHrefUIDChanges(t *testing.T) {
	t.Parallel()

	database, err := OpenSQLite(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	if _, err := database.Conn.ExecContext(context.Background(), `
INSERT INTO projects (id, calendar_href, display_name, sync_strategy, sync_token, created_at, updated_at)
VALUES ('project-1', '/cal/work/', 'Work', 'webdav_sync', 'token-1', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);

INSERT INTO tasks (
	id, project_id, uid, href, etag, server_version, title, status, raw_vtodo, base_vtodo, project_name, sync_status, created_at, updated_at
) VALUES
	('task-local', 'project-1', 'uid-local', '/cal/work/shared.ics', '"etag-old"', 2, 'Local', 'needs-action', 'BEGIN:VTODO\nUID:uid-local\nSUMMARY:Local\nEND:VTODO', 'BEGIN:VTODO\nUID:uid-local\nSUMMARY:Local\nEND:VTODO', 'Work', 'synced', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
`); err != nil {
		t.Fatalf("seed data: %v", err)
	}

	result, err := database.ApplyWebDAVSyncProject(context.Background(), "project-1", []ImportedTask{
		importedTask("uid-remote", "/cal/work/shared.ics", `"etag-new"`, "Remote UID"),
	}, nil, "token-2", false)
	if err != nil {
		t.Fatalf("apply webdav sync: %v", err)
	}
	if result.Conflicts != 1 || result.Inserted != 0 {
		t.Fatalf("unexpected result: %#v", result)
	}
	assertSingleTextResult(t, database, `SELECT sync_status FROM tasks WHERE id='task-local';`, "conflict")
	assertSingleIntResult(t, database, `SELECT COUNT(*) FROM tasks WHERE uid='uid-remote';`, 0)
	assertSingleIntResult(t, database, `SELECT COUNT(*) FROM conflicts WHERE task_id='task-local' AND conflict_type='field_conflict' AND remote_vtodo LIKE '%uid-remote%';`, 1)
}

func TestApplyWebDAVSyncProjectBaselineRemovesMissingCleanTasks(t *testing.T) {
	t.Parallel()

	database, err := OpenSQLite(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	if _, err := database.Conn.ExecContext(context.Background(), `
INSERT INTO projects (id, calendar_href, display_name, sync_strategy, created_at, updated_at)
VALUES ('project-1', '/cal/work/', 'Work', 'webdav_sync', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);

INSERT INTO tasks (
	id, project_id, uid, href, etag, server_version, title, status, raw_vtodo, base_vtodo, project_name, sync_status, created_at, updated_at
) VALUES
	('task-keep', 'project-1', 'uid-keep', '/cal/work/uid-keep.ics', '"etag-keep"', 1, 'Keep Old', 'needs-action', 'BEGIN:VTODO\nUID:uid-keep\nSUMMARY:Keep Old\nEND:VTODO', 'BEGIN:VTODO\nUID:uid-keep\nSUMMARY:Keep Old\nEND:VTODO', 'Work', 'synced', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
	('task-missing', 'project-1', 'uid-missing', '/cal/work/uid-missing.ics', '"etag-missing"', 1, 'Missing', 'needs-action', 'BEGIN:VTODO\nUID:uid-missing\nSUMMARY:Missing\nEND:VTODO', 'BEGIN:VTODO\nUID:uid-missing\nSUMMARY:Missing\nEND:VTODO', 'Work', 'synced', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
`); err != nil {
		t.Fatalf("seed data: %v", err)
	}

	result, err := database.ApplyWebDAVSyncProject(context.Background(), "project-1", []ImportedTask{
		importedTask("uid-keep", "/cal/work/uid-keep.ics", `"etag-new"`, "Keep Remote"),
	}, nil, "token-1", true)
	if err != nil {
		t.Fatalf("apply webdav sync: %v", err)
	}
	if result.Updated != 1 || result.Deleted != 1 {
		t.Fatalf("unexpected result: %#v", result)
	}
	assertSingleTextResult(t, database, `SELECT title FROM tasks WHERE uid='uid-keep';`, "Keep Remote")
	assertSingleIntResult(t, database, `SELECT COUNT(*) FROM tasks WHERE uid='uid-missing';`, 0)
}
