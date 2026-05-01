package main

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func TestRootCauseErrno(t *testing.T) {
	t.Parallel()

	err := errors.New("outer")
	if got := rootCauseErrno(err); got != "" {
		t.Fatalf("expected empty errno, got %q", got)
	}
}

func TestRootCauseErrnoFromWrappedPathError(t *testing.T) {
	t.Parallel()

	wrapped := &os.PathError{
		Op:   "open",
		Path: "/tmp/state/caldo.db.startup.lock",
		Err:  syscall.ENOENT,
	}

	if got := rootCauseErrno(wrapped); got == "" {
		t.Fatal("expected errno to be extracted")
	}
}

func TestRootCausePath(t *testing.T) {
	t.Parallel()

	wrapped := &os.PathError{
		Op:   "open",
		Path: filepath.Join("/tmp", "state", "caldo.db.startup.lock"),
		Err:  syscall.EPERM,
	}

	got := rootCausePath(wrapped)
	if got == "" {
		t.Fatal("expected root cause path to be extracted")
	}
	if got != "/tmp/state/caldo.db.startup.lock" {
		t.Fatalf("unexpected root cause path: %q", got)
	}
}
