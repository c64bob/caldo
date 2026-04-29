package sync

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

const (
	StrategyWebDAVSync = "webdav_sync"
	StrategyCTag       = "ctag"
	StrategyFullScan   = "fullscan"
)

// ErrFallbackRequired indicates that the current strategy cannot be used for this project.
var ErrFallbackRequired = errors.New("sync strategy fallback required")

// ProjectState carries per-project sync state needed by strategy runners.
type ProjectState struct {
	ID           string
	CalendarHref string
	SyncStrategy string
	SyncToken    string
	CTag         string
}

// ProjectStore abstracts project sync metadata persistence.
type ProjectStore interface {
	ListSyncProjects(ctx context.Context) ([]ProjectState, error)
	UpdateProjectSyncStrategy(ctx context.Context, projectID string, strategy string) error
}

// StrategyRunner executes one strategy for one project.
type StrategyRunner interface {
	Run(ctx context.Context, project ProjectState) error
}

// Engine runs project sync with fallback from webdav_sync -> ctag -> fullscan.
type Engine struct {
	store      ProjectStore
	webdavSync StrategyRunner
	ctag       StrategyRunner
	fullscan   StrategyRunner
}

// NewEngine constructs a sync engine with required dependencies.
func NewEngine(store ProjectStore, webdavSync StrategyRunner, ctag StrategyRunner, fullscan StrategyRunner) (*Engine, error) {
	if store == nil || webdavSync == nil || ctag == nil || fullscan == nil {
		return nil, fmt.Errorf("new sync engine: dependencies are required")
	}
	return &Engine{store: store, webdavSync: webdavSync, ctag: ctag, fullscan: fullscan}, nil
}

// Run executes one sync pass for all projects, applying per-project strategy fallback.
func (e *Engine) Run(ctx context.Context) error {
	projects, err := e.store.ListSyncProjects(ctx)
	if err != nil {
		return fmt.Errorf("sync run: list projects: %w", err)
	}

	for _, project := range projects {
		if err := e.runProject(ctx, project); err != nil {
			return err
		}
	}

	return nil
}

func (e *Engine) runProject(ctx context.Context, project ProjectState) error {
	strategy := normalizeStrategy(project.SyncStrategy)

	for {
		err := e.runnerFor(strategy).Run(ctx, project)
		if !errors.Is(err, ErrFallbackRequired) {
			if err != nil {
				return fmt.Errorf("sync run: project %q strategy %q: %w", project.ID, strategy, err)
			}
			if strategy != project.SyncStrategy {
				if updateErr := e.store.UpdateProjectSyncStrategy(ctx, project.ID, strategy); updateErr != nil {
					return fmt.Errorf("sync run: project %q persist strategy %q: %w", project.ID, strategy, updateErr)
				}
			}
			return nil
		}

		next := nextStrategy(strategy)
		if next == strategy {
			return fmt.Errorf("sync run: project %q strategy %q: %w", project.ID, strategy, err)
		}
		strategy = next
	}
}

func normalizeStrategy(strategy string) string {
	switch strings.TrimSpace(strategy) {
	case StrategyWebDAVSync:
		return StrategyWebDAVSync
	case StrategyCTag:
		return StrategyCTag
	default:
		return StrategyFullScan
	}
}

func nextStrategy(strategy string) string {
	switch strategy {
	case StrategyWebDAVSync:
		return StrategyCTag
	case StrategyCTag:
		return StrategyFullScan
	default:
		return StrategyFullScan
	}
}

func (e *Engine) runnerFor(strategy string) StrategyRunner {
	switch strategy {
	case StrategyWebDAVSync:
		return e.webdavSync
	case StrategyCTag:
		return e.ctag
	default:
		return e.fullscan
	}
}
