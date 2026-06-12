package db

import (
	"context"
	"path/filepath"
	"testing"
)

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
