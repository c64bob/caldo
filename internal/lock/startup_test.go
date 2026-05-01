package lock

import (
	"errors"
	"path/filepath"
	"testing"
)

func TestAcquireStartupLock(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "caldo.db")

	firstLock, err := AcquireStartupLock(dbPath)
	if err != nil {
		t.Fatalf("acquire first startup lock: %v", err)
	}
	t.Cleanup(func() {
		if err := firstLock.Release(); err != nil {
			t.Fatalf("release first startup lock: %v", err)
		}
	})

	if firstLock.Path() != dbPath+".startup.lock" {
		t.Fatalf("unexpected lock path: %s", firstLock.Path())
	}

	secondLock, err := AcquireStartupLock(dbPath)
	if err == nil {
		_ = secondLock.Release()
		t.Fatal("expected startup lock acquisition to fail while already held")
	}

	if !errors.Is(err, ErrNotAcquired) {
		t.Fatalf("expected ErrNotAcquired, got: %v", err)
	}
}

func TestReleaseStartupLockAllowsReacquire(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "caldo.db")

	lock, err := AcquireStartupLock(dbPath)
	if err != nil {
		t.Fatalf("acquire startup lock: %v", err)
	}

	if err := lock.Release(); err != nil {
		t.Fatalf("release startup lock: %v", err)
	}

	reacquired, err := AcquireStartupLock(dbPath)
	if err != nil {
		t.Fatalf("reacquire startup lock: %v", err)
	}
	defer func() {
		if err := reacquired.Release(); err != nil {
			t.Fatalf("release reacquired startup lock: %v", err)
		}
	}()
}

func TestReleaseNilStartupLock(t *testing.T) {
	t.Parallel()

	var lock *StartupLock
	if err := lock.Release(); err != nil {
		t.Fatalf("expected nil startup lock release to be no-op, got: %v", err)
	}
}

func TestAcquireStartupLockCreatesMissingDirectory(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "nested", "state", "caldo.db")

	lock, err := AcquireStartupLock(dbPath)
	if err != nil {
		t.Fatalf("acquire startup lock for nested path: %v", err)
	}
	defer func() {
		if err := lock.Release(); err != nil {
			t.Fatalf("release startup lock for nested path: %v", err)
		}
	}()

	if lock.Path() != dbPath+".startup.lock" {
		t.Fatalf("unexpected lock path: %s", lock.Path())
	}
}
