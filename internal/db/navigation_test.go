package db

import (
	"context"
	"testing"
	"time"
)

func TestLoadNavigationSnapshotReturnsCountsAndEntries(t *testing.T) {
	t.Parallel()

	database := openViewTestDB(t)
	seedNavigationData(t, database)

	snapshot, err := database.LoadNavigationSnapshot(context.Background(), time.Date(2026, 4, 28, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("load navigation snapshot: %v", err)
	}

	if snapshot.TodayCount != 2 {
		t.Fatalf("today count: got %d want 2", snapshot.TodayCount)
	}
	if snapshot.UpcomingCount != 1 {
		t.Fatalf("upcoming count: got %d want 1", snapshot.UpcomingCount)
	}
	if snapshot.OverdueCount != 1 {
		t.Fatalf("overdue count: got %d want 1", snapshot.OverdueCount)
	}
	if snapshot.FavoriteCount != 1 {
		t.Fatalf("favorite count: got %d want 1", snapshot.FavoriteCount)
	}
	if snapshot.NoDateCount != 1 {
		t.Fatalf("no date count: got %d want 1", snapshot.NoDateCount)
	}
	if snapshot.ConflictCount != 1 {
		t.Fatalf("conflict count: got %d want 1", snapshot.ConflictCount)
	}
	if len(snapshot.Projects) != 2 || snapshot.Projects[0].Name != "Inbox" || snapshot.Projects[0].OpenTaskCount != 3 {
		t.Fatalf("unexpected projects: %#v", snapshot.Projects)
	}
	if len(snapshot.Labels) != 2 || snapshot.Labels[0].Name != "Büro" || snapshot.Labels[0].OpenTaskCount != 1 {
		t.Fatalf("unexpected labels: %#v", snapshot.Labels)
	}
	if len(snapshot.SavedFilters) != 2 || snapshot.SavedFilters[0].Name != "Favorit" {
		t.Fatalf("unexpected filters: %#v", snapshot.SavedFilters)
	}
}

func seedNavigationData(t *testing.T, database *Database) {
	t.Helper()

	if _, err := database.Conn.Exec(`
UPDATE settings SET upcoming_days = 3 WHERE id = 'default';

INSERT INTO projects (
	id, calendar_href, display_name, sync_strategy, server_version, is_default, created_at, updated_at
) VALUES
	('project-inbox', '/calendars/inbox', 'Inbox', 'fullscan', 1, TRUE, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
	('project-work', '/calendars/work', 'Work', 'fullscan', 1, FALSE, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);

INSERT INTO labels (id, name, created_at) VALUES
	('label-buro', 'Büro', CURRENT_TIMESTAMP),
	('label-home', 'Home', CURRENT_TIMESTAMP);

INSERT INTO tasks (
	id, project_id, uid, href, etag, server_version, title, description, status, raw_vtodo, base_vtodo,
	label_names, project_name, sync_status, due_date, due_at, created_at, updated_at
) VALUES
	('task-overdue', 'project-inbox', 'uid-overdue', '/calendars/inbox/overdue.ics', '"etag-1"', 1,
		'Overdue', '', 'needs-action', 'BEGIN:VTODO\nUID:uid-overdue\nEND:VTODO',
		'BEGIN:VTODO\nUID:uid-overdue\nEND:VTODO', 'Büro STARRED', 'Inbox', 'synced', '2026-04-27', NULL, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
	('task-today', 'project-inbox', 'uid-today', '/calendars/inbox/today.ics', '"etag-2"', 1,
		'Today', '', 'needs-action', 'BEGIN:VTODO\nUID:uid-today\nEND:VTODO',
		'BEGIN:VTODO\nUID:uid-today\nEND:VTODO', '', 'Inbox', 'synced', '2026-04-28', NULL, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
	('task-upcoming', 'project-work', 'uid-upcoming', '/calendars/work/upcoming.ics', '"etag-3"', 1,
		'Upcoming', '', 'needs-action', 'BEGIN:VTODO\nUID:uid-upcoming\nEND:VTODO',
		'BEGIN:VTODO\nUID:uid-upcoming\nEND:VTODO', 'Home', 'Work', 'synced', '2026-05-01', NULL, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
	('task-no-date', 'project-inbox', 'uid-no-date', '/calendars/inbox/no-date.ics', '"etag-4"', 1,
		'No date', '', 'needs-action', 'BEGIN:VTODO\nUID:uid-no-date\nEND:VTODO',
		'BEGIN:VTODO\nUID:uid-no-date\nEND:VTODO', '', 'Inbox', 'synced', NULL, NULL, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
	('task-completed', 'project-work', 'uid-completed', '/calendars/work/completed.ics', '"etag-5"', 1,
		'Done', '', 'completed', 'BEGIN:VTODO\nUID:uid-completed\nEND:VTODO',
		'BEGIN:VTODO\nUID:uid-completed\nEND:VTODO', 'Büro', 'Work', 'synced', '2026-04-28', NULL, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);

INSERT INTO task_labels (task_id, label_id) VALUES
	('task-overdue', 'label-buro'),
	('task-completed', 'label-buro'),
	('task-upcoming', 'label-home');

INSERT INTO conflicts (id, task_id, project_id, conflict_type, created_at)
VALUES ('conflict-open', 'task-overdue', 'project-inbox', 'field_conflict', CURRENT_TIMESTAMP);
INSERT INTO conflicts (id, task_id, project_id, conflict_type, created_at, resolved_at)
VALUES ('conflict-resolved', 'task-overdue', 'project-inbox', 'field_conflict', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);

INSERT INTO saved_filters (id, name, query, is_favorite)
VALUES ('filter-normal', 'Normal', 'today', 0), ('filter-favorite', 'Favorit', 'starred:true', 1);
`); err != nil {
		t.Fatalf("seed navigation data: %v", err)
	}
}
