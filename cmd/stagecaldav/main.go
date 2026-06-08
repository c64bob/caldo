package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"caldo/internal/stagecaldav"
)

const defaultAddr = "127.0.0.1:8090"

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	server, err := stagecaldav.New(stagecaldav.Config{
		Username:   getenvDefault("STAGE_CALDAV_USERNAME", "stage"),
		Password:   getenvDefault("STAGE_CALDAV_PASSWORD", "stage"),
		AdminToken: getenvDefault("STAGE_CALDAV_ADMIN_TOKEN", "stage-admin"),
	})
	if err != nil {
		logger.Error("stage_caldav_config_failed", "error", err)
		os.Exit(1)
	}

	addr := getenvDefault("STAGE_CALDAV_ADDR", defaultAddr)
	httpServer := &http.Server{
		Addr:              addr,
		Handler:           server,
		ReadHeaderTimeout: 5 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		logger.Info("stage_caldav_starting", "addr", addr, "admin_state_path", "/stage/admin/state", "admin_tasks_path", "/stage/admin/tasks")
		errCh <- httpServer.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			logger.Error("stage_caldav_shutdown_failed", "error", err)
			os.Exit(1)
		}
		logger.Info("stage_caldav_stopped")
	case err := <-errCh:
		if !errors.Is(err, http.ErrServerClosed) {
			logger.Error("stage_caldav_failed", "error", err)
			os.Exit(1)
		}
	}
}

func getenvDefault(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}
