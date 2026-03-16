package main

import (
	"errors"
	"log"
	"net/http"

	"caldo/internal/app"
)

func main() {
	application, err := app.New()
	if err != nil {
		log.Fatalf("startup failed: %v", err)
	}
	defer application.DB.Close()

	if err := application.Run(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("server failed: %v", err)
	}
}
