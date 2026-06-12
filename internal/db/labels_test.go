package db

import (
	"context"
	"errors"
	"path/filepath"
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
