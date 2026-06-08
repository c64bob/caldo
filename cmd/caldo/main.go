package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"syscall"

	"caldo/internal/assets"
	"caldo/internal/caldav"
	"caldo/internal/config"
	"caldo/internal/db"
	"caldo/internal/handler"
	"caldo/internal/lock"
	"caldo/internal/logging"
	"caldo/internal/scheduler"
	"caldo/internal/shutdown"
	caldosync "caldo/internal/sync"
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

	startupLock, err := lock.AcquireStartupLock(cfg.DBPath)
	if err != nil {
		return fmt.Errorf("acquire startup lock: %w", err)
	}
	defer func() {
		_ = startupLock.Release()
	}()

	sqliteDB, err := db.OpenSQLiteConnection(cfg.DBPath)
	if err != nil {
		return fmt.Errorf("open sqlite: %w", err)
	}
	defer func() {
		_ = sqliteDB.Close()
	}()

	if err := sqliteDB.RunMigrations(context.Background(), cfg.DBPath); err != nil {
		return err
	}

	syncRunner, err := caldosync.NewRunner(sqliteDB, cfg.EncryptionKey, caldav.NewTodoClient(nil))
	if err != nil {
		return fmt.Errorf("initialize sync runner: %w", err)
	}
	appScheduler := scheduler.NewPeriodicScheduler(logger, sqliteDB, syncRunner)

	setupStatus, err := sqliteDB.LoadSetupStatus(context.Background())
	if err != nil {
		return fmt.Errorf("load setup status: %w", err)
	}

	if setupStatus.Complete {
		verifyCalDAVCredentials(context.Background(), logger, sqliteDB, cfg.EncryptionKey)
		if err := appScheduler.Start(lifecycleCtx); err != nil {
			return fmt.Errorf("start scheduler: %w", err)
		}
	}

	manifest, err := assets.LoadManifest("web/static/manifest.json")
	if err != nil {
		return fmt.Errorf("load static manifest: %w", err)
	}

	server := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: handler.NewRouter(logger, cfg.ProxyUserHeader, manifest, setupStatus.Complete, cfg.EncryptionKey, sqliteDB, lifecycleCtx, appScheduler, syncRunner),
	}
	server.RegisterOnShutdown(cancelLifecycle)

	coordinator := shutdown.NewCoordinator(logger, appScheduler, shutdown.DefaultTimeout)
	shutdownCode := make(chan int, 1)
	shutdownReady := make(chan struct{})
	go func() {
		shutdownCode <- coordinator.HandleReady(context.Background(), server, shutdownReady)
	}()
	<-shutdownReady

	serverErr := make(chan error, 1)
	go func() {
		err := server.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- fmt.Errorf("listen and serve: %w", err)
			return
		}
		serverErr <- nil
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

func verifyCalDAVCredentials(ctx context.Context, logger *slog.Logger, database *db.Database, encryptionKey []byte) {
	if _, err := database.LoadCalDAVCredentials(ctx, encryptionKey); err != nil {
		logger.Warn(
			"caldav_credentials_unavailable",
			"error_type", rootCauseType(err),
			"code", calDAVCredentialErrorCode(err),
		)
	}
}

func calDAVCredentialErrorCode(err error) string {
	switch {
	case errors.Is(err, db.ErrCalDAVCredentialsNotConfigured):
		return "not_configured"
	case errors.Is(err, db.ErrCalDAVCredentialsUnavailable):
		return "decrypt_failed"
	default:
		return "load_failed"
	}
}

func logStartupError(logger *slog.Logger, err error) {
	var validationErr *config.ValidationError
	if errors.As(err, &validationErr) {
		logger.Error("startup_failed", "error_type", "config_validation", "field", validationErr.Field, "code", validationErr.Code)
		return
	}

	logger.Error(
		"startup_failed",
		"error_type",
		reflect.TypeOf(err).String(),
		"root_cause_type",
		rootCauseType(err),
		"root_cause_errno",
		rootCauseErrno(err),
		"root_cause_path",
		rootCausePath(err),
	)
}

func rootCauseType(err error) string {
	types := rootCauseLeafTypes(err)
	if len(types) == 0 {
		return "<nil>"
	}

	sort.Strings(types)
	return strings.Join(types, ",")
}

func rootCauseLeafTypes(err error) []string {
	if err == nil {
		return nil
	}

	type multiUnwrapper interface {
		Unwrap() []error
	}

	if multi, ok := err.(multiUnwrapper); ok {
		children := multi.Unwrap()
		if len(children) == 0 {
			return []string{reflect.TypeOf(err).String()}
		}

		seen := make(map[string]struct{})
		types := make([]string, 0, len(children))
		for _, child := range children {
			for _, childType := range rootCauseLeafTypes(child) {
				if _, exists := seen[childType]; exists {
					continue
				}
				seen[childType] = struct{}{}
				types = append(types, childType)
			}
		}
		return types
	}

	if next := errors.Unwrap(err); next != nil {
		return rootCauseLeafTypes(next)
	}

	return []string{reflect.TypeOf(err).String()}
}

func rootCauseErrno(err error) string {
	var errno syscall.Errno
	if !errors.As(err, &errno) {
		return ""
	}

	return fmt.Sprintf("%d", int(errno))
}

func rootCausePath(err error) string {
	var pathErr *os.PathError
	if !errors.As(err, &pathErr) {
		return ""
	}

	return filepath.Clean(pathErr.Path)
}
