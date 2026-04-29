package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"caldo/internal/db"
)

type stubManualSyncRunner struct{ err error }

func (s stubManualSyncRunner) Run(context.Context) error { return s.err }

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

	status, err := database.LoadSyncStatus(context.Background())
	if err != nil {
		t.Fatalf("load status: %v", err)
	}
	if status.State != "running" && status.State != "idle" {
		t.Fatalf("unexpected state: %s", status.State)
	}
	if !status.LastSuccessAt.Valid {
		t.Fatalf("expected last success to be set")
	}
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

	status, err := database.LoadSyncStatus(context.Background())
	if err != nil {
		t.Fatalf("load status: %v", err)
	}
	if status.State != "idle" {
		t.Fatalf("expected idle state, got %s", status.State)
	}
	if !status.LastErrorCode.Valid || status.LastErrorCode.String != "sync_failed" {
		t.Fatalf("unexpected error code: %v", status.LastErrorCode)
	}
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
