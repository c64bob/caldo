package db

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"
)

func TestApplyFullScanProjectAppliesRemoteChangesAndConflicts(t *testing.T) {
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
	('task-clean', 'project-1', 'uid-clean', '/cal/work/uid-clean.ics', '"etag-old"', 2, 'Old', 'needs-action', 'BEGIN:VTODO\nUID:uid-clean\nSUMMARY:Old\nEND:VTODO', 'BEGIN:VTODO\nUID:uid-clean\nSUMMARY:Old\nEND:VTODO', 'Work', 'synced', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
	('task-deleted', 'project-1', 'uid-deleted', '/cal/work/uid-deleted.ics', '"etag-del"', 3, 'Deleted', 'needs-action', 'BEGIN:VTODO\nUID:uid-deleted\nSUMMARY:Deleted\nEND:VTODO', 'BEGIN:VTODO\nUID:uid-deleted\nSUMMARY:Deleted\nEND:VTODO', 'Work', 'synced', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
	('task-dirty', 'project-1', 'uid-dirty', '/cal/work/uid-dirty.ics', '"etag-base"', 4, 'Local', 'needs-action', 'BEGIN:VTODO\nUID:uid-dirty\nSUMMARY:Local\nEND:VTODO', 'BEGIN:VTODO\nUID:uid-dirty\nSUMMARY:Base\nEND:VTODO', 'Work', 'pending', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
`); err != nil {
		t.Fatalf("seed data: %v", err)
	}
	dirtyBase := "BEGIN:VTODO\nUID:uid-dirty\nSUMMARY:Base\nEND:VTODO"
	dirtyLocal := "BEGIN:VTODO\nUID:uid-dirty\nSUMMARY:Local\nEND:VTODO"
	if _, err := database.Conn.ExecContext(context.Background(), `
UPDATE tasks
SET raw_vtodo = ?, base_vtodo = ?
WHERE id = 'task-dirty';
`, dirtyLocal, dirtyBase); err != nil {
		t.Fatalf("fix dirty vtodo seed: %v", err)
	}

	result, err := database.ApplyFullScanProject(context.Background(), "project-1", []ImportedTask{
		importedTask("uid-clean", "/cal/work/uid-clean.ics", `"etag-new"`, "Remote"),
		importedTask("uid-new", "/cal/work/uid-new.ics", `"etag-new-task"`, "New Remote"),
		importedTask("uid-dirty", "/cal/work/uid-dirty.ics", `"etag-remote"`, "Remote Dirty"),
	})
	if err != nil {
		t.Fatalf("apply fullscan: %v", err)
	}
	if result.Inserted != 1 || result.Updated != 1 || result.Deleted != 1 || result.Conflicts != 1 {
		t.Fatalf("unexpected result: %#v", result)
	}

	assertSingleTextResult(t, database, `SELECT title FROM tasks WHERE uid='uid-clean';`, "Remote")
	assertSingleTextResult(t, database, `SELECT sync_status FROM tasks WHERE uid='uid-clean';`, "synced")
	assertSingleIntResult(t, database, `SELECT server_version FROM tasks WHERE uid='uid-clean';`, 3)
	assertSingleIntResult(t, database, `SELECT COUNT(*) FROM tasks WHERE uid='uid-new' AND sync_status='synced';`, 1)
	assertSingleIntResult(t, database, `SELECT COUNT(*) FROM tasks WHERE uid='uid-deleted';`, 0)
	assertSingleTextResult(t, database, `SELECT sync_status FROM tasks WHERE uid='uid-dirty';`, "conflict")
	assertSingleTextResult(t, database, `SELECT etag FROM tasks WHERE uid='uid-dirty';`, `"etag-remote"`)
	assertSingleIntResult(t, database, `SELECT COUNT(*) FROM conflicts WHERE task_id='task-dirty' AND conflict_type='field_conflict' AND remote_vtodo LIKE '%Remote Dirty%';`, 1)
}

func TestApplyFullScanProjectAutoMergesConflictFreeDirtyTask(t *testing.T) {
	t.Parallel()

	database, err := OpenSQLite(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	base := "BEGIN:VTODO\nUID:uid-merge\nSUMMARY:Base\nDESCRIPTION:Base description\nEND:VTODO"
	local := "BEGIN:VTODO\nUID:uid-merge\nSUMMARY:Local title\nDESCRIPTION:Base description\nEND:VTODO"
	remote := "BEGIN:VTODO\nUID:uid-merge\nSUMMARY:Base\nDESCRIPTION:Remote description\nEND:VTODO"
	if _, err := database.Conn.ExecContext(context.Background(), `
INSERT INTO projects (id, calendar_href, display_name, sync_strategy, created_at, updated_at)
VALUES ('project-1', '/cal/work/', 'Work', 'fullscan', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);

INSERT INTO tasks (
	id, project_id, uid, href, etag, server_version, title, description, status, raw_vtodo, base_vtodo, project_name, sync_status, created_at, updated_at
) VALUES
	('task-merge', 'project-1', 'uid-merge', '/cal/work/uid-merge.ics', '"etag-base"', 4, 'Local title', 'Base description', 'needs-action', ?, ?, 'Work', 'pending', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
`, local, base); err != nil {
		t.Fatalf("seed data: %v", err)
	}

	result, err := database.ApplyFullScanProject(context.Background(), "project-1", []ImportedTask{{
		UID:         "uid-merge",
		Href:        "/cal/work/uid-merge.ics",
		ETag:        `"etag-remote"`,
		Title:       "Base",
		Description: "Remote description",
		Status:      "needs-action",
		RawVTODO:    remote,
		BaseVTODO:   remote,
		ProjectName: "Work",
	}})
	if err != nil {
		t.Fatalf("apply fullscan: %v", err)
	}
	if result.Updated != 1 || result.Conflicts != 0 {
		t.Fatalf("unexpected result: %#v", result)
	}

	assertSingleTextResult(t, database, `SELECT title FROM tasks WHERE id='task-merge';`, "Local title")
	assertSingleTextResult(t, database, `SELECT description FROM tasks WHERE id='task-merge';`, "Remote description")
	assertSingleTextResult(t, database, `SELECT etag FROM tasks WHERE id='task-merge';`, `"etag-remote"`)
	assertSingleTextResult(t, database, `SELECT sync_status FROM tasks WHERE id='task-merge';`, "synced")
	assertSingleIntResult(t, database, `SELECT server_version FROM tasks WHERE id='task-merge';`, 5)
	assertSingleIntResult(t, database, `SELECT COUNT(*) FROM conflicts WHERE task_id='task-merge';`, 0)

	var raw, persistedBase string
	if err := database.Conn.QueryRowContext(context.Background(), `SELECT raw_vtodo, base_vtodo FROM tasks WHERE id='task-merge';`).Scan(&raw, &persistedBase); err != nil {
		t.Fatalf("query merged task: %v", err)
	}
	if raw != persistedBase {
		t.Fatalf("expected merged base to match raw")
	}
	if !strings.Contains(raw, "SUMMARY:Local title") || !strings.Contains(raw, "DESCRIPTION:Remote description") {
		t.Fatalf("unexpected merged payload: %q", raw)
	}
}

func TestApplyFullScanProjectMissingBaseDisablesAutoMerge(t *testing.T) {
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
	('task-nobase', 'project-1', 'uid-nobase', '/cal/work/uid-nobase.ics', '"etag-base"', 4, 'Local', 'needs-action', 'BEGIN:VTODO\nUID:uid-nobase\nSUMMARY:Local\nEND:VTODO', NULL, 'Work', 'pending', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
`); err != nil {
		t.Fatalf("seed data: %v", err)
	}

	result, err := database.ApplyFullScanProject(context.Background(), "project-1", []ImportedTask{
		importedTask("uid-nobase", "/cal/work/uid-nobase.ics", `"etag-remote"`, "Remote"),
	})
	if err != nil {
		t.Fatalf("apply fullscan: %v", err)
	}
	if result.Conflicts != 1 || result.Updated != 0 {
		t.Fatalf("unexpected result: %#v", result)
	}

	assertSingleTextResult(t, database, `SELECT sync_status FROM tasks WHERE id='task-nobase';`, "conflict")
	assertSingleTextResult(t, database, `SELECT etag FROM tasks WHERE id='task-nobase';`, `"etag-remote"`)
	assertSingleIntResult(t, database, `SELECT COUNT(*) FROM conflicts WHERE task_id='task-nobase' AND conflict_type='field_conflict' AND base_vtodo IS NULL;`, 1)
}

func TestApplyFullScanProjectRecordsEditDeleteConflictForDirtyMissingRemote(t *testing.T) {
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
	('task-dirty', 'project-1', 'uid-dirty', '/cal/work/uid-dirty.ics', '"etag-base"', 4, 'Local', 'needs-action', 'BEGIN:VTODO\nUID:uid-dirty\nSUMMARY:Local\nEND:VTODO', 'BEGIN:VTODO\nUID:uid-dirty\nSUMMARY:Base\nEND:VTODO', 'Work', 'pending', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
`); err != nil {
		t.Fatalf("seed data: %v", err)
	}

	result, err := database.ApplyFullScanProject(context.Background(), "project-1", nil)
	if err != nil {
		t.Fatalf("apply fullscan: %v", err)
	}
	if result.Conflicts != 1 || result.Deleted != 0 {
		t.Fatalf("unexpected result: %#v", result)
	}

	assertSingleTextResult(t, database, `SELECT sync_status FROM tasks WHERE uid='uid-dirty';`, "conflict")
	assertSingleIntResult(t, database, `SELECT COUNT(*) FROM conflicts WHERE task_id='task-dirty' AND conflict_type='edit_delete' AND remote_vtodo IS NULL;`, 1)
}

func TestApplyFullScanProjectUpdatesParentLinksForRemoteSubtasks(t *testing.T) {
	t.Parallel()

	database, err := OpenSQLite(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	if _, err := database.Conn.ExecContext(context.Background(), `
INSERT INTO projects (id, calendar_href, display_name, sync_strategy, created_at, updated_at)
VALUES ('project-1', '/cal/work/', 'Work', 'fullscan', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
`); err != nil {
		t.Fatalf("seed project: %v", err)
	}

	_, err = database.ApplyFullScanProject(context.Background(), "project-1", []ImportedTask{
		importedTask("uid-parent", "/cal/work/uid-parent.ics", `"etag-parent"`, "Parent"),
		{
			UID:         "uid-child",
			Href:        "/cal/work/uid-child.ics",
			ETag:        `"etag-child"`,
			Title:       "Child",
			Status:      "needs-action",
			ParentUID:   "uid-parent",
			RawVTODO:    "BEGIN:VTODO\nUID:uid-child\nSUMMARY:Child\nRELATED-TO;RELTYPE=PARENT:uid-parent\nEND:VTODO",
			BaseVTODO:   "BEGIN:VTODO\nUID:uid-child\nSUMMARY:Child\nRELATED-TO;RELTYPE=PARENT:uid-parent\nEND:VTODO",
			ProjectName: "Work",
		},
	})
	if err != nil {
		t.Fatalf("apply fullscan: %v", err)
	}

	var parentID sql.NullString
	if err := database.Conn.QueryRowContext(context.Background(), `SELECT parent_id FROM tasks WHERE uid='uid-child';`).Scan(&parentID); err != nil {
		t.Fatalf("query child parent: %v", err)
	}
	if !parentID.Valid {
		t.Fatal("expected child parent_id to be set")
	}
}

func importedTask(uid string, href string, etag string, title string) ImportedTask {
	raw := "BEGIN:VTODO\nUID:" + uid + "\nSUMMARY:" + title + "\nEND:VTODO"
	return ImportedTask{
		UID:         uid,
		Href:        href,
		ETag:        etag,
		Title:       title,
		Status:      "needs-action",
		RawVTODO:    raw,
		BaseVTODO:   raw,
		ProjectName: "Work",
	}
}
