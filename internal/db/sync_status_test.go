package db

import (
	"context"
	"path/filepath"
	"testing"
)

func TestManualSyncStatusLifecycle(t *testing.T) {
	database, err := OpenSQLite(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil { t.Fatalf("open sqlite: %v", err) }
	t.Cleanup(func() { _ = database.Close() })
	ctx := context.Background()
	started, err := database.TryStartManualSync(ctx)
	if err != nil || !started { t.Fatalf("start sync: %v started=%v", err, started) }
	started, err = database.TryStartManualSync(ctx)
	if err != nil || started { t.Fatalf("second start should be blocked: %v started=%v", err, started) }
	if err := database.FinishManualSyncSuccess(ctx); err != nil { t.Fatalf("finish sync: %v", err) }
	status, err := database.LoadSyncStatus(ctx)
	if err != nil { t.Fatalf("load status: %v", err) }
	if status.State != "idle" || !status.LastSuccessAt.Valid { t.Fatalf("unexpected status: %#v", status) }
}
