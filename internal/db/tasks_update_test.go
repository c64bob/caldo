package db

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
)

func TestPrepareTaskUpdateCreatesUndoSnapshotAndMarksPending(t *testing.T) {
	t.Parallel()
	database := openTaskUpdateTestDB(t)
	seedTaskUpdateTestData(t, database)

	prepared, err := database.PrepareTaskUpdate(context.Background(), TaskUpdateInput{
		TaskID:          "task-1",
		ExpectedVersion: 2,
		SessionID:       "session-1",
		TabID:           "tab-1",
		ProjectID:       "project-1",
		ProjectName:     "Inbox",
		Href:            "/cal/inbox/uid-1.ics",
		RawVTODO:        "BEGIN:VTODO\nUID:uid-1\nSUMMARY:new\nEND:VTODO",
		Title:           "new",
		Description:     "desc",
		Status:          "needs-action",
		DueDate:         sql.NullString{String: "2026-06-01", Valid: true},
		Priority:        sql.NullInt64{Int64: 3, Valid: true},
		LabelNames:      sql.NullString{String: "home,STARRED,urgent", Valid: true},
	})
	if err != nil {
		t.Fatalf("prepare task update: %v", err)
	}
	if prepared.PreviousHref != "/cal/inbox/uid-1.ics" {
		t.Fatalf("unexpected previous href: %q", prepared.PreviousHref)
	}

	var syncStatus, title, description, labelNames string
	var version int
	if err := database.Conn.QueryRowContext(context.Background(), `SELECT sync_status, title, description, label_names, server_version FROM tasks WHERE id = 'task-1';`).Scan(&syncStatus, &title, &description, &labelNames, &version); err != nil {
		t.Fatalf("query task: %v", err)
	}
	if syncStatus != "pending" || title != "new" || description != "desc" || labelNames != "home,STARRED,urgent" {
		t.Fatalf("unexpected task row: status=%q title=%q description=%q labels=%q", syncStatus, title, description, labelNames)
	}
	if prepared.PendingVersion != 3 || version != 3 {
		t.Fatalf("unexpected pending version: prepared=%d version=%d", prepared.PendingVersion, version)
	}

	var snapshotCount int
	if err := database.Conn.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM undo_snapshots WHERE session_id = 'session-1' AND tab_id = 'tab-1' AND task_id = 'task-1';`).Scan(&snapshotCount); err != nil {
		t.Fatalf("query undo snapshot: %v", err)
	}
	if snapshotCount != 1 {
		t.Fatalf("expected one snapshot, got %d", snapshotCount)
	}

	var snapshotTitle, snapshotStatus, snapshotLabelNames, snapshotEtag string
	if err := database.Conn.QueryRowContext(context.Background(), `SELECT json_extract(snapshot_fields, '$.title'), json_extract(snapshot_fields, '$.status'), json_extract(snapshot_fields, '$.label_names'), etag_at_snapshot FROM undo_snapshots WHERE session_id = 'session-1' AND tab_id = 'tab-1';`).Scan(&snapshotTitle, &snapshotStatus, &snapshotLabelNames, &snapshotEtag); err != nil {
		t.Fatalf("query undo snapshot fields: %v", err)
	}
	if snapshotTitle != "old" || snapshotStatus != "needs-action" || snapshotLabelNames != "home" || snapshotEtag != "\"etag-1\"" {
		t.Fatalf("unexpected snapshot data: title=%q status=%q labels=%q etag=%q", snapshotTitle, snapshotStatus, snapshotLabelNames, snapshotEtag)
	}

	assertSingleIntResult(t, database, `SELECT COUNT(*) FROM labels WHERE LOWER(name) = 'urgent';`, 1)
	assertSingleIntResult(t, database, `SELECT COUNT(*) FROM labels WHERE LOWER(name) = 'starred';`, 0)
	assertSingleIntResult(t, database, `SELECT COUNT(*) FROM task_labels WHERE task_id = 'task-1';`, 2)
}

func TestPrepareTaskUpdateRejectsVersionMismatch(t *testing.T) {
	t.Parallel()
	database := openTaskUpdateTestDB(t)
	seedTaskUpdateTestData(t, database)

	_, err := database.PrepareTaskUpdate(context.Background(), TaskUpdateInput{
		TaskID:          "task-1",
		ExpectedVersion: 7,
		SessionID:       "session-1",
		TabID:           "tab-1",
		ProjectID:       "project-1",
		ProjectName:     "Inbox",
		Href:            "/cal/inbox/uid-1.ics",
		RawVTODO:        "BEGIN:VTODO\nUID:uid-1\nSUMMARY:new\nEND:VTODO",
		Title:           "new",
		Status:          "needs-action",
	})
	if err != ErrTaskVersionMismatch {
		t.Fatalf("expected version mismatch, got %v", err)
	}
}

func TestListOpenDirectSubtaskIDsReturnsOnlyOpenDirectChildren(t *testing.T) {
	t.Parallel()
	database := openTaskUpdateTestDB(t)
	seedTaskUpdateTestData(t, database)

	if _, err := database.Conn.ExecContext(context.Background(), `
INSERT INTO tasks (
    id, project_id, uid, href, etag, server_version, title, status, parent_id, raw_vtodo, base_vtodo,
    project_name, sync_status, created_at, updated_at
) VALUES
    ('child-open-a', 'project-1', 'uid-child-open-a', '/cal/inbox/uid-child-open-a.ics', '"etag-a"', 1, 'a', 'needs-action', 'task-1', 'BEGIN:VTODO\nUID:uid-child-open-a\nEND:VTODO', 'BEGIN:VTODO\nUID:uid-child-open-a\nEND:VTODO', 'Inbox', 'synced', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    ('child-completed', 'project-1', 'uid-child-completed', '/cal/inbox/uid-child-completed.ics', '"etag-b"', 1, 'b', 'completed', 'task-1', 'BEGIN:VTODO\nUID:uid-child-completed\nEND:VTODO', 'BEGIN:VTODO\nUID:uid-child-completed\nEND:VTODO', 'Inbox', 'synced', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    ('child-open-b', 'project-1', 'uid-child-open-b', '/cal/inbox/uid-child-open-b.ics', '"etag-c"', 1, 'c', 'needs-action', 'task-1', 'BEGIN:VTODO\nUID:uid-child-open-b\nEND:VTODO', 'BEGIN:VTODO\nUID:uid-child-open-b\nEND:VTODO', 'Inbox', 'synced', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    ('grandchild-open', 'project-1', 'uid-grandchild-open', '/cal/inbox/uid-grandchild-open.ics', '"etag-d"', 1, 'd', 'needs-action', 'child-open-a', 'BEGIN:VTODO\nUID:uid-grandchild-open\nEND:VTODO', 'BEGIN:VTODO\nUID:uid-grandchild-open\nEND:VTODO', 'Inbox', 'synced', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
`); err != nil {
		t.Fatalf("seed subtasks: %v", err)
	}

	subtaskIDs, err := database.ListOpenDirectSubtaskIDs(context.Background(), "task-1")
	if err != nil {
		t.Fatalf("list open direct subtasks: %v", err)
	}
	if len(subtaskIDs) != 2 || subtaskIDs[0] != "child-open-a" || subtaskIDs[1] != "child-open-b" {
		t.Fatalf("unexpected subtask ids: %#v", subtaskIDs)
	}
}

func openTaskUpdateTestDB(t *testing.T) *Database {
	t.Helper()
	database, err := OpenSQLite(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	return database
}

func seedTaskUpdateTestData(t *testing.T, database *Database) {
	t.Helper()
	if _, err := database.Conn.ExecContext(context.Background(), `
INSERT INTO projects (id, calendar_href, display_name, sync_strategy, is_default, created_at, updated_at)
VALUES ('project-1', '/cal/inbox/', 'Inbox', 'fullscan', TRUE, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
INSERT INTO tasks (
    id, project_id, uid, href, etag, server_version, title, description, status, raw_vtodo, base_vtodo,
    label_names, project_name, sync_status, created_at, updated_at
) VALUES (
    'task-1', 'project-1', 'uid-1', '/cal/inbox/uid-1.ics', '"etag-1"', 2, 'old', 'old-desc', 'needs-action',
    'BEGIN:VTODO\nUID:uid-1\nSUMMARY:old\nEND:VTODO', 'BEGIN:VTODO\nUID:uid-1\nSUMMARY:old\nEND:VTODO',
    'home', 'Inbox', 'synced', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
);
`); err != nil {
		t.Fatalf("seed update data: %v", err)
	}
}
