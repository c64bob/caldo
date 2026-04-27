package lock

import (
	"errors"
	"fmt"
	"os"
	"syscall"
)

// ErrNotAcquired indicates that the startup lock cannot be acquired.
var ErrNotAcquired = errors.New("startup lock not acquired")

// StartupLock keeps the process-level startup lock alive until released.
type StartupLock struct {
	file *os.File
	path string
}

// AcquireStartupLock acquires an exclusive non-blocking startup lock for a database path.
func AcquireStartupLock(dbPath string) (*StartupLock, error) {
	lockPath := dbPath + ".startup.lock"

	file, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open startup lock file: %w", err)
	}

	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		closeErr := file.Close()
		if closeErr != nil {
			return nil, fmt.Errorf("acquire startup lock: %w", errors.Join(err, closeErr))
		}

		if errors.Is(err, syscall.EWOULDBLOCK) {
			return nil, fmt.Errorf("acquire startup lock: %w", ErrNotAcquired)
		}

		return nil, fmt.Errorf("acquire startup lock: %w", err)
	}

	return &StartupLock{file: file, path: lockPath}, nil
}

// Path returns the lock file path.
func (l *StartupLock) Path() string {
	if l == nil {
		return ""
	}

	return l.path
}

// Release unlocks and closes the startup lock.
func (l *StartupLock) Release() error {
	if l == nil || l.file == nil {
		return nil
	}

	fd := int(l.file.Fd())
	if err := syscall.Flock(fd, syscall.LOCK_UN); err != nil {
		return fmt.Errorf("unlock startup lock: %w", err)
	}

	if err := l.file.Close(); err != nil {
		return fmt.Errorf("close startup lock file: %w", err)
	}

	l.file = nil
	return nil
}
