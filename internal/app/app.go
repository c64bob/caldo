package app

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"caldo/internal/http/render"
	"caldo/internal/jobs"
	"caldo/internal/security"
	"caldo/internal/service"
	storesqlite "caldo/internal/store/sqlite"
)

type App struct {
	Config    Config
	DB        *storesqlite.DB
	Server    *http.Server
	scheduler *jobs.Scheduler
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
	syncRepo := storesqlite.NewSyncStateRepo(db)
	settingsSvc := service.NewSettingsService(repo, key, cfg.CalDAV.ServerURL)
	taskSvc := service.NewTaskService(repo, key, cfg.CalDAV.DefaultList)
	syncSvc := service.NewSyncService(repo, syncRepo, key, cfg.CalDAV.DefaultList)
	templates, err := render.LoadTemplates()
	if err != nil {
		return nil, err
	}
	router := NewRouter(cfg, settingsSvc, taskSvc, syncSvc, templates)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.Port),
		Handler: router,
	}

	var scheduler *jobs.Scheduler
	if cfg.Sync.Enabled && strings.TrimSpace(cfg.Sync.DefaultPrincipal) != "" {
		job := &jobs.SyncJob{Service: syncSvc, PrincipalID: strings.TrimSpace(cfg.Sync.DefaultPrincipal)}
		scheduler = jobs.NewScheduler(time.Duration(cfg.Sync.IntervalSeconds)*time.Second, job)
	}

	return &App{Config: cfg, DB: db, Server: server, scheduler: scheduler}, nil
}

func (a *App) Run() error {
	if a.scheduler != nil {
		a.scheduler.Start(context.Background())
		defer a.scheduler.Stop()
	}
	log.Printf("caldo listening on %s", a.Server.Addr)
	return a.Server.ListenAndServe()
}
