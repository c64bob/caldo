package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func TestRunStartupSourceOrderMatchesArchitecture(t *testing.T) {
	t.Parallel()

	sourceBytes, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatalf("read main.go: %v", err)
	}
	source := string(sourceBytes)

	assertSourceOrder(t, source,
		"config.Load()",
		"lock.AcquireStartupLock",
		"db.OpenSQLiteConnection",
		"sqliteDB.RunMigrations",
		"scheduler.NewPeriodicScheduler",
		"sqliteDB.LoadSetupStatus",
		"verifyCalDAVCredentials",
		"appScheduler.Start",
		"assets.LoadManifest",
		"shutdown.NewCoordinator",
		"coordinator.HandleReady",
		"server.ListenAndServe",
	)
}

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

func assertSourceOrder(t *testing.T, source string, markers ...string) {
	t.Helper()

	lastIndex := -1
	for _, marker := range markers {
		index := strings.Index(source, marker)
		if index == -1 {
			t.Fatalf("source missing startup marker %q", marker)
		}
		if index <= lastIndex {
			t.Fatalf("startup marker %q is out of order", marker)
		}
		lastIndex = index
	}
}
