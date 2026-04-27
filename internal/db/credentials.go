package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	appcrypto "caldo/internal/crypto"
)

var (
	// ErrCalDAVCredentialsUnavailable indicates that persisted credentials cannot be decrypted with the current key.
	ErrCalDAVCredentialsUnavailable = errors.New("caldav credentials unavailable")
	// ErrCalDAVCredentialsNotConfigured indicates that credentials have not been persisted yet.
	ErrCalDAVCredentialsNotConfigured = errors.New("caldav credentials not configured")
)

// CalDAVCredentials contains credentials needed to authenticate with a CalDAV server.
type CalDAVCredentials struct {
	URL      string
	Username string
	Password string
}

// SaveCalDAVCredentials encrypts and persists CalDAV credentials in the settings singleton.
func (d *Database) SaveCalDAVCredentials(ctx context.Context, key []byte, credentials CalDAVCredentials) error {
	encryptedPassword, err := appcrypto.EncryptCredential(key, credentials.Password)
	if err != nil {
		return fmt.Errorf("encrypt caldav password: %w", err)
	}

	d.WriteMu.Lock()
	defer d.WriteMu.Unlock()

	result, err := d.Conn.ExecContext(ctx, `
UPDATE settings
SET caldav_url = ?,
    caldav_username = ?,
    caldav_password_enc = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = 'default';
`, credentials.URL, credentials.Username, encryptedPassword)
	if err != nil {
		return fmt.Errorf("update caldav credentials: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read affected rows: %w", err)
	}
	if affected != 1 {
		return fmt.Errorf("update caldav credentials: expected 1 row affected, got %d", affected)
	}

	return nil
}

// LoadCalDAVCredentials loads and decrypts CalDAV credentials from the settings singleton.
func (d *Database) LoadCalDAVCredentials(ctx context.Context, key []byte) (CalDAVCredentials, error) {
	var url, username, encryptedPassword sql.NullString
	if err := d.Conn.QueryRowContext(ctx, `
SELECT caldav_url, caldav_username, caldav_password_enc
FROM settings
WHERE id = 'default';
`).Scan(&url, &username, &encryptedPassword); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return CalDAVCredentials{}, ErrCalDAVCredentialsNotConfigured
		}
		return CalDAVCredentials{}, fmt.Errorf("query caldav credentials: %w", err)
	}

	if !url.Valid || !username.Valid || !encryptedPassword.Valid {
		return CalDAVCredentials{}, ErrCalDAVCredentialsNotConfigured
	}

	password, err := appcrypto.DecryptCredential(key, encryptedPassword.String)
	if err != nil {
		return CalDAVCredentials{}, ErrCalDAVCredentialsUnavailable
	}

	return CalDAVCredentials{
		URL:      url.String,
		Username: username.String,
		Password: password,
	}, nil
}
