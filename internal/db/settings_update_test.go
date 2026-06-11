package db

import (
	"context"
	"path/filepath"
	"testing"
)

func TestSaveSettingsCalendarsAddsMappingsAndUpdatesDefaultWithoutDeletingTasks(t *testing.T) {
	t.Parallel()

	database, err := OpenSQLite(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	if _, err := database.Conn.ExecContext(context.Background(), `
INSERT INTO projects (id, calendar_href, display_name, sync_strategy, is_default, created_at, updated_at)
VALUES ('project-work', '/cal/work/', 'Work', 'fullscan', TRUE, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
INSERT INTO projects (id, calendar_href, display_name, sync_strategy, is_default, created_at, updated_at)
VALUES ('project-empty', '/cal/empty/', 'Empty', 'fullscan', FALSE, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
UPDATE settings SET default_project_id = 'project-work' WHERE id = 'default';
INSERT INTO tasks (id, project_id, uid, href, title, status, raw_vtodo, sync_status, project_name, created_at, updated_at)
VALUES ('task-1', 'project-work', 'uid-1', '/cal/work/task-1.ics', 'Task', 'needs-action', 'BEGIN:VTODO\nUID:uid-1\nEND:VTODO', 'synced', 'Work', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
`); err != nil {
		t.Fatalf("seed settings calendar state: %v", err)
	}

	if err := database.SaveSettingsCalendars(context.Background(), []SelectedCalendar{
		{Href: "/cal/work/", DisplayName: "Work Remote"},
		{Href: "/cal/home/", DisplayName: "Home"},
	}, "/cal/home/", "ctag"); err != nil {
		t.Fatalf("save settings calendars: %v", err)
	}

	assertSingleIntResult(t, database, `SELECT COUNT(*) FROM projects;`, 2)
	assertSingleIntResult(t, database, `SELECT COUNT(*) FROM projects WHERE id = 'project-empty';`, 0)
	assertSingleIntResult(t, database, `SELECT COUNT(*) FROM tasks WHERE id = 'task-1' AND project_id = 'project-work';`, 1)
	assertSingleTextResult(t, database, `SELECT display_name FROM projects WHERE calendar_href = '/cal/work/';`, "Work Remote")
	assertSingleTextResult(t, database, `SELECT project_name FROM tasks WHERE id = 'task-1';`, "Work Remote")
	assertSingleTextResult(t, database, `SELECT sync_strategy FROM projects WHERE calendar_href = '/cal/home/';`, "ctag")
	assertSingleIntResult(t, database, `SELECT COUNT(*) FROM projects WHERE is_default = TRUE AND calendar_href = '/cal/home/';`, 1)
	assertSingleTextResult(t, database, `SELECT p.calendar_href FROM settings s JOIN projects p ON p.id = s.default_project_id WHERE s.id = 'default';`, "/cal/home/")
}
