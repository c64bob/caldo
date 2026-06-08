package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"caldo/internal/db"
)

type stubManualSyncRunner struct{ err error }

func (s stubManualSyncRunner) Run(context.Context) error { return s.err }

type blockingManualSyncRunner struct {
	started chan struct{}
	release chan struct{}
}

func (s blockingManualSyncRunner) Run(context.Context) error {
	close(s.started)
	<-s.release
	return nil
}

func TestManualSyncHandlerStartsSync(t *testing.T) {
	database := openSQLiteForSyncHandlerTest(t)
	broker := newEventBroker()
	h := ManualSync(syncDependencies{database: database, broker: broker, runner: stubManualSyncRunner{}})

	req := httptest.NewRequest(http.MethodPost, "/sync/manual", strings.NewReader(""))
	w := httptest.NewRecorder()
	h(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("unexpected code: %d", w.Code)
	}

	waitForSyncStatus(t, database, func(status db.SyncStatus) bool {
		return status.State == "idle" && status.LastSuccessAt.Valid
	})
}

func TestManualSyncHandlerReturnsWhileRunnerIsRunning(t *testing.T) {
	database := openSQLiteForSyncHandlerTest(t)
	broker := newEventBroker()
	runner := blockingManualSyncRunner{started: make(chan struct{}), release: make(chan struct{})}
	h := ManualSync(syncDependencies{database: database, broker: broker, runner: runner})

	req := httptest.NewRequest(http.MethodPost, "/sync/manual", strings.NewReader(""))
	w := httptest.NewRecorder()
	h(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("unexpected code: %d", w.Code)
	}

	select {
	case <-runner.started:
	case <-time.After(time.Second):
		t.Fatal("runner did not start")
	}
	status, err := database.LoadSyncStatus(context.Background())
	if err != nil {
		t.Fatalf("load status: %v", err)
	}
	if status.State != "running" {
		t.Fatalf("expected running state, got %s", status.State)
	}

	close(runner.release)
	waitForSyncStatus(t, database, func(status db.SyncStatus) bool {
		return status.State == "idle" && status.LastSuccessAt.Valid
	})
}

func TestManualSyncHandlerMarksErrorWhenSyncFails(t *testing.T) {
	database := openSQLiteForSyncHandlerTest(t)
	broker := newEventBroker()
	h := ManualSync(syncDependencies{database: database, broker: broker, runner: stubManualSyncRunner{err: errors.New("boom")}})

	req := httptest.NewRequest(http.MethodPost, "/sync/manual", strings.NewReader(""))
	w := httptest.NewRecorder()
	h(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("unexpected code: %d", w.Code)
	}

	waitForSyncStatus(t, database, func(status db.SyncStatus) bool {
		return status.State == "idle" && status.LastErrorCode.Valid && status.LastErrorCode.String == "sync_failed"
	})
}

func openSQLiteForSyncHandlerTest(t *testing.T) *db.Database {
	t.Helper()
	database, err := db.OpenSQLite(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	return database
}

func waitForSyncStatus(t *testing.T, database *db.Database, match func(db.SyncStatus) bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	var last db.SyncStatus
	for time.Now().Before(deadline) {
		status, err := database.LoadSyncStatus(context.Background())
		if err != nil {
			t.Fatalf("load status: %v", err)
		}
		last = status
		if match(status) {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("sync status did not match before deadline: %#v", last)
}
