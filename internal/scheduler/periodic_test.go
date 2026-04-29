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
	if err := s.SetInterval(ctx, 5*time.Millisecond); err != nil {
		t.Fatalf("reset interval: %v", err)
	}
	time.Sleep(40 * time.Millisecond)
	if runner.count() <= first {
		t.Fatalf("expected more runs after interval restart")
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
