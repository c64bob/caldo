package scheduler

import (
	"context"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"caldo/internal/db"
)

type stubRunner struct {
	mu            sync.Mutex
	runs          int
	block         chan struct{}
	ignoreContext bool
}

func TestPeriodicSchedulerRunSyncTickCleansUpArtifacts(t *testing.T) {
	database, err := db.OpenSQLite(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer database.Close()

	now := time.Now().UTC()
	if _, err := database.Conn.ExecContext(context.Background(), `
INSERT INTO projects (id, calendar_href, display_name, sync_strategy, created_at, updated_at)
VALUES ('project-1', '/cal/p1/', 'Project 1', 'fullscan', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
INSERT INTO undo_snapshots (id, session_id, tab_id, task_id, action_type, snapshot_vtodo, snapshot_fields, created_at, expires_at)
VALUES ('undo-expired', 's1', 't1', 'task-1', 'task_updated', 'BEGIN:VTODO\nEND:VTODO', '{}', CURRENT_TIMESTAMP, ?);
INSERT INTO conflicts (id, task_id, project_id, conflict_type, created_at, resolved_at)
VALUES ('conflict-old', 'task-1', 'project-1', 'field_conflict', CURRENT_TIMESTAMP, DATETIME(CURRENT_TIMESTAMP, '-8 days'));
`, now.Add(-time.Minute)); err != nil {
		t.Fatalf("seed cleanup data: %v", err)
	}

	runner := &stubRunner{}
	s := NewPeriodicScheduler(nil, database, runner)
	s.lastResolvedConflictCleanup = now.Add(-25 * time.Hour)
	s.runSyncTick(context.Background())

	var count int
	if err := database.Conn.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM undo_snapshots`).Scan(&count); err != nil {
		t.Fatalf("count undo snapshots: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected expired undo snapshots deleted, got %d", count)
	}
	if err := database.Conn.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM conflicts WHERE id = 'conflict-old'`).Scan(&count); err != nil {
		t.Fatalf("count conflicts: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected old resolved conflict deleted, got %d", count)
	}
}

func (s *stubRunner) Run(ctx context.Context) error {
	s.mu.Lock()
	s.runs++
	s.mu.Unlock()
	if s.block != nil {
		if s.ignoreContext {
			<-s.block
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-s.block:
		}
	}
	return nil
}

func (s *stubRunner) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.runs
}

func TestPeriodicSchedulerRunsAndRestartsInterval(t *testing.T) {
	database, err := db.OpenSQLite(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer database.Close()
	runner := &stubRunner{}
	s := NewPeriodicScheduler(nil, database, runner)
	if err := s.SetInterval(context.Background(), 10*time.Millisecond); err != nil {
		t.Fatalf("set interval: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := s.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	time.Sleep(35 * time.Millisecond)
	first := runner.count()
	if first < 1 {
		t.Fatalf("expected at least one run, got %d", first)
	}
	if err := s.SetInterval(context.Background(), 5*time.Millisecond); err != nil {
		t.Fatalf("reset interval: %v", err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for runner.count() <= first && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if runner.count() <= first {
		t.Fatalf("expected more runs after interval restart; first=%d current=%d", first, runner.count())
	}
	if err := s.Stop(context.Background()); err != nil {
		t.Fatalf("stop: %v", err)
	}
}

func TestPeriodicSchedulerSkipsWhenSyncAlreadyRunning(t *testing.T) {
	database, err := db.OpenSQLite(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer database.Close()
	if _, err := database.TryStartManualSync(context.Background()); err != nil {
		t.Fatalf("seed running state: %v", err)
	}
	runner := &stubRunner{}
	s := NewPeriodicScheduler(nil, database, runner)
	if err := s.SetInterval(context.Background(), 5*time.Millisecond); err != nil {
		t.Fatalf("set interval: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := s.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	time.Sleep(25 * time.Millisecond)
	if err := s.Stop(context.Background()); err != nil {
		t.Fatalf("stop: %v", err)
	}
	if runner.count() != 0 {
		t.Fatalf("expected zero runs while sync is running, got %d", runner.count())
	}
}

func TestPeriodicSchedulerStopWaitsForInFlightRun(t *testing.T) {
	database, err := db.OpenSQLite(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer database.Close()

	runner := &stubRunner{block: make(chan struct{}), ignoreContext: true}
	s := NewPeriodicScheduler(nil, database, runner)
	if err := s.SetInterval(context.Background(), 5*time.Millisecond); err != nil {
		t.Fatalf("set interval: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := s.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	deadline := time.Now().Add(100 * time.Millisecond)
	for runner.count() == 0 && time.Now().Before(deadline) {
		time.Sleep(2 * time.Millisecond)
	}
	if runner.count() == 0 {
		t.Fatal("expected scheduler to begin one run")
	}

	stopDone := make(chan error, 1)
	go func() {
		stopDone <- s.Stop(context.Background())
	}()

	select {
	case err := <-stopDone:
		t.Fatalf("stop returned before in-flight run finished: %v", err)
	case <-time.After(20 * time.Millisecond):
	}

	close(runner.block)

	select {
	case err := <-stopDone:
		if err != nil {
			t.Fatalf("stop: %v", err)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("stop did not return after in-flight run finished")
	}
}
