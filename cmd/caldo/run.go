package main

import (
	"errors"
	"net/http"

	"caldo/internal/app"
)

func run() error {
	application, err := app.New()
	if err != nil {
		return err
	}
	defer application.DB.Close()

	if err := application.Run(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}
