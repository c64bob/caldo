package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"caldo/internal/db"
)

func TestManualSyncHandlerStartsSync(t *testing.T) {
	database := openSQLiteForSyncHandlerTest(t)
	broker := newSyncEventBroker()
	h := ManualSync(syncDependencies{database: database, broker: broker})

	req := httptest.NewRequest(http.MethodPost, "/sync/manual", strings.NewReader(""))
	w := httptest.NewRecorder()
	h(w, req)
	if w.Code != http.StatusOK { t.Fatalf("unexpected code: %d", w.Code) }

	status, err := database.LoadSyncStatus(context.Background())
	if err != nil { t.Fatalf("load status: %v", err) }
	if status.State != "running" && status.State != "idle" { t.Fatalf("unexpected state: %s", status.State) }
}

func openSQLiteForSyncHandlerTest(t *testing.T) *db.Database {
	t.Helper()
	database, err := db.OpenSQLite(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil { t.Fatalf("open sqlite: %v", err) }
	t.Cleanup(func() { _ = database.Close() })
	return database
}
