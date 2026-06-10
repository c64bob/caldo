package db

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestListTodayTasksIncludesTodayAndOverdue(t *testing.T) {
	t.Parallel()

	database := openViewTestDB(t)
	seedViewTasks(t, database)

	results, err := database.ListTodayTasks(context.Background(), time.Date(2026, 4, 28, 12, 0, 0, 0, time.UTC), 50)
	if err != nil {
		t.Fatalf("list today tasks: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("unexpected result count: got %d want %d", len(results), 2)
	}
	if results[0].ID != "task-overdue-active" || results[1].ID != "task-today-active" {
		t.Fatalf("unexpected order or ids: %#v", results)
	}

	today := results[1]
	if today.Description != "Heute Beschreibung" ||
		today.ProjectName != "Work" ||
		today.DueISODate != "2026-04-28" ||
		today.Priority != 4 ||
		!today.HasPriority ||
		today.LabelNames != "Büro,urgent" ||
		today.SyncStatus != "pending" ||
		today.ServerVersion != 3 ||
		today.IsSubtask {
		t.Fatalf("today result missing row metadata: %#v", today)
	}
}

func TestListTodayTasksIncludesDueAtStoredAsDriverTimestamp(t *testing.T) {
	t.Parallel()

	database := openViewTestDB(t)
	seedViewTasks(t, database)

	if _, err := database.Conn.Exec(`
INSERT INTO tasks (
	id, project_id, uid, href, etag, server_version, title, description, status, raw_vtodo, base_vtodo,
	label_names, project_name, sync_status, due_date, due_at, created_at, updated_at
) VALUES (
	?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
);
`, "task-today-due-at", "project-1", "uid-today-due-at", "/calendars/work/task-today-due-at.ics", `"etag-8"`, 1,
		"Heute mit due_at", "", "needs-action", "BEGIN:VTODO\nUID:uid-today-due-at\nEND:VTODO", "BEGIN:VTODO\nUID:uid-today-due-at\nEND:VTODO",
		"", "Work", "synced", nil, time.Date(2026, 4, 28, 15, 30, 0, 0, time.UTC),
	); err != nil {
		t.Fatalf("insert due_at task: %v", err)
	}

	results, err := database.ListTodayTasks(context.Background(), time.Date(2026, 4, 28, 12, 0, 0, 0, time.UTC), 50)
	if err != nil {
		t.Fatalf("list today tasks: %v", err)
	}

	found := false
	for _, row := range results {
		if row.ID == "task-today-due-at" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("today results missing due_at task: %#v", results)
	}
}

func TestListTodayTasksIncludesSubtaskRelationshipMetadata(t *testing.T) {
	t.Parallel()

	database := openViewTestDB(t)
	seedViewTasks(t, database)

	if _, err := database.Conn.Exec(`
INSERT INTO tasks (
	id, project_id, uid, href, etag, server_version, title, description, status, parent_id, raw_vtodo, base_vtodo,
	label_names, project_name, sync_status, due_date, created_at, updated_at
) VALUES (
	'task-today-child', 'project-1', 'uid-today-child', '/calendars/work/task-today-child.ics', '"etag-child"', 1,
	'Heute Unteraufgabe', '', 'needs-action', 'task-today-active', 'BEGIN:VTODO\nUID:uid-today-child\nRELATED-TO;RELTYPE=PARENT:uid-today-active\nEND:VTODO',
	'BEGIN:VTODO\nUID:uid-today-child\nRELATED-TO;RELTYPE=PARENT:uid-today-active\nEND:VTODO', '', 'Work', 'synced', '2026-04-28', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
),
(
	'task-today-child-done', 'project-1', 'uid-today-child-done', '/calendars/work/task-today-child-done.ics', '"etag-child-done"', 1,
	'Erledigte Unteraufgabe', '', 'completed', 'task-today-active', 'BEGIN:VTODO\nUID:uid-today-child-done\nRELATED-TO;RELTYPE=PARENT:uid-today-active\nEND:VTODO',
	'BEGIN:VTODO\nUID:uid-today-child-done\nRELATED-TO;RELTYPE=PARENT:uid-today-active\nEND:VTODO', '', 'Work', 'synced', '2026-04-28', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
);
`); err != nil {
		t.Fatalf("insert subtask: %v", err)
	}

	results, err := database.ListTodayTasks(context.Background(), time.Date(2026, 4, 28, 12, 0, 0, 0, time.UTC), 50)
	if err != nil {
		t.Fatalf("list today tasks: %v", err)
	}

	var parent DatedTaskViewRow
	var child DatedTaskViewRow
	for _, row := range results {
		switch row.ID {
		case "task-today-active":
			parent = row
		case "task-today-child":
			child = row
		}
	}
	if parent.ID == "" || child.ID == "" {
		t.Fatalf("expected parent and child in today results: %#v", results)
	}
	if parent.SubtaskCount != 2 || parent.OpenSubtaskCount != 1 || parent.IsSubtask {
		t.Fatalf("parent missing subtask count metadata: %#v", parent)
	}
	if child.ParentID != "task-today-active" ||
		child.ParentTitle != "Heute Aufgabe" ||
		!child.IsSubtask ||
		child.SubtaskCount != 0 ||
		child.OpenSubtaskCount != 0 {
		t.Fatalf("child missing relationship metadata: %#v", child)
	}
}

func TestListUpcomingTasksUsesConfiguredWindow(t *testing.T) {
	t.Parallel()

	database := openViewTestDB(t)
	seedViewTasks(t, database)

	if _, err := database.Conn.Exec(`UPDATE settings SET upcoming_days = 3 WHERE id = 'default';`); err != nil {
		t.Fatalf("update upcoming_days: %v", err)
	}

	results, err := database.ListUpcomingTasks(context.Background(), time.Date(2026, 4, 28, 0, 0, 0, 0, time.UTC), 50)
	if err != nil {
		t.Fatalf("list upcoming tasks: %v", err)
	}

	if len(results) != 1 || results[0].ID != "task-upcoming-in-range" {
		t.Fatalf("unexpected upcoming results: %#v", results)
	}
}

func TestListUpcomingTasksDefaultWindowIsSevenDays(t *testing.T) {
	t.Parallel()

	database := openViewTestDB(t)
	seedViewTasks(t, database)

	results, err := database.ListUpcomingTasks(context.Background(), time.Date(2026, 4, 28, 0, 0, 0, 0, time.UTC), 50)
	if err != nil {
		t.Fatalf("list upcoming tasks: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("unexpected upcoming result count: got %d want %d", len(results), 2)
	}
	if results[0].ID != "task-upcoming-in-range" || results[1].ID != "task-upcoming-day7" {
		t.Fatalf("unexpected upcoming results: %#v", results)
	}
}

func TestDateViewsRespectShowCompletedSetting(t *testing.T) {
	t.Parallel()

	database := openViewTestDB(t)
	seedViewTasks(t, database)
	referenceDate := time.Date(2026, 4, 28, 0, 0, 0, 0, time.UTC)

	overdueDefault, err := database.ListOverdueTasks(context.Background(), referenceDate, 50)
	if err != nil {
		t.Fatalf("list overdue tasks: %v", err)
	}
	if len(overdueDefault) != 1 || overdueDefault[0].ID != "task-overdue-active" {
		t.Fatalf("unexpected overdue results with default setting: %#v", overdueDefault)
	}

	if _, err := database.Conn.Exec(`UPDATE settings SET show_completed = TRUE WHERE id = 'default';`); err != nil {
		t.Fatalf("update show_completed: %v", err)
	}

	overdueWithCompleted, err := database.ListOverdueTasks(context.Background(), referenceDate, 50)
	if err != nil {
		t.Fatalf("list overdue tasks with completed: %v", err)
	}
	if len(overdueWithCompleted) != 2 {
		t.Fatalf("unexpected overdue result count with completed: got %d want %d", len(overdueWithCompleted), 2)
	}
}

func openViewTestDB(t *testing.T) *Database {
	t.Helper()

	database, err := OpenSQLite(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() {
		if err := database.Close(); err != nil {
			t.Fatalf("close sqlite: %v", err)
		}
	})

	return database
}

func seedViewTasks(t *testing.T, database *Database) {
	t.Helper()

	if _, err := database.Conn.Exec(`
INSERT INTO projects (
	id, calendar_href, display_name, sync_strategy, server_version, created_at, updated_at
) VALUES (
	'project-1', '/calendars/work', 'Work', 'fullscan', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
);

INSERT INTO tasks (
	id, project_id, uid, href, etag, server_version, title, description, status, raw_vtodo, base_vtodo,
	priority, label_names, project_name, sync_status, due_date, due_at, created_at, updated_at
) VALUES
(
	'task-overdue-active', 'project-1', 'uid-overdue-active', '/calendars/work/task-overdue-active.ics', '"etag-1"', 1,
	'Überfällige Aufgabe', '', 'needs-action', 'BEGIN:VTODO\nUID:uid-overdue-active\nEND:VTODO',
	'BEGIN:VTODO\nUID:uid-overdue-active\nEND:VTODO', NULL, '', 'Work', 'synced', '2026-04-27', NULL, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
),
(
	'task-overdue-completed', 'project-1', 'uid-overdue-completed', '/calendars/work/task-overdue-completed.ics', '"etag-2"', 1,
	'Überfällig erledigt', '', 'completed', 'BEGIN:VTODO\nUID:uid-overdue-completed\nEND:VTODO',
	'BEGIN:VTODO\nUID:uid-overdue-completed\nEND:VTODO', NULL, '', 'Work', 'synced', '2026-04-26', NULL, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
),
(
	'task-today-active', 'project-1', 'uid-today-active', '/calendars/work/task-today-active.ics', '"etag-3"', 3,
	'Heute Aufgabe', 'Heute Beschreibung', 'needs-action', 'BEGIN:VTODO\nUID:uid-today-active\nEND:VTODO',
	'BEGIN:VTODO\nUID:uid-today-active\nEND:VTODO', 4, 'Büro,urgent', 'Work', 'pending', '2026-04-28', NULL, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
),
(
	'task-upcoming-in-range', 'project-1', 'uid-upcoming-in-range', '/calendars/work/task-upcoming-in-range.ics', '"etag-4"', 1,
	'Bald Aufgabe', '', 'needs-action', 'BEGIN:VTODO\nUID:uid-upcoming-in-range\nEND:VTODO',
	'BEGIN:VTODO\nUID:uid-upcoming-in-range\nEND:VTODO', NULL, '', 'Work', 'synced', '2026-05-01', NULL, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
),
(
	'task-upcoming-day7', 'project-1', 'uid-upcoming-day7', '/calendars/work/task-upcoming-day7.ics', '"etag-5"', 1,
	'In 7 Tagen', '', 'needs-action', 'BEGIN:VTODO\nUID:uid-upcoming-day7\nEND:VTODO',
	'BEGIN:VTODO\nUID:uid-upcoming-day7\nEND:VTODO', NULL, '', 'Work', 'synced', '2026-05-05', NULL, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
),
(
	'task-upcoming-out-of-range', 'project-1', 'uid-upcoming-out-of-range', '/calendars/work/task-upcoming-out-of-range.ics', '"etag-6"', 1,
	'Später Aufgabe', '', 'needs-action', 'BEGIN:VTODO\nUID:uid-upcoming-out-of-range\nEND:VTODO',
	'BEGIN:VTODO\nUID:uid-upcoming-out-of-range\nEND:VTODO', NULL, '', 'Work', 'synced', '2026-05-06', NULL, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
),
(
	'task-without-due', 'project-1', 'uid-without-due', '/calendars/work/task-without-due.ics', '"etag-7"', 1,
	'Ohne Fälligkeit', '', 'needs-action', 'BEGIN:VTODO\nUID:uid-without-due\nEND:VTODO',
	'BEGIN:VTODO\nUID:uid-without-due\nEND:VTODO', NULL, '', 'Work', 'synced', NULL, NULL, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
);
`); err != nil {
		t.Fatalf("seed date view tasks: %v", err)
	}
}
