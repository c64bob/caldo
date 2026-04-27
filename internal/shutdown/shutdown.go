package shutdown

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const (
	// DefaultTimeout bounds graceful shutdown operations.
	DefaultTimeout = 30 * time.Second
)

// Exit codes returned by Handle for process termination.
const (
	ExitCodeSuccess = 0
	ExitCodeFailure = 1
)

// Scheduler can stop launching new jobs and wait for running work.
type Scheduler interface {
	Stop(ctx context.Context) error
}

// Coordinator orchestrates graceful process shutdown.
type Coordinator struct {
	logger        *slog.Logger
	scheduler     Scheduler
	timeout       time.Duration
	notifyContext func(context.Context, ...os.Signal) (context.Context, context.CancelFunc)
}

// NewCoordinator constructs a shutdown coordinator with default signal handling.
func NewCoordinator(logger *slog.Logger, scheduler Scheduler, timeout time.Duration) *Coordinator {
	if timeout <= 0 {
		timeout = DefaultTimeout
	}

	return &Coordinator{
		logger:        logger,
		scheduler:     scheduler,
		timeout:       timeout,
		notifyContext: signal.NotifyContext,
	}
}

// Handle waits for SIGTERM/SIGINT and runs graceful shutdown for scheduler and HTTP server.
func (c *Coordinator) Handle(ctx context.Context, server *http.Server) int {
	signalCtx, stop := c.notifyContext(ctx, syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	<-signalCtx.Done()

	c.logger.Info("shutdown_start")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	if server != nil {
		server.SetKeepAlivesEnabled(false)
		if err := server.Shutdown(shutdownCtx); err != nil {
			c.logger.Error("shutdown_http_server_failed", "error", err)
			if errors.Is(err, context.DeadlineExceeded) || errors.Is(shutdownCtx.Err(), context.DeadlineExceeded) {
				c.logger.Error("shutdown_timeout_exceeded")
				return ExitCodeFailure
			}
			return ExitCodeFailure
		}

		c.logger.Info("shutdown_http_server_stopped")
	}

	if c.scheduler != nil {
		if err := c.scheduler.Stop(shutdownCtx); err != nil {
			c.logger.Error("shutdown_scheduler_failed", "error", err)
			if errors.Is(err, context.DeadlineExceeded) || errors.Is(shutdownCtx.Err(), context.DeadlineExceeded) {
				c.logger.Error("shutdown_timeout_exceeded")
				return ExitCodeFailure
			}
			return ExitCodeFailure
		}

		c.logger.Info("shutdown_scheduler_stopped")
	}

	if errors.Is(shutdownCtx.Err(), context.DeadlineExceeded) {
		c.logger.Error("shutdown_timeout_exceeded")
		return ExitCodeFailure
	}

	c.logger.Info("shutdown_process_exit", "exit_code", ExitCodeSuccess)
	return ExitCodeSuccess
}
