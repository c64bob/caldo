package db

import (
	"context"
	"path/filepath"
	"testing"

	"caldo/internal/model"
)

func TestTaskViewPreferenceRoundTripAndReset(t *testing.T) {
	t.Parallel()
	database, err := OpenSQLite(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })

	ctx := context.Background()
	preference, err := database.LoadTaskViewPreference(ctx, model.TaskViewProject, "project-1")
	if err != nil {
		t.Fatal(err)
	}
	if preference.SortBy != model.TaskSortDefault || preference.GroupBy != model.TaskGroupNone {
		t.Fatalf("unexpected default: %#v", preference)
	}

	preference.SortBy = model.TaskSortPriority
	preference.SortOrder = model.TaskSortDescending
	preference.GroupBy = model.TaskGroupDue
	if err := database.SaveTaskViewPreference(ctx, preference); err != nil {
		t.Fatal(err)
	}
	loaded, err := database.LoadTaskViewPreference(ctx, model.TaskViewProject, "project-1")
	if err != nil {
		t.Fatal(err)
	}
	if loaded != preference {
		t.Fatalf("loaded preference=%#v want=%#v", loaded, preference)
	}

	if err := database.DeleteTaskViewPreference(ctx, model.TaskViewProject, "project-1"); err != nil {
		t.Fatal(err)
	}
	reset, err := database.LoadTaskViewPreference(ctx, model.TaskViewProject, "project-1")
	if err != nil {
		t.Fatal(err)
	}
	if reset.SortBy != model.TaskSortDefault || reset.SortOrder != model.TaskSortAscending || reset.GroupBy != model.TaskGroupNone {
		t.Fatalf("unexpected reset preference: %#v", reset)
	}
}
