package db

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
)

func TestLoadProjectRenameBaseReturnsProjectAndValidatesVersion(t *testing.T) {
	t.Parallel()

	database, err := OpenSQLite(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	if _, err := database.Conn.ExecContext(context.Background(), `
INSERT INTO projects (id, calendar_href, display_name, sync_strategy, server_version, created_at, updated_at)
VALUES ('project-1', '/cal/work/', 'Work', 'fullscan', 3, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
`); err != nil {
		t.Fatalf("seed project: %v", err)
	}

	base, err := database.LoadProjectRenameBase(context.Background(), "project-1", 3, "Renamed")
	if err != nil {
		t.Fatalf("load project rename base: %v", err)
	}

	if base.CalendarHref != "/cal/work/" || base.CurrentName != "Work" || base.CurrentVersion != 3 || base.ReservedVersion != 4 {
		t.Fatalf("unexpected base payload: %#v", base)
	}

	var reservedVersion int
	if err := database.Conn.QueryRowContext(context.Background(), `SELECT server_version FROM projects WHERE id = 'project-1';`).Scan(&reservedVersion); err != nil {
		t.Fatalf("load reserved version: %v", err)
	}
	if reservedVersion != 4 {
		t.Fatalf("unexpected reserved version: got %d want %d", reservedVersion, 4)
	}

	_, err = database.LoadProjectRenameBase(context.Background(), "project-1", 3, "Renamed Again")
	if !errors.Is(err, ErrProjectVersionMismatch) {
		t.Fatalf("expected version mismatch, got %v", err)
	}
}

func TestRenameProjectUpdatesProjectAndTaskDenormalizedName(t *testing.T) {
	t.Parallel()

	database, err := OpenSQLite(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	if _, err := database.Conn.ExecContext(context.Background(), `
INSERT INTO projects (id, calendar_href, display_name, sync_strategy, server_version, created_at, updated_at)
VALUES ('project-1', '/cal/work/', 'Work', 'fullscan', 4, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
`); err != nil {
		t.Fatalf("seed project: %v", err)
	}
	if _, err := database.Conn.ExecContext(context.Background(), `
INSERT INTO tasks (
    id, project_id, uid, href, server_version, title, status, raw_vtodo, project_name, sync_status, created_at, updated_at
) VALUES (
    'task-1', 'project-1', 'uid-1', '/cal/work/uid-1.ics', 1, 'Task', 'needs-action', 'BEGIN:VTODO\nUID:uid-1\nEND:VTODO', 'Work', 'synced', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
);
`); err != nil {
		t.Fatalf("seed task: %v", err)
	}

	base, err := database.LoadProjectRenameBase(context.Background(), "project-1", 4, "Renamed Work")
	if err != nil {
		t.Fatalf("load project rename base: %v", err)
	}

	if err := database.RenameProject(context.Background(), "project-1", base.ReservedVersion, "Renamed Work"); err != nil {
		t.Fatalf("rename project: %v", err)
	}

	var displayName string
	var version int
	if err := database.Conn.QueryRowContext(context.Background(), `SELECT display_name, server_version FROM projects WHERE id = 'project-1';`).Scan(&displayName, &version); err != nil {
		t.Fatalf("load project: %v", err)
	}
	if displayName != "Renamed Work" || version != 5 {
		t.Fatalf("unexpected project row: name=%q version=%d", displayName, version)
	}

	var projectName string
	if err := database.Conn.QueryRowContext(context.Background(), `SELECT project_name FROM tasks WHERE id = 'task-1';`).Scan(&projectName); err != nil {
		t.Fatalf("load task: %v", err)
	}
	if projectName != "Renamed Work" {
		t.Fatalf("unexpected task project_name: got %q want %q", projectName, "Renamed Work")
	}
}

func TestRenameProjectRejectsStaleVersionWithoutPartialUpdate(t *testing.T) {
	t.Parallel()

	database, err := OpenSQLite(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	if _, err := database.Conn.ExecContext(context.Background(), `
INSERT INTO projects (id, calendar_href, display_name, sync_strategy, server_version, created_at, updated_at)
VALUES ('project-1', '/cal/work/', 'Work', 'fullscan', 2, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
`); err != nil {
		t.Fatalf("seed project: %v", err)
	}
	if _, err := database.Conn.ExecContext(context.Background(), `
INSERT INTO tasks (
    id, project_id, uid, href, server_version, title, status, raw_vtodo, project_name, sync_status, created_at, updated_at
) VALUES (
    'task-1', 'project-1', 'uid-1', '/cal/work/uid-1.ics', 1, 'Task', 'needs-action', 'BEGIN:VTODO\nUID:uid-1\nEND:VTODO', 'Work', 'synced', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
);
`); err != nil {
		t.Fatalf("seed task: %v", err)
	}

	err = database.RenameProject(context.Background(), "project-1", 1, "Renamed Work")
	if !errors.Is(err, ErrProjectVersionMismatch) {
		t.Fatalf("expected version mismatch, got %v", err)
	}

	var displayName string
	if err := database.Conn.QueryRowContext(context.Background(), `SELECT display_name FROM projects WHERE id = 'project-1';`).Scan(&displayName); err != nil {
		t.Fatalf("load project: %v", err)
	}
	if displayName != "Work" {
		t.Fatalf("project should stay unchanged, got %q", displayName)
	}

	var projectName string
	if err := database.Conn.QueryRowContext(context.Background(), `SELECT project_name FROM tasks WHERE id = 'task-1';`).Scan(&projectName); err != nil {
		t.Fatalf("load task: %v", err)
	}
	if projectName != "Work" {
		t.Fatalf("task denormalized name should stay unchanged, got %q", projectName)
	}
}

func TestCancelProjectRenameReservationRestoresVersion(t *testing.T) {
	t.Parallel()

	database, err := OpenSQLite(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	if _, err := database.Conn.ExecContext(context.Background(), `
INSERT INTO projects (id, calendar_href, display_name, sync_strategy, server_version, created_at, updated_at)
VALUES ('project-1', '/cal/work/', 'Work', 'fullscan', 2, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
`); err != nil {
		t.Fatalf("seed project: %v", err)
	}

	base, err := database.LoadProjectRenameBase(context.Background(), "project-1", 2, "Renamed Work")
	if err != nil {
		t.Fatalf("load project rename base: %v", err)
	}

	if err := database.CancelProjectRenameReservation(context.Background(), "project-1", base.ReservedVersion); err != nil {
		t.Fatalf("cancel reservation: %v", err)
	}

	var version int
	if err := database.Conn.QueryRowContext(context.Background(), `SELECT server_version FROM projects WHERE id = 'project-1';`).Scan(&version); err != nil {
		t.Fatalf("load project version: %v", err)
	}
	if version != 2 {
		t.Fatalf("unexpected project version after cancel: got %d want %d", version, 2)
	}
}
