package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

// ErrProjectDeleteConfirmationMismatch indicates strong confirmation did not match the project name.
var ErrProjectDeleteConfirmationMismatch = errors.New("project delete confirmation mismatch")

// ProjectDeleteBase captures persisted project metadata required for delete operations.
type ProjectDeleteBase struct {
	ProjectID          string
	CalendarHref       string
	CurrentName        string
	CurrentVersion     int
	ReservedVersion    int
	ExpectedVersion    int
	AffectedTaskCount  int
	ConfirmationString string
}

// LoadProjectDeleteBase loads project metadata, validates confirmation, and reserves a project version before delete.
func (d *Database) LoadProjectDeleteBase(ctx context.Context, projectID string, expectedVersion int, confirmation string) (ProjectDeleteBase, error) {
	trimmedProjectID := strings.TrimSpace(projectID)
	if trimmedProjectID == "" {
		return ProjectDeleteBase{}, fmt.Errorf("load project delete base: project id is required")
	}
	if expectedVersion < 1 {
		return ProjectDeleteBase{}, fmt.Errorf("load project delete base: expected version is required")
	}

	base := ProjectDeleteBase{
		ProjectID:          trimmedProjectID,
		ExpectedVersion:    expectedVersion,
		ConfirmationString: strings.TrimSpace(confirmation),
	}

	d.WriteMu.Lock()
	defer d.WriteMu.Unlock()

	tx, err := d.Conn.BeginTx(ctx, nil)
	if err != nil {
		return ProjectDeleteBase{}, fmt.Errorf("load project delete base: begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := tx.QueryRowContext(ctx, `
SELECT calendar_href, display_name, server_version
FROM projects
WHERE id = ?;
`, trimmedProjectID).Scan(&base.CalendarHref, &base.CurrentName, &base.CurrentVersion); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ProjectDeleteBase{}, ErrProjectNotFound
		}
		return ProjectDeleteBase{}, fmt.Errorf("load project delete base: query project: %w", err)
	}

	if base.CurrentVersion != expectedVersion {
		return ProjectDeleteBase{}, ErrProjectVersionMismatch
	}
	if base.ConfirmationString != base.CurrentName {
		return ProjectDeleteBase{}, ErrProjectDeleteConfirmationMismatch
	}

	if err := tx.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM tasks
WHERE project_id = ?;
`, trimmedProjectID).Scan(&base.AffectedTaskCount); err != nil {
		return ProjectDeleteBase{}, fmt.Errorf("load project delete base: count affected tasks: %w", err)
	}

	result, err := tx.ExecContext(ctx, `
UPDATE projects
SET server_version = server_version + 1,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ? AND server_version = ?;
`, trimmedProjectID, expectedVersion)
	if err != nil {
		return ProjectDeleteBase{}, fmt.Errorf("load project delete base: reserve version: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return ProjectDeleteBase{}, fmt.Errorf("load project delete base: read affected reservation rows: %w", err)
	}
	if affected != 1 {
		return ProjectDeleteBase{}, ErrProjectVersionMismatch
	}

	if err := tx.Commit(); err != nil {
		return ProjectDeleteBase{}, fmt.Errorf("load project delete base: commit reservation: %w", err)
	}

	base.ReservedVersion = expectedVersion + 1

	return base, nil
}

// DeleteProject removes a local project and all local tasks after successful remote calendar delete.
func (d *Database) DeleteProject(ctx context.Context, projectID string, expectedVersion int) error {
	trimmedProjectID := strings.TrimSpace(projectID)
	if trimmedProjectID == "" {
		return fmt.Errorf("delete project: project id is required")
	}
	if expectedVersion < 2 {
		return fmt.Errorf("delete project: expected version is required")
	}

	d.WriteMu.Lock()
	defer d.WriteMu.Unlock()

	tx, err := d.Conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("delete project: begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `
DELETE FROM tasks_fts
WHERE rowid IN (
    SELECT rowid FROM tasks WHERE project_id = ?
);
`, trimmedProjectID); err != nil && !strings.Contains(err.Error(), "no such table: tasks_fts") {
		return fmt.Errorf("delete project: delete fts entries: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
DELETE FROM tasks
WHERE project_id = ?;
`, trimmedProjectID); err != nil {
		return fmt.Errorf("delete project: delete tasks: %w", err)
	}

	result, err := tx.ExecContext(ctx, `
DELETE FROM projects
WHERE id = ? AND server_version = ?;
`, trimmedProjectID, expectedVersion)
	if err != nil {
		return fmt.Errorf("delete project: delete project: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete project: read affected project rows: %w", err)
	}
	if affected != 1 {
		return ErrProjectVersionMismatch
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("delete project: commit transaction: %w", err)
	}

	return nil
}

// CancelProjectDeleteReservation releases a reserved project version when remote delete fails.
func (d *Database) CancelProjectDeleteReservation(ctx context.Context, projectID string, reservedVersion int) error {
	trimmedProjectID := strings.TrimSpace(projectID)
	if trimmedProjectID == "" {
		return fmt.Errorf("cancel project delete reservation: project id is required")
	}

	if reservedVersion < 2 {
		return fmt.Errorf("cancel project delete reservation: reserved version is required")
	}

	d.WriteMu.Lock()
	defer d.WriteMu.Unlock()

	result, err := d.Conn.ExecContext(ctx, `
UPDATE projects
SET server_version = server_version - 1,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ? AND server_version = ?;
`, trimmedProjectID, reservedVersion)
	if err != nil {
		return fmt.Errorf("cancel project delete reservation: update reservation: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("cancel project delete reservation: read affected rows: %w", err)
	}
	if affected != 1 {
		return ErrProjectVersionMismatch
	}

	return nil
}
