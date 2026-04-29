package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"caldo/internal/assets"
	"caldo/internal/config"
	"caldo/internal/db"
	"caldo/internal/handler"
	"caldo/internal/lock"
	"caldo/internal/logging"
	"caldo/internal/scheduler"
	"caldo/internal/shutdown"
)

var errShutdownTimeout = errors.New("shutdown timeout exceeded")

func main() {
	logger := logging.New(os.Stderr, os.Getenv("APP_ENV"), os.Getenv("LOG_LEVEL"))
	slog.SetDefault(logger)

	if err := run(logger); err != nil {
		logStartupError(logger, err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	lifecycleCtx, cancelLifecycle := context.WithCancel(context.Background())
	defer cancelLifecycle()

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	manifest, err := assets.LoadManifest("web/static/manifest.json")
	if err != nil {
		return fmt.Errorf("load static manifest: %w", err)
	}

	startupLock, err := lock.AcquireStartupLock(cfg.DBPath)
	if err != nil {
		return fmt.Errorf("acquire startup lock: %w", err)
	}
	defer func() {
		_ = startupLock.Release()
	}()

	sqliteDB, err := db.OpenSQLite(cfg.DBPath)
	if err != nil {
		return fmt.Errorf("open sqlite: %w", err)
	}
	defer func() {
		_ = sqliteDB.Close()
	}()

	setupStatus, err := sqliteDB.LoadSetupStatus(context.Background())
	if err != nil {
		return fmt.Errorf("load setup status: %w", err)
	}

	appScheduler := scheduler.NewPeriodicScheduler(logger, sqliteDB, nil)
	if setupStatus.Complete {
		if err := appScheduler.Start(lifecycleCtx); err != nil {
			return fmt.Errorf("start scheduler: %w", err)
		}
	}

	server := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: handler.NewRouter(logger, cfg.ProxyUserHeader, manifest, setupStatus.Complete, cfg.EncryptionKey, sqliteDB, lifecycleCtx, appScheduler),
	}
	server.RegisterOnShutdown(cancelLifecycle)

	serverErr := make(chan error, 1)
	go func() {
		err := server.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- fmt.Errorf("listen and serve: %w", err)
			return
		}
		serverErr <- nil
	}()

	coordinator := shutdown.NewCoordinator(logger, appScheduler, shutdown.DefaultTimeout)
	shutdownCode := make(chan int, 1)
	go func() {
		shutdownCode <- coordinator.Handle(context.Background(), server)
	}()

	select {
	case err := <-serverErr:
		return err
	case code := <-shutdownCode:
		if code != shutdown.ExitCodeSuccess {
			return errShutdownTimeout
		}
		return <-serverErr
	}
}

func logStartupError(logger *slog.Logger, err error) {
	var validationErr *config.ValidationError
	if errors.As(err, &validationErr) {
		logger.Error("startup_failed", "error_type", "config_validation", "field", validationErr.Field, "code", validationErr.Code)
		return
	}

	logger.Error("startup_failed", "error", err)
}
