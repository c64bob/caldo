package view

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestSavedFiltersPageRendersCreateEditAndDeleteForms(t *testing.T) {
	t.Parallel()

	ctx := WithCSRFToken(context.Background(), "token-123")
	component := SavedFiltersPage([]SavedFilterItemView{
		{
			ID:            "filter-1",
			Name:          "Heute Fokus",
			Query:         "today AND @urgent",
			IsFavorite:    true,
			ServerVersion: 3,
		},
	}, SavedFilterCreateFormView{})

	var rendered bytes.Buffer
	if err := component.Render(ctx, &rendered); err != nil {
		t.Fatalf("render saved filters page: %v", err)
	}

	output := rendered.String()
	for _, want := range []string{
		`data-saved-filter-create-form`,
		`hx-post="/filters"`,
		`href="/filters/filter-1"`,
		`hx-patch="/filters/filter-1"`,
		`hx-delete="/filters/filter-1"`,
		`name="expected_version" value="3"`,
		`X-CSRF-Token`,
		`token-123`,
		`today AND @urgent`,
		`Favorit`,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected saved filter markup %q in %s", want, output)
		}
	}
}

func TestSavedFiltersPagePreservesEditErrorState(t *testing.T) {
	t.Parallel()

	component := SavedFiltersPage([]SavedFilterItemView{
		{
			ID:                "filter-1",
			Name:              "Heute Fokus",
			Query:             "today",
			IsFavorite:        true,
			ServerVersion:     3,
			EditError:         "filter wurde zwischenzeitlich geändert",
			EditName:          "Heute",
			EditQuery:         "today AND (",
			EditFavorite:      false,
			EditFavoriteValid: true,
		},
	}, SavedFilterCreateFormView{})

	var rendered bytes.Buffer
	if err := component.Render(context.Background(), &rendered); err != nil {
		t.Fatalf("render saved filters page: %v", err)
	}

	output := rendered.String()
	for _, want := range []string{
		`data-saved-filter-edit-error`,
		`filter wurde zwischenzeitlich geändert`,
		`value="Heute"`,
		`value="today AND ("`,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected edit state %q in %s", want, output)
		}
	}
}
