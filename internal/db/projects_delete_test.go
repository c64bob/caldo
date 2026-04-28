package db

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
)

func TestLoadProjectDeleteBaseReturnsMetadataAndTaskCount(t *testing.T) {
	t.Parallel()

	database, err := OpenSQLite(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	seedProjectWithTasks(t, database, 3)

	base, err := database.LoadProjectDeleteBase(context.Background(), "project-1", 3, "Work")
	if err != nil {
		t.Fatalf("load project delete base: %v", err)
	}
	if base.CalendarHref != "/cal/work/" || base.CurrentName != "Work" || base.AffectedTaskCount != 2 {
		t.Fatalf("unexpected delete base: %#v", base)
	}
	if base.ReservedVersion != 4 {
		t.Fatalf("unexpected reserved version: got %d want %d", base.ReservedVersion, 4)
	}
}

func TestLoadProjectDeleteBaseRejectsWrongConfirmation(t *testing.T) {
	t.Parallel()

	database, err := OpenSQLite(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	seedProjectWithTasks(t, database, 2)

	_, err = database.LoadProjectDeleteBase(context.Background(), "project-1", 2, "wrong")
	if !errors.Is(err, ErrProjectDeleteConfirmationMismatch) {
		t.Fatalf("expected ErrProjectDeleteConfirmationMismatch, got %v", err)
	}
}

func TestDeleteProjectRemovesProjectTasksAndDefaultBinding(t *testing.T) {
	t.Parallel()

	database, err := OpenSQLite(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	seedProjectWithTasks(t, database, 4)
	if _, err := database.Conn.ExecContext(context.Background(), `
UPDATE settings
SET default_project_id = 'project-1'
WHERE id = 'default';
`); err != nil {
		t.Fatalf("set default project: %v", err)
	}

	base, err := database.LoadProjectDeleteBase(context.Background(), "project-1", 4, "Work")
	if err != nil {
		t.Fatalf("load project delete base: %v", err)
	}
	if err := database.DeleteProject(context.Background(), "project-1", base.ReservedVersion); err != nil {
		t.Fatalf("delete project: %v", err)
	}

	var projectCount int
	if err := database.Conn.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM projects WHERE id = 'project-1';`).Scan(&projectCount); err != nil {
		t.Fatalf("count project: %v", err)
	}
	if projectCount != 0 {
		t.Fatalf("expected project deleted, got %d rows", projectCount)
	}

	var taskCount int
	if err := database.Conn.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM tasks WHERE project_id = 'project-1';`).Scan(&taskCount); err != nil {
		t.Fatalf("count tasks: %v", err)
	}
	if taskCount != 0 {
		t.Fatalf("expected tasks deleted, got %d rows", taskCount)
	}

	var defaultProjectID sql.NullString
	if err := database.Conn.QueryRowContext(context.Background(), `SELECT default_project_id FROM settings WHERE id = 'default';`).Scan(&defaultProjectID); err != nil {
		t.Fatalf("load default project id: %v", err)
	}
	if defaultProjectID.Valid {
		t.Fatalf("expected default_project_id to be null, got %q", defaultProjectID.String)
	}
}

func TestCancelProjectDeleteReservationRestoresVersion(t *testing.T) {
	t.Parallel()

	database, err := OpenSQLite(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	seedProjectWithTasks(t, database, 2)

	base, err := database.LoadProjectDeleteBase(context.Background(), "project-1", 2, "Work")
	if err != nil {
		t.Fatalf("load project delete base: %v", err)
	}
	if err := database.CancelProjectDeleteReservation(context.Background(), "project-1", base.ReservedVersion); err != nil {
		t.Fatalf("cancel delete reservation: %v", err)
	}

	var version int
	if err := database.Conn.QueryRowContext(context.Background(), `SELECT server_version FROM projects WHERE id = 'project-1';`).Scan(&version); err != nil {
		t.Fatalf("load project version: %v", err)
	}
	if version != 2 {
		t.Fatalf("unexpected restored version: got %d want %d", version, 2)
	}
}

func seedProjectWithTasks(t *testing.T, database *Database, version int) {
	t.Helper()

	if _, err := database.Conn.ExecContext(context.Background(), `
INSERT INTO projects (id, calendar_href, display_name, sync_strategy, server_version, created_at, updated_at)
VALUES ('project-1', '/cal/work/', 'Work', 'fullscan', ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
`, version); err != nil {
		t.Fatalf("seed project: %v", err)
	}
	if _, err := database.Conn.ExecContext(context.Background(), `
INSERT INTO tasks (
	id, project_id, uid, href, server_version, title, status, raw_vtodo, project_name, sync_status, created_at, updated_at
) VALUES
	('task-1', 'project-1', 'uid-1', '/cal/work/uid-1.ics', 1, 'Task 1', 'needs-action', 'BEGIN:VTODO\nUID:uid-1\nEND:VTODO', 'Work', 'synced', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
	('task-2', 'project-1', 'uid-2', '/cal/work/uid-2.ics', 1, 'Task 2', 'needs-action', 'BEGIN:VTODO\nUID:uid-2\nEND:VTODO', 'Work', 'synced', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
`); err != nil {
		t.Fatalf("seed tasks: %v", err)
	}
}
