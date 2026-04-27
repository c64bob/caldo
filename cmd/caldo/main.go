package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"caldo/internal/config"
	"caldo/internal/handler"
)

func main() {
	if err := run(); err != nil {
		slog.Error("startup failed", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	server := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: handler.NewRouter(),
	}

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("listen and serve: %w", err)
	}

	return nil
}
