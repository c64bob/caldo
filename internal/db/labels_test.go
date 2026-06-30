package db

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadLabelDetailReturnsCounters(t *testing.T) {
	t.Parallel()

	database := openViewTestDB(t)
	seedViewTasks(t, database)
	if _, err := database.Conn.Exec(`
INSERT INTO labels (id, name, created_at) VALUES ('label-buro', 'Büro', CURRENT_TIMESTAMP);
INSERT INTO task_labels (task_id, label_id) VALUES
	('task-today-active', 'label-buro'),
	('task-overdue-completed', 'label-buro');
`); err != nil {
		t.Fatalf("seed label detail: %v", err)
	}

	label, err := database.LoadLabelDetail(context.Background(), "label-buro")
	if err != nil {
		t.Fatalf("load label detail: %v", err)
	}
	if label.ID != "label-buro" || label.Name != "Büro" || label.OpenTaskCount != 1 || label.TaskCount != 2 {
		t.Fatalf("unexpected label detail: %#v", label)
	}
}

func TestLoadLabelDetailReturnsNotFound(t *testing.T) {
	t.Parallel()

	database := openViewTestDB(t)

	_, err := database.LoadLabelDetail(context.Background(), "missing")
	if !errors.Is(err, ErrLabelNotFound) {
		t.Fatalf("expected ErrLabelNotFound, got %v", err)
	}
}

func TestListLabelOptionsOrdersLabels(t *testing.T) {
	t.Parallel()

	database, err := OpenSQLite(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	if _, err := database.Conn.ExecContext(context.Background(), `
INSERT INTO labels (id, name, created_at) VALUES
('label-work', 'Work', CURRENT_TIMESTAMP),
('label-alpha', 'alpha', CURRENT_TIMESTAMP),
('label-home', 'home', CURRENT_TIMESTAMP);
`); err != nil {
		t.Fatalf("seed labels: %v", err)
	}

	options, err := database.ListLabelOptions(context.Background())
	if err != nil {
		t.Fatalf("list label options: %v", err)
	}
	if len(options) != 3 {
		t.Fatalf("unexpected label count: got %d labels=%#v", len(options), options)
	}
	if options[0].Name != "alpha" || options[1].Name != "home" || options[2].Name != "Work" {
		t.Fatalf("unexpected label order: %#v", options)
	}
}

func TestListLabelMutationTasksReturnsAllAssignedTasks(t *testing.T) {
	t.Parallel()

	database := openLabelMutationTestDB(t)
	seedLabelMutationTestData(t, database)

	tasks, err := database.ListLabelMutationTasks(context.Background(), "label-home")
	if err != nil {
		t.Fatalf("list label mutation tasks: %v", err)
	}
	if len(tasks) != 2 {
		t.Fatalf("unexpected task count: got %d tasks=%#v", len(tasks), tasks)
	}
	if tasks[0].ID != "task-1" || tasks[1].ID != "task-2" {
		t.Fatalf("unexpected task order: %#v", tasks)
	}
	if tasks[1].ServerVersion != 4 || tasks[1].ETag != `"etag-2"` || !strings.Contains(tasks[1].RawVTODO, "UID:uid-2") {
		t.Fatalf("unexpected mutation task fields: %#v", tasks[1])
	}
}

func TestRenameUnusedLabelRenamesAndMergesWithExistingLabel(t *testing.T) {
	t.Parallel()

	database := openLabelMutationTestDB(t)
	if _, err := database.Conn.ExecContext(context.Background(), `
INSERT INTO labels (id, name, created_at) VALUES
	('label-empty', 'Empty', CURRENT_TIMESTAMP),
	('label-existing', 'Existing', CURRENT_TIMESTAMP);
`); err != nil {
		t.Fatalf("seed labels: %v", err)
	}

	if err := database.RenameUnusedLabel(context.Background(), "label-empty", "Renamed"); err != nil {
		t.Fatalf("rename unused label: %v", err)
	}
	assertSingleTextResult(t, database, `SELECT name FROM labels WHERE id = 'label-empty';`, "Renamed")

	if err := database.RenameUnusedLabel(context.Background(), "label-empty", "Existing"); err != nil {
		t.Fatalf("merge unused label: %v", err)
	}
	assertSingleIntResult(t, database, `SELECT COUNT(*) FROM labels WHERE id = 'label-empty';`, 0)
	assertSingleIntResult(t, database, `SELECT COUNT(*) FROM labels WHERE id = 'label-existing';`, 1)
}

func TestUnusedLabelMutationsRejectInUseLabel(t *testing.T) {
	t.Parallel()

	database := openLabelMutationTestDB(t)
	seedLabelMutationTestData(t, database)

	if err := database.RenameUnusedLabel(context.Background(), "label-home", "Renamed"); !errors.Is(err, ErrLabelStillInUse) {
		t.Fatalf("expected ErrLabelStillInUse for rename, got %v", err)
	}
	if err := database.DeleteUnusedLabel(context.Background(), "label-home"); !errors.Is(err, ErrLabelStillInUse) {
		t.Fatalf("expected ErrLabelStillInUse for delete, got %v", err)
	}

	if _, err := database.Conn.ExecContext(context.Background(), `INSERT INTO labels (id, name, created_at) VALUES ('label-empty', 'Empty', CURRENT_TIMESTAMP);`); err != nil {
		t.Fatalf("seed empty label: %v", err)
	}
	if err := database.DeleteUnusedLabel(context.Background(), "label-empty"); err != nil {
		t.Fatalf("delete unused label: %v", err)
	}
	assertSingleIntResult(t, database, `SELECT COUNT(*) FROM labels WHERE id = 'label-empty';`, 0)
}

func TestRenameLabelRowAllowsInUseCaseOnlyRename(t *testing.T) {
	t.Parallel()

	database := openLabelMutationTestDB(t)
	seedLabelMutationTestData(t, database)

	if err := database.RenameLabelRow(context.Background(), "label-home", "Home"); err != nil {
		t.Fatalf("rename label row: %v", err)
	}
	assertSingleTextResult(t, database, `SELECT name FROM labels WHERE id = 'label-home';`, "Home")
	assertSingleIntResult(t, database, `SELECT COUNT(*) FROM task_labels WHERE label_id = 'label-home';`, 2)
}

func openLabelMutationTestDB(t *testing.T) *Database {
	t.Helper()

	database, err := OpenSQLite(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	return database
}

func seedLabelMutationTestData(t *testing.T, database *Database) {
	t.Helper()

	if _, err := database.Conn.ExecContext(context.Background(), `
INSERT INTO projects (id, calendar_href, display_name, sync_strategy, created_at, updated_at)
VALUES ('project-1', '/cal/work/', 'Work', 'fullscan', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
INSERT INTO labels (id, name, created_at) VALUES ('label-home', 'home', CURRENT_TIMESTAMP);
INSERT INTO tasks (
	id, project_id, uid, href, etag, server_version, title, status, raw_vtodo, base_vtodo,
	label_names, project_name, sync_status, created_at, updated_at
) VALUES
	('task-1', 'project-1', 'uid-1', '/cal/work/task-1.ics', '"etag-1"', 3, 'Task 1', 'needs-action',
	 'BEGIN:VTODO\nUID:uid-1\nSUMMARY:Task 1\nCATEGORIES:home\nEND:VTODO',
	 'BEGIN:VTODO\nUID:uid-1\nSUMMARY:Task 1\nCATEGORIES:home\nEND:VTODO',
	 'home', 'Work', 'synced', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
	('task-2', 'project-1', 'uid-2', '/cal/work/task-2.ics', '"etag-2"', 4, 'Task 2', 'completed',
	 'BEGIN:VTODO\nUID:uid-2\nSUMMARY:Task 2\nCATEGORIES:home\nEND:VTODO',
	 'BEGIN:VTODO\nUID:uid-2\nSUMMARY:Task 2\nCATEGORIES:home\nEND:VTODO',
	 'home', 'Work', 'synced', CURRENT_TIMESTAMP, DATETIME(CURRENT_TIMESTAMP, '+1 second'));
INSERT INTO task_labels (task_id, label_id) VALUES
	('task-1', 'label-home'),
	('task-2', 'label-home');
`); err != nil {
		t.Fatalf("seed label mutation data: %v", err)
	}
}
