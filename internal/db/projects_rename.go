package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

var (
	// ErrProjectNotFound indicates the referenced project does not exist.
	ErrProjectNotFound = errors.New("project not found")
	// ErrProjectVersionMismatch indicates optimistic-lock check failure for projects.
	ErrProjectVersionMismatch = errors.New("project version mismatch")
)

// ProjectRenameBase captures persisted project metadata required for rename operations.
type ProjectRenameBase struct {
	ProjectID       string
	CalendarHref    string
	CurrentName     string
	CurrentVersion  int
	ReservedVersion int
	RequestedName   string
	ExpectedVersion int
}

// LoadProjectRenameBase loads project metadata and reserves a version before a write-through rename.
func (d *Database) LoadProjectRenameBase(ctx context.Context, projectID string, expectedVersion int, requestedName string) (ProjectRenameBase, error) {
	trimmedProjectID := strings.TrimSpace(projectID)
	if trimmedProjectID == "" {
		return ProjectRenameBase{}, fmt.Errorf("load project rename base: project id is required")
	}

	trimmedName := strings.TrimSpace(requestedName)
	if trimmedName == "" {
		return ProjectRenameBase{}, fmt.Errorf("load project rename base: display name is required")
	}

	if expectedVersion < 1 {
		return ProjectRenameBase{}, fmt.Errorf("load project rename base: expected version is required")
	}

	base := ProjectRenameBase{ProjectID: trimmedProjectID, RequestedName: trimmedName, ExpectedVersion: expectedVersion}

	d.WriteMu.Lock()
	defer d.WriteMu.Unlock()

	tx, err := d.Conn.BeginTx(ctx, nil)
	if err != nil {
		return ProjectRenameBase{}, fmt.Errorf("load project rename base: begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := tx.QueryRowContext(ctx, `
SELECT calendar_href, display_name, server_version
FROM projects
WHERE id = ?;
`, trimmedProjectID).Scan(&base.CalendarHref, &base.CurrentName, &base.CurrentVersion); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ProjectRenameBase{}, ErrProjectNotFound
		}
		return ProjectRenameBase{}, fmt.Errorf("load project rename base: query project: %w", err)
	}

	if base.CurrentVersion != expectedVersion {
		return ProjectRenameBase{}, ErrProjectVersionMismatch
	}

	result, err := tx.ExecContext(ctx, `
UPDATE projects
SET server_version = server_version + 1,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ? AND server_version = ?;
`, trimmedProjectID, expectedVersion)
	if err != nil {
		return ProjectRenameBase{}, fmt.Errorf("load project rename base: reserve version: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return ProjectRenameBase{}, fmt.Errorf("load project rename base: read affected reservation rows: %w", err)
	}
	if affected != 1 {
		return ProjectRenameBase{}, ErrProjectVersionMismatch
	}

	if err := tx.Commit(); err != nil {
		return ProjectRenameBase{}, fmt.Errorf("load project rename base: commit reservation: %w", err)
	}

	base.ReservedVersion = expectedVersion + 1

	return base, nil
}

// RenameProject updates the local project plus denormalized task project names after successful remote rename.
func (d *Database) RenameProject(ctx context.Context, projectID string, expectedVersion int, displayName string) error {
	trimmedProjectID := strings.TrimSpace(projectID)
	if trimmedProjectID == "" {
		return fmt.Errorf("rename project: project id is required")
	}

	trimmedName := strings.TrimSpace(displayName)
	if trimmedName == "" {
		return fmt.Errorf("rename project: display name is required")
	}

	d.WriteMu.Lock()
	defer d.WriteMu.Unlock()

	tx, err := d.Conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("rename project: begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	result, err := tx.ExecContext(ctx, `
UPDATE projects
SET display_name = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ? AND server_version = ?;
`, trimmedName, trimmedProjectID, expectedVersion)
	if err != nil {
		return fmt.Errorf("rename project: update project: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rename project: read affected project rows: %w", err)
	}
	if affected != 1 {
		return ErrProjectVersionMismatch
	}

	if _, err := tx.ExecContext(ctx, `
UPDATE tasks
SET project_name = ?
WHERE project_id = ?;
`, trimmedName, trimmedProjectID); err != nil {
		return fmt.Errorf("rename project: update task project names: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("rename project: commit transaction: %w", err)
	}

	return nil
}

// CancelProjectRenameReservation releases a reserved project version when remote rename fails.
func (d *Database) CancelProjectRenameReservation(ctx context.Context, projectID string, reservedVersion int) error {
	trimmedProjectID := strings.TrimSpace(projectID)
	if trimmedProjectID == "" {
		return fmt.Errorf("cancel project rename reservation: project id is required")
	}

	if reservedVersion < 2 {
		return fmt.Errorf("cancel project rename reservation: reserved version is required")
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
		return fmt.Errorf("cancel project rename reservation: update reservation: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("cancel project rename reservation: read affected rows: %w", err)
	}
	if affected != 1 {
		return ErrProjectVersionMismatch
	}

	return nil
}
