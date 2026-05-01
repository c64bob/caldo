package handler

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"caldo/internal/assets"
	"caldo/internal/caldav"
	"caldo/internal/db"
	"caldo/internal/view"
	"github.com/go-chi/chi/v5"
)

const staticAssetsCacheControl = "public, max-age=31536000, immutable"

var staticAssetsRoot = defaultStaticAssetsRoot()

// NewRouter returns the HTTP router for Caldo.
func NewRouter(logger *slog.Logger, proxyUserHeader string, manifest assets.Manifest, setupComplete bool, csrfSecret []byte, database *db.Database, lifecycleCtx context.Context, scheduler SetupSchedulerStarter) http.Handler {
	router := chi.NewRouter()
	syncBroker := newEventBroker()
	syncDeps := syncDependencies{database: database, broker: syncBroker}
	setupState := NewSetupState(setupComplete)
	router.Use(RequestIDMiddleware())
	router.Use(RecoveryMiddleware(logger))
	router.Use(SafeLoggingMiddleware(logger))
	router.Use(SecurityHeadersMiddleware())
	router.Use(ReverseProxyAuthMiddleware(proxyUserHeader))
	router.Use(SetupGateMiddleware(setupState))
	router.Use(AssetManifestMiddleware(manifest))

	router.Get("/health", Health)
	router.Get("/", Home)
	router.Get("/today", Today(dateViewDependencies{database: database}))
	router.Get("/upcoming", Upcoming(dateViewDependencies{database: database}))
	router.Get("/overdue", Overdue(dateViewDependencies{database: database}))
	router.Get("/favorites", Favorites(dateViewDependencies{database: database}))
	router.Get("/no-date", NoDate(dateViewDependencies{database: database}))
	router.Get("/completed", Completed(dateViewDependencies{database: database}))
	router.Get("/search", Search(searchDependencies{database: database}))
	router.Get("/projects", ProjectsPage())
	router.Get("/labels", LabelsPage())
	router.Get("/filters", FiltersPage())
	router.Get("/settings", SettingsPage())
	router.Get("/quick-add", QuickAddPage(quickAddDependencies{database: database}))
	router.Post("/quick-add/preview", QuickAddPreview(quickAddDependencies{database: database}))
	router.Get("/conflicts", Conflicts(conflictDependencies{database: database}))
	router.Get("/conflicts/{conflictID}", ConflictDetail(conflictDependencies{database: database}))
	router.With(SetupCSRFMiddleware(csrfSecret)).Post("/conflicts/{conflictID}/resolve", ResolveConflict(taskUpdateDependencies{
		database:      database,
		encryptionKey: csrfSecret,
		todos:         caldav.NewTodoClient(nil),
		broker:        syncBroker,
	}))
	router.Get("/events", Events(syncDeps))
	router.Get("/api/tasks/versions", TaskVersions(taskVersionsDependencies{database: database}))
	router.Get("/sync/status", SyncStatus(syncDeps))
	router.Handle("/static/*", staticFileServer(staticAssetsRoot))

	router.Route("/tasks", func(taskRouter chi.Router) {
		taskRouter.Use(SetupCSRFMiddleware(csrfSecret))
		taskRouter.Post("/", TaskCreate(taskCreateDependencies{
			database:      database,
			encryptionKey: csrfSecret,
			todos:         caldav.NewTodoClient(nil),
			broker:        syncBroker,
		}))
		taskRouter.Post("/{taskID}/subtasks", TaskCreateSubtask(taskCreateDependencies{
			database:      database,
			encryptionKey: csrfSecret,
			todos:         caldav.NewTodoClient(nil),
			broker:        syncBroker,
		}))
		taskRouter.Patch("/{taskID}", TaskUpdate(taskUpdateDependencies{
			database:      database,
			encryptionKey: csrfSecret,
			todos:         caldav.NewTodoClient(nil),
			broker:        syncBroker,
		}))
		taskRouter.Post("/{taskID}/complete", TaskComplete(taskUpdateDependencies{
			database:      database,
			encryptionKey: csrfSecret,
			todos:         caldav.NewTodoClient(nil),
			broker:        syncBroker,
		}))
		taskRouter.Post("/{taskID}/reopen", TaskReopen(taskUpdateDependencies{
			database:      database,
			encryptionKey: csrfSecret,
			todos:         caldav.NewTodoClient(nil),
			broker:        syncBroker,
		}))
		taskRouter.Post("/{taskID}/favorite", TaskFavorite(taskUpdateDependencies{
			database:      database,
			encryptionKey: csrfSecret,
			todos:         caldav.NewTodoClient(nil),
			broker:        syncBroker,
		}))
		taskRouter.Delete("/{taskID}", TaskDelete(taskUpdateDependencies{
			database:      database,
			encryptionKey: csrfSecret,
			todos:         caldav.NewTodoClient(nil),
			broker:        syncBroker,
		}))
		taskRouter.Post("/undo", TaskUndo(taskUpdateDependencies{
			database:      database,
			encryptionKey: csrfSecret,
			todos:         caldav.NewTodoClient(nil),
			broker:        syncBroker,
		}))
	})

	router.Route("/projects", func(projectRouter chi.Router) {
		projectRouter.Use(SetupCSRFMiddleware(csrfSecret))
		projectRouter.Post("/", ProjectCreate(projectCreateDependencies{
			database:      database,
			encryptionKey: csrfSecret,
			calendar:      caldav.NewCalendarClient(nil),
			broker:        syncBroker,
		}))
		projectRouter.Patch("/{projectID}", ProjectRename(projectRenameDependencies{
			database:      database,
			encryptionKey: csrfSecret,
			calendar:      caldav.NewCalendarClient(nil),
			broker:        syncBroker,
		}))
		projectRouter.Delete("/{projectID}", ProjectDelete(projectDeleteDependencies{
			database:      database,
			encryptionKey: csrfSecret,
			calendar:      caldav.NewCalendarClient(nil),
			broker:        syncBroker,
		}))
	})

	router.Route("/sync", func(syncRouter chi.Router) {
		syncRouter.Use(SetupCSRFMiddleware(csrfSecret))
		syncRouter.Post("/manual", ManualSync(syncDeps))
	})

	router.Route("/setup", func(setupRouter chi.Router) {
		setupRouter.Use(SetupCSRFMiddleware(csrfSecret))
		setupRouter.Get("/", SetupPage)
		importBroker := newSetupImportEventBroker()
		setupDeps := setupDependencies{
			database:      database,
			encryptionKey: csrfSecret,
			tester:        caldav.NewConnectionTester(nil),
			calendar:      caldav.NewCalendarClient(nil),
			todos:         caldav.NewTodoClient(nil),
			scheduler:     scheduler,
			setupState:    setupState,
			logger:        logger,
			importBroker:  importBroker,
			lifecycleCtx:  lifecycleCtx,
		}
		setupRouter.Post("/caldav", SetupCalDAV(setupDeps))
		setupRouter.Get("/calendars", SetupCalendarsPage(setupDeps))
		setupRouter.Post("/calendars", SetupCalendars(setupDeps))
		setupRouter.Post("/import", SetupImport(setupDeps))
		setupRouter.Get("/import/events", SetupImportEvents(setupDeps))
		setupRouter.Post("/complete", SetupComplete(setupDeps))
	})

	return router
}

func staticFileServer(root string) http.Handler {
	fileServer := http.FileServer(http.Dir(root))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", staticAssetsCacheControl)
		http.StripPrefix("/static", fileServer).ServeHTTP(w, r)
	})
}

func defaultStaticAssetsRoot() string {
	executablePath, err := os.Executable()
	if err == nil {
		executableDir := filepath.Dir(executablePath)
		candidate := filepath.Clean(filepath.Join(executableDir, "web", "static"))
		if directoryExists(candidate) {
			return candidate
		}
	}

	workingDir, err := os.Getwd()
	if err == nil {
		currentDir := workingDir
		for range 8 {
			candidate := filepath.Clean(filepath.Join(currentDir, "web", "static"))
			if directoryExists(candidate) {
				return candidate
			}

			parentDir := filepath.Dir(currentDir)
			if parentDir == currentDir {
				break
			}
			currentDir = parentDir
		}
	}

	return filepath.Clean(filepath.Join("web", "static"))
}

func directoryExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	return info.IsDir()
}

// AssetManifestMiddleware injects static asset resolution data and CSRF token into request context.
func AssetManifestMiddleware(manifest assets.Manifest) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := view.WithAssetManifest(r.Context(), manifest)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
