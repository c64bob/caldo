package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"caldo/internal/db"
)

const defaultInterval = 15 * time.Minute

type fullSyncRunner interface {
	Run(ctx context.Context) error
}

// PeriodicScheduler executes periodic full-sync runs within the Go process.
type PeriodicScheduler struct {
	logger   *slog.Logger
	database *db.Database
	runner   fullSyncRunner

	mu                          sync.Mutex
	started                     bool
	cancel                      context.CancelFunc
	done                        chan struct{}
	interval                    time.Duration
	lastResolvedConflictCleanup time.Time
}

// NewPeriodicScheduler creates a periodic scheduler with an optional runner.
func NewPeriodicScheduler(logger *slog.Logger, database *db.Database, runner fullSyncRunner) *PeriodicScheduler {
	return &PeriodicScheduler{logger: logger, database: database, runner: runner, interval: defaultInterval}
}

// Start starts the scheduler loop once.
func (s *PeriodicScheduler) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.started {
		return nil
	}
	if s.interval <= 0 {
		s.interval = defaultInterval
	}
	loopCtx, cancel := context.WithCancel(ctx)
	s.cancel = cancel
	s.done = make(chan struct{})
	s.started = true
	done := s.done
	go s.run(loopCtx, done)
	return nil
}

// Stop stops the scheduler loop.
func (s *PeriodicScheduler) Stop(ctx context.Context) error {
	s.mu.Lock()
	if !s.started {
		s.mu.Unlock()
		return nil
	}
	s.cancel()
	done := s.done
	s.cancel = nil
	s.done = nil
	s.started = false
	s.mu.Unlock()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// SetInterval updates the active scheduler interval and restarts the loop.
func (s *PeriodicScheduler) SetInterval(ctx context.Context, interval time.Duration) error {
	if interval <= 0 {
		return fmt.Errorf("set interval: invalid interval")
	}
	s.mu.Lock()
	s.interval = interval
	if !s.started {
		s.mu.Unlock()
		return nil
	}
	s.cancel()
	done := s.done
	s.mu.Unlock()

	select {
	case <-done:
		s.mu.Lock()
		defer s.mu.Unlock()
		if !s.started {
			return nil
		}
		loopCtx, cancel := context.WithCancel(ctx)
		s.cancel = cancel
		s.done = make(chan struct{})
		go s.run(loopCtx, s.done)
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *PeriodicScheduler) run(ctx context.Context, done chan struct{}) {
	defer close(done)
	ticker := time.NewTicker(s.currentInterval())
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.runSyncTick(ctx)
		}
	}
}

func (s *PeriodicScheduler) currentInterval() time.Duration {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.interval
}

func (s *PeriodicScheduler) runSyncTick(ctx context.Context) {
	if s.database == nil {
		return
	}
	persistCtx := context.WithoutCancel(ctx)
	started, err := s.database.TryStartManualSync(persistCtx)
	if err != nil {
		s.logError("scheduler_sync_start_failed", err)
		return
	}
	if !started {
		return
	}
	if s.runner == nil {
		_ = s.database.FinishManualSyncError(persistCtx, "sync_unavailable")
		_ = s.runCleanup(persistCtx)
		return
	}
	if err := s.runner.Run(ctx); err != nil {
		_ = s.database.FinishManualSyncError(persistCtx, "sync_failed")
		_ = s.runCleanup(persistCtx)
		s.logError("scheduler_sync_failed", err)
		return
	}
	if err := s.runCleanup(persistCtx); err != nil {
		s.logError("scheduler_sync_cleanup_failed", err)
	}
	if err := s.database.FinishManualSyncSuccess(persistCtx); err != nil {
		s.logError("scheduler_sync_finish_failed", err)
	}
}

func (s *PeriodicScheduler) runCleanup(ctx context.Context) error {
	now := time.Now().UTC()
	cleanupResolvedConflicts := s.shouldCleanupResolvedConflicts(now)
	_, err := s.database.CleanupSyncArtifacts(ctx, now, cleanupResolvedConflicts)
	return err
}

func (s *PeriodicScheduler) shouldCleanupResolvedConflicts(now time.Time) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.lastResolvedConflictCleanup.IsZero() || now.Sub(s.lastResolvedConflictCleanup) >= 24*time.Hour {
		s.lastResolvedConflictCleanup = now
		return true
	}
	return false
}

func (s *PeriodicScheduler) logError(msg string, err error) {
	if s.logger != nil {
		s.logger.Error(msg, "error", err)
	}
}
