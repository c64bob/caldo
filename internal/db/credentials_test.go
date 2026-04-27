package db

import (
	"bytes"
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

func TestSaveCalDAVCredentialsEncryptsPassword(t *testing.T) {
	t.Parallel()

	database := openSQLiteForCredentialsTest(t)
	t.Cleanup(func() {
		_ = database.Close()
	})

	ctx := context.Background()
	key := bytes.Repeat([]byte{0x7A}, 32)
	credentials := CalDAVCredentials{URL: "https://dav.example", Username: "alice", Password: "secret-password"}

	if err := database.SaveCalDAVCredentials(ctx, key, credentials); err != nil {
		t.Fatalf("save credentials: %v", err)
	}

	var stored string
	if err := database.Conn.QueryRowContext(ctx, `SELECT caldav_password_enc FROM settings WHERE id = 'default';`).Scan(&stored); err != nil {
		t.Fatalf("query encrypted password: %v", err)
	}

	if strings.Contains(stored, credentials.Password) {
		t.Fatalf("password persisted in clear text: %q", stored)
	}

	parts := strings.Split(stored, ":")
	if len(parts) != 3 {
		t.Fatalf("unexpected encrypted payload format: %q", stored)
	}
	if parts[0] != "v1" {
		t.Fatalf("unexpected payload version: got %q want %q", parts[0], "v1")
	}
}

func TestLoadCalDAVCredentialsRoundtrip(t *testing.T) {
	t.Parallel()

	database := openSQLiteForCredentialsTest(t)
	t.Cleanup(func() {
		_ = database.Close()
	})

	ctx := context.Background()
	key := bytes.Repeat([]byte{0x33}, 32)
	want := CalDAVCredentials{URL: "https://dav.example", Username: "alice", Password: "secret-password"}

	if err := database.SaveCalDAVCredentials(ctx, key, want); err != nil {
		t.Fatalf("save credentials: %v", err)
	}

	got, err := database.LoadCalDAVCredentials(ctx, key)
	if err != nil {
		t.Fatalf("load credentials: %v", err)
	}
	if got != want {
		t.Fatalf("credentials mismatch: got %#v want %#v", got, want)
	}
}

func TestLoadCalDAVCredentialsWithWrongKeyReturnsUnavailable(t *testing.T) {
	t.Parallel()

	database := openSQLiteForCredentialsTest(t)
	t.Cleanup(func() {
		_ = database.Close()
	})

	ctx := context.Background()
	goodKey := bytes.Repeat([]byte{0x44}, 32)
	wrongKey := bytes.Repeat([]byte{0x55}, 32)

	if err := database.SaveCalDAVCredentials(ctx, goodKey, CalDAVCredentials{URL: "https://dav.example", Username: "alice", Password: "secret-password"}); err != nil {
		t.Fatalf("save credentials: %v", err)
	}

	_, err := database.LoadCalDAVCredentials(ctx, wrongKey)
	if !errors.Is(err, ErrCalDAVCredentialsUnavailable) {
		t.Fatalf("expected unavailable error, got %v", err)
	}
}

func TestLoadCalDAVCredentialsWithoutStoredValues(t *testing.T) {
	t.Parallel()

	database := openSQLiteForCredentialsTest(t)
	t.Cleanup(func() {
		_ = database.Close()
	})

	_, err := database.LoadCalDAVCredentials(context.Background(), bytes.Repeat([]byte{0x66}, 32))
	if !errors.Is(err, ErrCalDAVCredentialsNotConfigured) {
		t.Fatalf("expected not configured error, got %v", err)
	}
}

func openSQLiteForCredentialsTest(t *testing.T) *Database {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "caldo.db")
	database, err := OpenSQLite(dbPath)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	return database
}
