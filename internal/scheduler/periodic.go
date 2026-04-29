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

	mu       sync.Mutex
	started  bool
	cancel   context.CancelFunc
	interval time.Duration
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
	s.started = true
	go s.run(loopCtx)
	return nil
}

// Stop stops the scheduler loop.
func (s *PeriodicScheduler) Stop(ctx context.Context) error {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.started {
		return nil
	}
	s.cancel()
	s.cancel = nil
	s.started = false
	return nil
}

// SetInterval updates the active scheduler interval and restarts the loop.
func (s *PeriodicScheduler) SetInterval(ctx context.Context, interval time.Duration) error {
	if interval <= 0 {
		return fmt.Errorf("set interval: invalid interval")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.interval = interval
	if !s.started {
		return nil
	}
	s.cancel()
	loopCtx, cancel := context.WithCancel(ctx)
	s.cancel = cancel
	go s.run(loopCtx)
	return nil
}

func (s *PeriodicScheduler) run(ctx context.Context) {
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
	started, err := s.database.TryStartManualSync(ctx)
	if err != nil {
		s.logError("scheduler_sync_start_failed", err)
		return
	}
	if !started {
		return
	}
	if s.runner == nil {
		_ = s.database.FinishManualSyncError(ctx, "sync_unavailable")
		return
	}
	if err := s.runner.Run(ctx); err != nil {
		_ = s.database.FinishManualSyncError(ctx, "sync_failed")
		s.logError("scheduler_sync_failed", err)
		return
	}
	if err := s.database.FinishManualSyncSuccess(ctx); err != nil {
		s.logError("scheduler_sync_finish_failed", err)
	}
}

func (s *PeriodicScheduler) logError(msg string, err error) {
	if s.logger != nil {
		s.logger.Error(msg, "error", err)
	}
}
