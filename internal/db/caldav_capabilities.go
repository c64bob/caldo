package db

import (
	"context"
	"encoding/json"
	"fmt"
)

// CalDAVServerCapabilities stores globally detected account/server capabilities.
type CalDAVServerCapabilities struct {
	WebDAVSync bool `json:"webdav_sync"`
	CTag       bool `json:"ctag"`
	ETag       bool `json:"etag"`
	FullScan   bool `json:"fullscan"`
}

// SaveCalDAVServerCapabilities persists detected CalDAV server capabilities in the settings singleton.
func (d *Database) SaveCalDAVServerCapabilities(ctx context.Context, capabilities CalDAVServerCapabilities) error {
	payload, err := json.Marshal(capabilities)
	if err != nil {
		return fmt.Errorf("marshal caldav server capabilities: %w", err)
	}

	d.WriteMu.Lock()
	defer d.WriteMu.Unlock()

	result, err := d.Conn.ExecContext(ctx, `
UPDATE settings
SET caldav_server_capabilities = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = 'default';
`, string(payload))
	if err != nil {
		return fmt.Errorf("update caldav server capabilities: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read affected rows: %w", err)
	}
	if affected != 1 {
		return fmt.Errorf("update caldav server capabilities: expected 1 row affected, got %d", affected)
	}

	return nil
}
