package model

import (
	"errors"
	"strings"
)

const (
	TaskViewToday     = "today"
	TaskViewUpcoming  = "upcoming"
	TaskViewOverdue   = "overdue"
	TaskViewFavorites = "favorites"
	TaskViewNoDate    = "no_date"
	TaskViewCompleted = "completed"
	TaskViewProject   = "project"
	TaskViewLabel     = "label"
	TaskViewFilter    = "filter"
	TaskViewSearch    = "search"

	TaskSortDefault  = "default"
	TaskSortDue      = "due"
	TaskSortPriority = "priority"
	TaskSortName     = "name"
	TaskSortAdded    = "added"

	TaskSortAscending  = "asc"
	TaskSortDescending = "desc"

	TaskGroupNone     = "none"
	TaskGroupProject  = "project"
	TaskGroupDue      = "due"
	TaskGroupAdded    = "added"
	TaskGroupPriority = "priority"
)

// ErrInvalidTaskViewPreference indicates an unsupported task-list display value.
var ErrInvalidTaskViewPreference = errors.New("invalid task view preference")

// TaskViewPreference stores display-only ordering for one task-list scope.
type TaskViewPreference struct {
	ViewKind  string
	ViewID    string
	SortBy    string
	SortOrder string
	GroupBy   string
}

// DefaultTaskViewPreference returns the default display preference for a scope.
func DefaultTaskViewPreference(viewKind, viewID string) TaskViewPreference {
	return TaskViewPreference{
		ViewKind:  strings.TrimSpace(viewKind),
		ViewID:    strings.TrimSpace(viewID),
		SortBy:    TaskSortDefault,
		SortOrder: TaskSortAscending,
		GroupBy:   TaskGroupNone,
	}
}

// ValidateTaskViewPreference verifies all persisted display values.
func ValidateTaskViewPreference(preference TaskViewPreference) error {
	preference.ViewKind = strings.TrimSpace(preference.ViewKind)
	preference.ViewID = strings.TrimSpace(preference.ViewID)
	if !validTaskViewScope(preference.ViewKind, preference.ViewID) ||
		!oneOf(preference.SortBy, TaskSortDefault, TaskSortDue, TaskSortPriority, TaskSortName, TaskSortAdded) ||
		!oneOf(preference.SortOrder, TaskSortAscending, TaskSortDescending) ||
		!oneOf(preference.GroupBy, TaskGroupNone, TaskGroupProject, TaskGroupDue, TaskGroupAdded, TaskGroupPriority) {
		return ErrInvalidTaskViewPreference
	}
	return nil
}

func validTaskViewScope(viewKind, viewID string) bool {
	switch viewKind {
	case TaskViewProject, TaskViewLabel, TaskViewFilter:
		return viewID != ""
	case TaskViewToday, TaskViewUpcoming, TaskViewOverdue, TaskViewFavorites, TaskViewNoDate, TaskViewCompleted, TaskViewSearch:
		return viewID == ""
	default:
		return false
	}
}

func oneOf(value string, allowed ...string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}
