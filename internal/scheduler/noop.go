package scheduler

import "context"

// NoopScheduler provides a concrete scheduler implementation used until sync scheduling is implemented.
type NoopScheduler struct{}

// NewNoopScheduler returns a scheduler that accepts start/stop calls without side effects.
func NewNoopScheduler() *NoopScheduler {
	return &NoopScheduler{}
}

// Start marks scheduler startup as successful.
func (s *NoopScheduler) Start(ctx context.Context) error {
	_ = ctx
	return nil
}

// Stop marks scheduler shutdown as successful.
func (s *NoopScheduler) Stop(ctx context.Context) error {
	_ = ctx
	return nil
}
