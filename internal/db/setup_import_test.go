package db

import (
	"context"
	"database/sql"
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

func TestReplaceSetupProjectTasksFlattensNestedSubtasksToSingleLevel(t *testing.T) {
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

	tasks := []ImportedTask{
		{UID: "uid-parent", ParentUID: "", Title: "Parent", Status: "needs-action", RawVTODO: "BEGIN:VTODO\nUID:uid-parent\nEND:VTODO", BaseVTODO: "BEGIN:VTODO\nUID:uid-parent\nEND:VTODO"},
		{UID: "uid-child", ParentUID: "uid-parent", Title: "Child", Status: "needs-action", RawVTODO: "BEGIN:VTODO\nUID:uid-child\nRELATED-TO;RELTYPE=PARENT:uid-parent\nEND:VTODO", BaseVTODO: "BEGIN:VTODO\nUID:uid-child\nRELATED-TO;RELTYPE=PARENT:uid-parent\nEND:VTODO"},
		{UID: "uid-grandchild", ParentUID: "uid-child", Title: "Grandchild", Status: "needs-action", RawVTODO: "BEGIN:VTODO\nUID:uid-grandchild\nRELATED-TO;RELTYPE=PARENT:uid-child\nEND:VTODO", BaseVTODO: "BEGIN:VTODO\nUID:uid-grandchild\nRELATED-TO;RELTYPE=PARENT:uid-child\nEND:VTODO"},
	}
	if err := database.ReplaceSetupProjectTasks(context.Background(), "project-1", tasks); err != nil {
		t.Fatalf("replace setup tasks: %v", err)
	}

	rows, err := database.Conn.Query(`SELECT uid, parent_id FROM tasks WHERE project_id='project-1'`)
	if err != nil {
		t.Fatalf("query tasks: %v", err)
	}
	defer rows.Close()
	parents := map[string]string{}
	for rows.Next() {
		var uid string
		var parent sql.NullString
		if err := rows.Scan(&uid, &parent); err != nil {
			t.Fatalf("scan: %v", err)
		}
		if parent.Valid {
			parents[uid] = parent.String
		}
	}
	if len(parents) != 1 {
		t.Fatalf("expected exactly one linked subtask, got %d", len(parents))
	}
	if _, ok := parents["uid-grandchild"]; ok {
		t.Fatalf("expected grandchild to be imported as root")
	}
}
