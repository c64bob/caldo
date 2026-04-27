package handler

import "sync/atomic"

// SetupState stores the runtime setup completion gate state.
type SetupState struct {
	complete atomic.Bool
}

// NewSetupState returns a setup gate state initialized from persisted setup status.
func NewSetupState(complete bool) *SetupState {
	state := &SetupState{}
	state.complete.Store(complete)
	return state
}

// IsComplete reports whether setup is completed for normal route access.
func (s *SetupState) IsComplete() bool {
	if s == nil {
		return false
	}
	return s.complete.Load()
}

// MarkComplete opens the setup gate for normal route access.
func (s *SetupState) MarkComplete() {
	if s == nil {
		return
	}
	s.complete.Store(true)
}
