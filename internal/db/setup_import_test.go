package db

import (
	"context"
	"path/filepath"
	"testing"
)

func TestReplaceSetupProjectTasksWritesSyncedWithBaseVTODO(t *testing.T) {
	t.Parallel()

	database, err := OpenSQLite(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	if _, err := database.Conn.Exec(`
INSERT INTO projects (id, calendar_href, display_name, sync_strategy, created_at, updated_at)
VALUES ('project-1', '/cal/work/', 'Work', 'fullscan', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
`); err != nil {
		t.Fatalf("insert project: %v", err)
	}

	dueAt := "2026-04-30T10:11:12Z"
	if err := database.ReplaceSetupProjectTasks(context.Background(), "project-1", []ImportedTask{{
		UID:         "uid-1",
		Href:        "/cal/work/uid-1.ics",
		ETag:        "\"etag-1\"",
		Title:       "Task 1",
		Status:      "needs-action",
		DueAt:       &dueAt,
		RawVTODO:    "BEGIN:VTODO\nUID:uid-1\nEND:VTODO",
		BaseVTODO:   "BEGIN:VTODO\nUID:uid-1\nEND:VTODO",
		LabelNames:  []string{"home", "errands"},
		ProjectName: "Work",
	}}); err != nil {
		t.Fatalf("replace setup tasks: %v", err)
	}

	var syncStatus, rawVTODO, baseVTODO, labelNames, projectName string
	if err := database.Conn.QueryRow(`
SELECT sync_status, raw_vtodo, base_vtodo, label_names, project_name
FROM tasks
WHERE project_id = 'project-1' AND uid = 'uid-1'
`).Scan(&syncStatus, &rawVTODO, &baseVTODO, &labelNames, &projectName); err != nil {
		t.Fatalf("query task: %v", err)
	}

	if syncStatus != "synced" {
		t.Fatalf("unexpected sync status: got %q", syncStatus)
	}
	if rawVTODO == "" || baseVTODO == "" || rawVTODO != baseVTODO {
		t.Fatalf("expected base_vtodo to equal raw_vtodo")
	}
	if labelNames == "" || projectName != "Work" {
		t.Fatalf("expected denormalized fields to be populated, got label_names=%q project_name=%q", labelNames, projectName)
	}
}

func TestLoadSetupImportProjects(t *testing.T) {
	t.Parallel()

	database, err := OpenSQLite(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	if _, err := database.Conn.Exec(`
INSERT INTO projects (id, calendar_href, display_name, sync_strategy, created_at, updated_at)
VALUES ('project-a', '/cal/a/', 'A', 'fullscan', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
       ('project-b', '/cal/b/', 'B', 'fullscan', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
`); err != nil {
		t.Fatalf("insert projects: %v", err)
	}

	projects, err := database.LoadSetupImportProjects(context.Background())
	if err != nil {
		t.Fatalf("load setup import projects: %v", err)
	}
	if len(projects) != 2 {
		t.Fatalf("unexpected projects count: %d", len(projects))
	}
}
