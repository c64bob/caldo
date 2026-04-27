package main

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"caldo/internal/config"
	"caldo/internal/db"
	"caldo/internal/handler"
	"caldo/internal/lock"
	"caldo/internal/logging"
)

func main() {
	logger := logging.New(os.Stderr, os.Getenv("APP_ENV"), os.Getenv("LOG_LEVEL"))
	slog.SetDefault(logger)

	if err := run(logger); err != nil {
		logStartupError(logger, err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
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

	server := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: handler.NewRouter(logger),
	}

	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("listen and serve: %w", err)
	}

	return nil
}

func logStartupError(logger *slog.Logger, err error) {
	var validationErr *config.ValidationError
	if errors.As(err, &validationErr) {
		logger.Error("startup_failed", "error_type", "config_validation", "field", validationErr.Field, "code", validationErr.Code)
		return
	}

	logger.Error("startup_failed", "error", err)
}
