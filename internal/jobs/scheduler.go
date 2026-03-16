package jobs

import (
	"context"
	"time"
)

type Scheduler struct {
	interval time.Duration
	job      *SyncJob
	stop     chan struct{}
}

func NewScheduler(interval time.Duration, job *SyncJob) *Scheduler {
	if interval <= 0 {
		interval = 5 * time.Minute
	}
	return &Scheduler{interval: interval, job: job, stop: make(chan struct{})}
}

func (s *Scheduler) Start(ctx context.Context) {
	if s == nil || s.job == nil {
		return
	}
	go func() {
		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-s.stop:
				return
			case <-ticker.C:
				s.job.Run(ctx)
			}
		}
	}()
}

func (s *Scheduler) Stop() {
	if s == nil {
		return
	}
	select {
	case <-s.stop:
		return
	default:
		close(s.stop)
	}
}
