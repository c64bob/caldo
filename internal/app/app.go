package app

import (
	"fmt"
	"log"
	"net/http"

	"caldo/internal/security"
	"caldo/internal/service"
	storesqlite "caldo/internal/store/sqlite"
)

type App struct {
	Config Config
	DB     *storesqlite.DB
	Server *http.Server
}

func New() (*App, error) {
	cfg, err := LoadConfig()
	if err != nil {
		return nil, err
	}
	key, err := security.LoadMasterKey(cfg.Security.EncryptionKeyFile)
	if err != nil {
		return nil, err
	}
	db, err := storesqlite.Open(cfg.Database.Path)
	if err != nil {
		return nil, err
	}
	repo := storesqlite.NewDAVAccountsRepo(db)
	settingsSvc := service.NewSettingsService(repo, key)
	router := NewRouter(cfg, settingsSvc)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.Port),
		Handler: router,
	}

	return &App{Config: cfg, DB: db, Server: server}, nil
}

func (a *App) Run() error {
	log.Printf("caldo listening on %s", a.Server.Addr)
	return a.Server.ListenAndServe()
}
