package db

import (
	"context"
	"testing"
)

func TestSavedFilterCRUDAndVersioning(t *testing.T) {
	database := openViewTestDB(t)
	t.Cleanup(func() { _ = database.Close() })

	created, err := database.CreateSavedFilter(context.Background(), "Heute", "today AND NOT completed:true", true)
	if err != nil {
		t.Fatalf("create saved filter: %v", err)
	}
	if created.ServerVersion != 1 {
		t.Fatalf("unexpected initial version: got %d want 1", created.ServerVersion)
	}

	updated, err := database.UpdateSavedFilter(context.Background(), created.ID, "Nur Heute", "today", false, 1)
	if err != nil {
		t.Fatalf("update saved filter: %v", err)
	}
	if updated.ServerVersion != 2 {
		t.Fatalf("unexpected updated version: got %d want 2", updated.ServerVersion)
	}

	if _, err := database.UpdateSavedFilter(context.Background(), created.ID, "Stale", "today", false, 1); err != ErrSavedFilterVersionConflict {
		t.Fatalf("expected version conflict, got %v", err)
	}

	list, err := database.ListSavedFilters(context.Background())
	if err != nil {
		t.Fatalf("list saved filters: %v", err)
	}
	if len(list) != 1 || list[0].Name != "Nur Heute" {
		t.Fatalf("unexpected list result: %+v", list)
	}

	if err := database.DeleteSavedFilter(context.Background(), created.ID, 2); err != nil {
		t.Fatalf("delete saved filter: %v", err)
	}
	if err := database.DeleteSavedFilter(context.Background(), created.ID, 2); err != ErrSavedFilterVersionConflict {
		t.Fatalf("expected version conflict for stale delete, got %v", err)
	}
}

func TestEvaluateSavedFilterInvalidSyntaxReturnsEmpty(t *testing.T) {
	where, args, ok, err := EvaluateSavedFilter("today AND (")
	if err != nil {
		t.Fatalf("evaluate saved filter: %v", err)
	}
	if ok {
		t.Fatalf("expected invalid query to return ok=false, got true where=%q args=%v", where, args)
	}
}

func TestEvaluateSavedFilterValidSyntax(t *testing.T) {
	where, args, ok, err := EvaluateSavedFilter("today AND NOT completed:true")
	if err != nil {
		t.Fatalf("evaluate saved filter: %v", err)
	}
	if !ok {
		t.Fatalf("expected valid query")
	}
	if where == "" || len(args) == 0 {
		t.Fatalf("expected compiled sql and args, got where=%q args=%v", where, args)
	}
}
