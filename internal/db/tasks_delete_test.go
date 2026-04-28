package db

import (
	"context"
	"testing"
)

func TestPrepareTaskDeleteCreatesUndoSnapshotAndMarksPending(t *testing.T) {
	t.Parallel()
	database := openTaskUpdateTestDB(t)
	seedTaskUpdateTestData(t, database)

	prepared, err := database.PrepareTaskDelete(context.Background(), TaskDeleteInput{
		TaskID:          "task-1",
		ExpectedVersion: 2,
		SessionID:       "session-1",
		TabID:           "tab-1",
	})
	if err != nil {
		t.Fatalf("prepare task delete: %v", err)
	}
	if prepared.PendingVersion != 3 {
		t.Fatalf("unexpected pending version: %d", prepared.PendingVersion)
	}

	var syncStatus string
	var version int
	if err := database.Conn.QueryRowContext(context.Background(), `SELECT sync_status, server_version FROM tasks WHERE id = 'task-1';`).Scan(&syncStatus, &version); err != nil {
		t.Fatalf("query task: %v", err)
	}
	if syncStatus != "pending" || version != 3 {
		t.Fatalf("unexpected task row: status=%q version=%d", syncStatus, version)
	}

	var actionType string
	if err := database.Conn.QueryRowContext(context.Background(), `SELECT action_type FROM undo_snapshots WHERE session_id = 'session-1' AND tab_id = 'tab-1';`).Scan(&actionType); err != nil {
		t.Fatalf("query undo snapshot: %v", err)
	}
	if actionType != "task_deleted" {
		t.Fatalf("unexpected undo action type: %q", actionType)
	}

	var snapshotProjectID string
	if err := database.Conn.QueryRowContext(context.Background(), `SELECT json_extract(snapshot_fields, '$.project_id') FROM undo_snapshots WHERE session_id = 'session-1' AND tab_id = 'tab-1';`).Scan(&snapshotProjectID); err != nil {
		t.Fatalf("query undo snapshot project id: %v", err)
	}
	if snapshotProjectID != "project-1" {
		t.Fatalf("unexpected undo snapshot project id: %q", snapshotProjectID)
	}
}

func TestPrepareTaskDeleteRejectsVersionMismatch(t *testing.T) {
	t.Parallel()
	database := openTaskUpdateTestDB(t)
	seedTaskUpdateTestData(t, database)

	_, err := database.PrepareTaskDelete(context.Background(), TaskDeleteInput{
		TaskID:          "task-1",
		ExpectedVersion: 9,
		SessionID:       "session-1",
		TabID:           "tab-1",
	})
	if err != ErrTaskVersionMismatch {
		t.Fatalf("expected version mismatch, got %v", err)
	}
}
