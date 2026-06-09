package db

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
)

func TestResolveTaskProjectUsesExplicitProject(t *testing.T) {
	t.Parallel()
	database := openSQLiteForTaskCreateTest(t)
	seedTaskCreateProjects(t, database)

	project, err := database.ResolveTaskProject(context.Background(), "project-work")
	if err != nil {
		t.Fatalf("resolve task project: %v", err)
	}
	if project.ID != "project-work" {
		t.Fatalf("unexpected project id: %q", project.ID)
	}
}

func TestResolveTaskProjectUsesDefaultProject(t *testing.T) {
	t.Parallel()
	database := openSQLiteForTaskCreateTest(t)
	seedTaskCreateProjects(t, database)

	project, err := database.ResolveTaskProject(context.Background(), "")
	if err != nil {
		t.Fatalf("resolve task project: %v", err)
	}
	if project.ID != "project-default" {
		t.Fatalf("unexpected default project id: %q", project.ID)
	}
}

func TestResolveTaskProjectReturnsUnavailableWhenDefaultMissing(t *testing.T) {
	t.Parallel()
	database := openSQLiteForTaskCreateTest(t)
	seedTaskCreateProjects(t, database)

	if _, err := database.Conn.ExecContext(context.Background(), `UPDATE settings SET default_project_id = NULL WHERE id = 'default';`); err != nil {
		t.Fatalf("update default project id: %v", err)
	}

	_, err := database.ResolveTaskProject(context.Background(), "")
	if !errors.Is(err, ErrTaskProjectUnavailable) {
		t.Fatalf("expected unavailable error, got %v", err)
	}
}

func TestInsertPendingTaskAndStatusTransitions(t *testing.T) {
	t.Parallel()
	database := openSQLiteForTaskCreateTest(t)
	seedTaskCreateProjects(t, database)

	taskID, err := database.InsertPendingTask(context.Background(), NewTaskInput{
		ProjectID:   "project-default",
		ProjectName: "Inbox",
		UID:         "uid-1",
		Href:        "/cal/inbox/uid-1.ics",
		Title:       "Task title",
		Description: "Task description",
		DueDate:     sql.NullString{String: "2026-06-09", Valid: true},
		Priority:    sql.NullInt64{Int64: 1, Valid: true},
		RRule:       "FREQ=WEEKLY;BYDAY=MO",
		LabelNames:  sql.NullString{String: "home,work", Valid: true},
		RawVTODO:    "BEGIN:VCALENDAR\nEND:VCALENDAR",
	})
	if err != nil {
		t.Fatalf("insert pending task: %v", err)
	}

	version, err := database.MarkTaskCreateSynced(context.Background(), taskID, `"etag-1"`)
	if err != nil {
		t.Fatalf("mark synced: %v", err)
	}
	if version != 2 {
		t.Fatalf("expected returned version 2, got %d", version)
	}

	var syncStatus string
	var etag sql.NullString
	var serverVersion int
	var description sql.NullString
	var dueDate sql.NullString
	var priority sql.NullInt64
	var rrule sql.NullString
	var labelNames sql.NullString
	if err := database.Conn.QueryRowContext(context.Background(), `SELECT sync_status, etag, server_version, description, date(due_date), priority, rrule, label_names FROM tasks WHERE id = ?;`, taskID).Scan(&syncStatus, &etag, &serverVersion, &description, &dueDate, &priority, &rrule, &labelNames); err != nil {
		t.Fatalf("query task: %v", err)
	}
	if syncStatus != "synced" || !etag.Valid || etag.String != `"etag-1"` || serverVersion != 2 {
		t.Fatalf("unexpected synced state: status=%q etag=%q version=%d", syncStatus, etag.String, serverVersion)
	}
	if !description.Valid || description.String != "Task description" || !dueDate.Valid || dueDate.String != "2026-06-09" || !priority.Valid || priority.Int64 != 1 || !rrule.Valid || rrule.String != "FREQ=WEEKLY;BYDAY=MO" || !labelNames.Valid || labelNames.String != "home,work" {
		t.Fatalf("unexpected denormalized create fields: description=%#v due=%#v priority=%#v rrule=%#v labels=%#v", description, dueDate, priority, rrule, labelNames)
	}
	var labelCount int
	if err := database.Conn.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM task_labels WHERE task_id = ?;`, taskID).Scan(&labelCount); err != nil {
		t.Fatalf("query task labels: %v", err)
	}
	if labelCount != 2 {
		t.Fatalf("unexpected task label count: got %d want 2", labelCount)
	}

	if err := database.MarkTaskCreateError(context.Background(), taskID); err != nil {
		t.Fatalf("mark error: %v", err)
	}
	if err := database.Conn.QueryRowContext(context.Background(), `SELECT sync_status, server_version FROM tasks WHERE id = ?;`, taskID).Scan(&syncStatus, &serverVersion); err != nil {
		t.Fatalf("query task after error: %v", err)
	}
	if syncStatus != "error" || serverVersion != 2 {
		t.Fatalf("unexpected error state: status=%q version=%d", syncStatus, serverVersion)
	}
}

func seedTaskCreateProjects(t *testing.T, database *Database) {
	t.Helper()
	if _, err := database.Conn.ExecContext(context.Background(), `
INSERT INTO projects (id, calendar_href, display_name, sync_strategy, is_default, created_at, updated_at)
VALUES
('project-default', '/cal/inbox/', 'Inbox', 'fullscan', TRUE, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('project-work', '/cal/work/', 'Work', 'fullscan', FALSE, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
UPDATE settings SET default_project_id = 'project-default' WHERE id = 'default';
`); err != nil {
		t.Fatalf("seed projects: %v", err)
	}
}

func openSQLiteForTaskCreateTest(t *testing.T) *Database {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "caldo.db")
	database, err := OpenSQLite(dbPath)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	return database
}
