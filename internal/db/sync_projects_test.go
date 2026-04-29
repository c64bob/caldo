package db

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
)

func TestListSyncProjectsReturnsMetadata(t *testing.T) {
	database := openSyncProjectsTestDB(t)
	t.Cleanup(func() { _ = database.Close() })

	_, err := database.Conn.Exec(`
INSERT INTO projects (id, calendar_href, display_name, sync_strategy, sync_token, ctag, created_at, updated_at)
VALUES
('p1', '/cal/one/', 'One', 'webdav_sync', 'token-1', 'ctag-1', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('p2', '/cal/two/', 'Two', 'fullscan', NULL, NULL, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
`)
	if err != nil {
		t.Fatalf("seed projects: %v", err)
	}

	projects, err := database.ListSyncProjects(context.Background())
	if err != nil {
		t.Fatalf("list sync projects: %v", err)
	}
	if len(projects) != 2 {
		t.Fatalf("unexpected project count: %d", len(projects))
	}
	if projects[0].SyncToken != "token-1" || projects[0].CTag != "ctag-1" {
		t.Fatalf("unexpected project metadata: %+v", projects[0])
	}
}

func TestUpdateProjectSyncStrategyIncrementsVersion(t *testing.T) {
	database := openSyncProjectsTestDB(t)
	t.Cleanup(func() { _ = database.Close() })

	_, err := database.Conn.Exec(`
INSERT INTO projects (id, calendar_href, display_name, sync_strategy, server_version, created_at, updated_at)
VALUES ('p1', '/cal/one/', 'One', 'webdav_sync', 5, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
`)
	if err != nil {
		t.Fatalf("seed project: %v", err)
	}

	if err := database.UpdateProjectSyncStrategy(context.Background(), "p1", "ctag"); err != nil {
		t.Fatalf("update strategy: %v", err)
	}

	var strategy string
	var version int
	if err := database.Conn.QueryRowContext(context.Background(), `SELECT sync_strategy, server_version FROM projects WHERE id='p1';`).Scan(&strategy, &version); err != nil {
		t.Fatalf("load project: %v", err)
	}
	if strategy != "ctag" || version != 6 {
		t.Fatalf("unexpected state: strategy=%q version=%d", strategy, version)
	}
}

func TestUpdateProjectSyncStrategyUnknownProject(t *testing.T) {
	database := openSyncProjectsTestDB(t)
	t.Cleanup(func() { _ = database.Close() })

	err := database.UpdateProjectSyncStrategy(context.Background(), "missing", "fullscan")
	if !errors.Is(err, ErrProjectNotFound) {
		t.Fatalf("expected ErrProjectNotFound, got %v", err)
	}
}

func openSyncProjectsTestDB(t *testing.T) *Database {
	t.Helper()
	database, err := OpenSQLite(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	return database
}
