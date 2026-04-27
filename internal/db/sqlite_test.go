package db

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOpenSQLiteConfiguresPragmasAndPool(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "caldo.db")
	database, err := OpenSQLite(dbPath)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() {
		if err := database.Close(); err != nil {
			t.Fatalf("close sqlite: %v", err)
		}
	})

	if _, err := os.Stat(dbPath); err != nil {
		t.Fatalf("expected sqlite file at configured path: %v", err)
	}

	if got := database.Conn.Stats().MaxOpenConnections; got != 1 {
		t.Fatalf("unexpected max open conns: got %d want 1", got)
	}

	assertSingleTextResult(t, database, "PRAGMA journal_mode;", "wal")
	assertSingleIntResult(t, database, "PRAGMA synchronous;", 1)
	assertSingleIntResult(t, database, "PRAGMA busy_timeout;", busyTimeoutMs)
	assertSingleIntResult(t, database, "PRAGMA foreign_keys;", 1)
}

func TestOpenSQLiteRunsMigrationsAndCreatesBackup(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "caldo.db")
	database, err := OpenSQLite(dbPath)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() {
		if err := database.Close(); err != nil {
			t.Fatalf("close sqlite: %v", err)
		}
	})

	assertSingleIntResult(t, database, `SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='schema_migrations';`, 1)
	assertSingleIntResult(t, database, `SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='settings';`, 1)
	assertSingleIntResult(t, database, `SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='projects';`, 1)
	assertSingleIntResult(t, database, `SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='tasks';`, 1)
	assertSingleIntResult(t, database, `SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='labels';`, 1)
	assertSingleIntResult(t, database, `SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='task_labels';`, 1)
	assertSingleIntResult(t, database, `SELECT COUNT(*) FROM settings WHERE id = 'default';`, 1)

	backupMatches, err := filepath.Glob(dbPath + ".backup-*")
	if err != nil {
		t.Fatalf("glob backup files: %v", err)
	}
	if len(backupMatches) == 0 {
		t.Fatal("expected backup file before first pending migration")
	}
}

func TestOpenSQLiteSeedsSettingsSingletonWithExpectedDefaults(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "caldo.db")
	database, err := OpenSQLite(dbPath)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() {
		if err := database.Close(); err != nil {
			t.Fatalf("close sqlite: %v", err)
		}
	})

	var (
		id                  string
		setupComplete       bool
		setupStep           string
		syncIntervalMinutes int
		uiLanguage          string
		darkMode            string
		defaultProjectID    *string
	)

	err = database.Conn.QueryRow(`
SELECT id, setup_complete, setup_step, sync_interval_minutes, ui_language, dark_mode, default_project_id
FROM settings
`).Scan(&id, &setupComplete, &setupStep, &syncIntervalMinutes, &uiLanguage, &darkMode, &defaultProjectID)
	if err != nil {
		t.Fatalf("query settings singleton: %v", err)
	}

	if id != "default" {
		t.Fatalf("unexpected settings id: got %q want %q", id, "default")
	}
	if setupComplete {
		t.Fatal("setup_complete should default to false")
	}
	if setupStep != "caldav" {
		t.Fatalf("unexpected setup_step default: got %q want %q", setupStep, "caldav")
	}
	if syncIntervalMinutes != 15 {
		t.Fatalf("unexpected sync interval default: got %d want %d", syncIntervalMinutes, 15)
	}
	if uiLanguage != "de" {
		t.Fatalf("unexpected ui language default: got %q want %q", uiLanguage, "de")
	}
	if darkMode != "system" {
		t.Fatalf("unexpected dark_mode default: got %q want %q", darkMode, "system")
	}
	if defaultProjectID != nil {
		t.Fatalf("default_project_id should be NULL before setup completion, got %q", *defaultProjectID)
	}
}

func TestSettingsSingletonRejectsNullID(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "caldo.db")
	database, err := OpenSQLite(dbPath)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() {
		if err := database.Close(); err != nil {
			t.Fatalf("close sqlite: %v", err)
		}
	})

	if _, err := database.Conn.Exec(`INSERT INTO settings (id, updated_at) VALUES (NULL, CURRENT_TIMESTAMP);`); err == nil {
		t.Fatal("expected NULL settings id insert to fail")
	}
}

func TestDatabaseCloseNilReceiver(t *testing.T) {
	t.Parallel()

	var database *Database
	if err := database.Close(); err != nil {
		t.Fatalf("nil close should be no-op: %v", err)
	}
}

func TestOpenSQLiteFailsWhenWalUnavailable(t *testing.T) {
	t.Parallel()

	database, err := OpenSQLite(":memory:")
	if err == nil {
		_ = database.Close()
		t.Fatal("expected OpenSQLite to fail when WAL mode is unavailable")
	}
}

func TestProjectsSyncStrategyConstraint(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "caldo.db")
	database, err := OpenSQLite(dbPath)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() {
		if err := database.Close(); err != nil {
			t.Fatalf("close sqlite: %v", err)
		}
	})

	for _, strategy := range []string{"webdav_sync", "ctag", "fullscan"} {
		if _, err := database.Conn.Exec(`
INSERT INTO projects (
    id, calendar_href, display_name, ctag, sync_token, sync_strategy, server_version, is_default, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
`, "project-"+strategy, "/calendars/"+strategy, "Project "+strategy, "ctag-1", "sync-token-1", strategy, 1, false); err != nil {
			t.Fatalf("insert project with sync_strategy %q: %v", strategy, err)
		}
	}

	if _, err := database.Conn.Exec(`
INSERT INTO projects (
    id, calendar_href, display_name, sync_strategy, created_at, updated_at
) VALUES ('project-invalid', '/calendars/invalid', 'Project invalid', 'invalid', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
`); err == nil {
		t.Fatal("expected invalid sync_strategy insert to fail")
	}
}

func TestProjectsCalendarHrefUniqueConstraint(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "caldo.db")
	database, err := OpenSQLite(dbPath)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() {
		if err := database.Close(); err != nil {
			t.Fatalf("close sqlite: %v", err)
		}
	})

	if _, err := database.Conn.Exec(`
INSERT INTO projects (
    id, calendar_href, display_name, sync_strategy, created_at, updated_at
) VALUES ('project-1', '/calendars/duplicate', 'Project One', 'fullscan', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
`); err != nil {
		t.Fatalf("insert first project: %v", err)
	}

	if _, err := database.Conn.Exec(`
INSERT INTO projects (
    id, calendar_href, display_name, sync_strategy, created_at, updated_at
) VALUES ('project-2', '/calendars/duplicate', 'Project Two', 'fullscan', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
`); err == nil {
		t.Fatal("expected duplicate calendar_href insert to fail")
	}
}

func TestProjectsOptimisticLockingByServerVersion(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "caldo.db")
	database, err := OpenSQLite(dbPath)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() {
		if err := database.Close(); err != nil {
			t.Fatalf("close sqlite: %v", err)
		}
	})

	if _, err := database.Conn.Exec(`
INSERT INTO projects (
    id, calendar_href, display_name, sync_strategy, server_version, is_default, created_at, updated_at
) VALUES ('project-1', '/calendars/p1', 'Initial name', 'fullscan', 1, TRUE, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
`); err != nil {
		t.Fatalf("insert project: %v", err)
	}

	result, err := database.Conn.Exec(`
UPDATE projects
SET display_name = ?, server_version = server_version + 1, updated_at = CURRENT_TIMESTAMP
WHERE id = ? AND server_version = ?
`, "Renamed project", "project-1", 1)
	if err != nil {
		t.Fatalf("update project with expected version: %v", err)
	}
	assertRowsAffected(t, result, 1)

	result, err = database.Conn.Exec(`
UPDATE projects
SET display_name = ?, server_version = server_version + 1, updated_at = CURRENT_TIMESTAMP
WHERE id = ? AND server_version = ?
`, "Should fail", "project-1", 1)
	if err != nil {
		t.Fatalf("update project with stale expected version: %v", err)
	}
	assertRowsAffected(t, result, 0)

	var (
		displayName   string
		serverVersion int
		isDefault     bool
	)
	err = database.Conn.QueryRow(`
SELECT display_name, server_version, is_default
FROM projects
WHERE id = 'project-1'
`).Scan(&displayName, &serverVersion, &isDefault)
	if err != nil {
		t.Fatalf("query project: %v", err)
	}

	if displayName != "Renamed project" {
		t.Fatalf("unexpected display_name: got %q want %q", displayName, "Renamed project")
	}
	if serverVersion != 2 {
		t.Fatalf("unexpected server_version: got %d want %d", serverVersion, 2)
	}
	if !isDefault {
		t.Fatal("expected is_default to remain true")
	}
}

func TestTasksTablePersistsRequiredTaskModelFields(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "caldo.db")
	database, err := OpenSQLite(dbPath)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() {
		if err := database.Close(); err != nil {
			t.Fatalf("close sqlite: %v", err)
		}
	})

	if _, err := database.Conn.Exec(`
INSERT INTO projects (
    id, calendar_href, display_name, sync_strategy, created_at, updated_at
) VALUES ('project-1', '/calendars/p1', 'Project 1', 'fullscan', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
`); err != nil {
		t.Fatalf("insert project: %v", err)
	}

	if _, err := database.Conn.Exec(`
INSERT INTO tasks (
    id, project_id, uid, href, etag, server_version,
    title, description, status, completed_at, due_date, due_at, priority, rrule,
    parent_id, raw_vtodo, base_vtodo, label_names, project_name, sync_status, created_at, updated_at
) VALUES (
    'task-1', 'project-1', 'uid-1', '/tasks/1.ics', 'etag-1', 3,
    'My task', 'Task description', 'needs-action', NULL, '2026-04-27', '2026-04-27 12:00:00', 2, 'FREQ=DAILY',
    NULL, 'BEGIN:VTODO\\nUID:uid-1\\nEND:VTODO', 'BEGIN:VTODO\\nUID:uid-1\\nEND:VTODO',
    'home,urgent', 'Project 1', 'pending', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
)
`); err != nil {
		t.Fatalf("insert task with complete model fields: %v", err)
	}

	var (
		projectID     string
		uid           string
		href          string
		etag          string
		serverVersion int
		title         string
		description   string
		status        string
		dueDate       string
		priority      int
		rrule         string
		rawVTODO      string
		baseVTODO     string
		syncStatus    string
		parentID      *string
		labelNames    string
		projectName   string
	)

	err = database.Conn.QueryRow(`
SELECT
    project_id, uid, href, etag, server_version, title, description, status, due_date, priority, rrule,
    raw_vtodo, base_vtodo, sync_status, parent_id, label_names, project_name
FROM tasks
WHERE id = 'task-1'
`).Scan(
		&projectID, &uid, &href, &etag, &serverVersion, &title, &description, &status, &dueDate, &priority, &rrule,
		&rawVTODO, &baseVTODO, &syncStatus, &parentID, &labelNames, &projectName,
	)
	if err != nil {
		t.Fatalf("query inserted task: %v", err)
	}

	if projectID != "project-1" || uid != "uid-1" || href != "/tasks/1.ics" || etag != "etag-1" {
		t.Fatalf("unexpected remote identity fields: got project_id=%q uid=%q href=%q etag=%q", projectID, uid, href, etag)
	}
	if serverVersion != 3 {
		t.Fatalf("unexpected server_version: got %d want 3", serverVersion)
	}
	if title != "My task" || description != "Task description" || status != "needs-action" {
		t.Fatalf("unexpected normalized task fields: got title=%q description=%q status=%q", title, description, status)
	}
	if dueDate != "2026-04-27T00:00:00Z" || priority != 2 || rrule != "FREQ=DAILY" {
		t.Fatalf("unexpected scheduling fields: got due_date=%q priority=%d rrule=%q", dueDate, priority, rrule)
	}
	if rawVTODO == "" {
		t.Fatal("expected raw_vtodo to be persisted")
	}
	if baseVTODO == "" {
		t.Fatal("expected base_vtodo to be persisted")
	}
	if syncStatus != "pending" {
		t.Fatalf("unexpected sync_status: got %q want %q", syncStatus, "pending")
	}
	if parentID != nil {
		t.Fatalf("expected parent_id to be NULL, got %q", *parentID)
	}
	if labelNames != "home,urgent" || projectName != "Project 1" {
		t.Fatalf("unexpected denormalized search fields: got label_names=%q project_name=%q", labelNames, projectName)
	}
}

func TestTasksProjectAndParentForeignKeys(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "caldo.db")
	database, err := OpenSQLite(dbPath)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() {
		if err := database.Close(); err != nil {
			t.Fatalf("close sqlite: %v", err)
		}
	})

	if _, err := database.Conn.Exec(`
INSERT INTO projects (
    id, calendar_href, display_name, sync_strategy, created_at, updated_at
) VALUES ('project-1', '/calendars/p1', 'Project 1', 'fullscan', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
`); err != nil {
		t.Fatalf("insert project: %v", err)
	}

	if _, err := database.Conn.Exec(`
INSERT INTO tasks (
    id, project_id, uid, title, status, raw_vtodo, sync_status, created_at, updated_at
) VALUES (
    'parent-task', 'project-1', 'uid-parent', 'Parent', 'needs-action', 'BEGIN:VTODO\\nUID:uid-parent\\nEND:VTODO',
    'synced', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
)
`); err != nil {
		t.Fatalf("insert parent task: %v", err)
	}

	if _, err := database.Conn.Exec(`
INSERT INTO tasks (
    id, project_id, uid, title, status, parent_id, raw_vtodo, sync_status, created_at, updated_at
) VALUES (
    'child-task', 'project-1', 'uid-child', 'Child', 'needs-action', 'parent-task', 'BEGIN:VTODO\\nUID:uid-child\\nEND:VTODO',
    'pending', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
)
`); err != nil {
		t.Fatalf("insert child task: %v", err)
	}

	if _, err := database.Conn.Exec(`
INSERT INTO tasks (
    id, project_id, uid, title, status, raw_vtodo, sync_status, created_at, updated_at
) VALUES (
    'orphan-project', 'missing-project', 'uid-2', 'Orphan', 'needs-action', 'BEGIN:VTODO\\nUID:uid-2\\nEND:VTODO',
    'pending', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
)
`); err == nil {
		t.Fatal("expected missing project foreign key violation")
	}

	if _, err := database.Conn.Exec(`
INSERT INTO tasks (
    id, project_id, uid, title, status, parent_id, raw_vtodo, sync_status, created_at, updated_at
) VALUES (
    'orphan-parent', 'project-1', 'uid-3', 'Orphan parent ref', 'needs-action', 'missing-parent',
    'BEGIN:VTODO\\nUID:uid-3\\nEND:VTODO', 'pending', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
)
`); err == nil {
		t.Fatal("expected missing parent task foreign key violation")
	}
}

func TestLabelsAndTaskLabelsConstraints(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "caldo.db")
	database, err := OpenSQLite(dbPath)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() {
		if err := database.Close(); err != nil {
			t.Fatalf("close sqlite: %v", err)
		}
	})

	if _, err := database.Conn.Exec(`
INSERT INTO projects (
    id, calendar_href, display_name, sync_strategy, created_at, updated_at
) VALUES ('project-1', '/calendars/p1', 'Project 1', 'fullscan', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
`); err != nil {
		t.Fatalf("insert project: %v", err)
	}

	if _, err := database.Conn.Exec(`
INSERT INTO tasks (
    id, project_id, uid, title, status, raw_vtodo, sync_status, created_at, updated_at
) VALUES (
    'task-1', 'project-1', 'uid-1', 'Task 1', 'needs-action', 'BEGIN:VTODO\\nUID:uid-1\\nEND:VTODO',
    'pending', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
)
`); err != nil {
		t.Fatalf("insert task: %v", err)
	}

	if _, err := database.Conn.Exec(`
INSERT INTO labels (id, name, created_at) VALUES ('label-home', 'home', CURRENT_TIMESTAMP)
`); err != nil {
		t.Fatalf("insert first label: %v", err)
	}

	if _, err := database.Conn.Exec(`
INSERT INTO labels (id, name, created_at) VALUES ('label-home-duplicate', 'home', CURRENT_TIMESTAMP)
`); err == nil {
		t.Fatal("expected duplicate label name insert to fail")
	}

	if _, err := database.Conn.Exec(`
INSERT INTO labels (id, name, created_at) VALUES ('label-home-uppercase', 'HOME', CURRENT_TIMESTAMP)
`); err == nil {
		t.Fatal("expected case-insensitive duplicate label name insert to fail")
	}

	if _, err := database.Conn.Exec(`
INSERT INTO labels (id, name, created_at) VALUES ('label-starred', 'STARRED', CURRENT_TIMESTAMP)
`); err == nil {
		t.Fatal("expected reserved STARRED label insert to fail")
	}

	if _, err := database.Conn.Exec(`
INSERT INTO labels (id, name, created_at) VALUES ('label-urgent', 'urgent', CURRENT_TIMESTAMP)
`); err != nil {
		t.Fatalf("insert second label: %v", err)
	}

	if _, err := database.Conn.Exec(`
INSERT INTO task_labels (task_id, label_id) VALUES ('task-1', 'label-home')
`); err != nil {
		t.Fatalf("attach first label to task: %v", err)
	}
	if _, err := database.Conn.Exec(`
INSERT INTO task_labels (task_id, label_id) VALUES ('task-1', 'label-urgent')
`); err != nil {
		t.Fatalf("attach second label to task: %v", err)
	}

	if _, err := database.Conn.Exec(`
INSERT INTO task_labels (task_id, label_id) VALUES ('task-1', 'label-home')
`); err == nil {
		t.Fatal("expected duplicate task-label assignment to fail")
	}

	assertSingleIntResult(t, database, `SELECT COUNT(*) FROM task_labels WHERE task_id = 'task-1';`, 2)
}

func assertSingleTextResult(t *testing.T, database *Database, query, want string) {
	t.Helper()

	var got string
	if err := database.Conn.QueryRow(query).Scan(&got); err != nil {
		t.Fatalf("query %q failed: %v", query, err)
	}

	if got != want {
		t.Fatalf("unexpected result for %q: got %q want %q", query, got, want)
	}
}

func assertSingleIntResult(t *testing.T, database *Database, query string, want int) {
	t.Helper()

	var got int
	if err := database.Conn.QueryRow(query).Scan(&got); err != nil {
		t.Fatalf("query %q failed: %v", query, err)
	}

	if got != want {
		t.Fatalf("unexpected result for %q: got %d want %d", query, got, want)
	}
}

func assertRowsAffected(t *testing.T, result interface{ RowsAffected() (int64, error) }, want int64) {
	t.Helper()

	got, err := result.RowsAffected()
	if err != nil {
		t.Fatalf("rows affected failed: %v", err)
	}

	if got != want {
		t.Fatalf("unexpected rows affected: got %d want %d", got, want)
	}
}
