package sync

import (
	"context"
	"errors"
	"testing"
)

type stubStore struct {
	projects []ProjectState
	updates  map[string]string
}

func (s *stubStore) ListSyncProjects(context.Context) ([]ProjectState, error) { return s.projects, nil }
func (s *stubStore) UpdateProjectSyncStrategy(_ context.Context, projectID string, strategy string) error {
	if s.updates == nil {
		s.updates = map[string]string{}
	}
	s.updates[projectID] = strategy
	return nil
}

type stubRunner struct{ err error }

func (r stubRunner) Run(context.Context, ProjectState) error { return r.err }

func TestRunFallsBackAndPersistsEffectiveStrategy(t *testing.T) {
	store := &stubStore{projects: []ProjectState{{ID: "p1", SyncStrategy: StrategyWebDAVSync}}}
	engine, err := NewEngine(store, stubRunner{err: ErrFallbackRequired}, stubRunner{err: nil}, stubRunner{err: nil})
	if err != nil {
		t.Fatalf("new engine: %v", err)
	}
	if err := engine.Run(context.Background()); err != nil {
		t.Fatalf("run: %v", err)
	}
	if got := store.updates["p1"]; got != StrategyCTag {
		t.Fatalf("unexpected persisted strategy: got %q want %q", got, StrategyCTag)
	}
}

func TestRunFallsBackToFullScan(t *testing.T) {
	store := &stubStore{projects: []ProjectState{{ID: "p1", SyncStrategy: StrategyWebDAVSync}}}
	engine, _ := NewEngine(store, stubRunner{err: ErrFallbackRequired}, stubRunner{err: ErrFallbackRequired}, stubRunner{})
	if err := engine.Run(context.Background()); err != nil {
		t.Fatalf("run: %v", err)
	}
	if got := store.updates["p1"]; got != StrategyFullScan {
		t.Fatalf("unexpected persisted strategy: got %q want %q", got, StrategyFullScan)
	}
}

func TestRunPropagatesNonFallbackError(t *testing.T) {
	boom := errors.New("boom")
	store := &stubStore{projects: []ProjectState{{ID: "p1", SyncStrategy: StrategyCTag}}}
	engine, _ := NewEngine(store, stubRunner{}, stubRunner{err: boom}, stubRunner{})
	if err := engine.Run(context.Background()); !errors.Is(err, boom) {
		t.Fatalf("expected boom, got %v", err)
	}
}
