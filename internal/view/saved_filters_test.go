package view

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"caldo/internal/query"
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
		`id="saved-filter-create-help"`,
		`id="saved-filter-help-filter-1"`,
		`data-filter-help-toggle`,
		`data-filter-help-popover`,
		`aria-describedby="saved-filter-create-help"`,
		`aria-describedby="saved-filter-help-filter-1"`,
		`priority:high`,
		`completed:false`,
		`before:YYYY-MM-DD`,
		`today AND @Büro`,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected saved filter markup %q in %s", want, output)
		}
	}
	if count := strings.Count(output, `data-filter-help-popover`); count != 2 {
		t.Fatalf("expected create and edit help popovers, got %d in %s", count, output)
	}
	if count := strings.Count(output, `data-filter-help-input`); count != 4 {
		t.Fatalf("expected all create and edit fields to expose help, got %d in %s", count, output)
	}
}

func TestSavedFilterHelpUsesActiveUILanguage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		language string
		want     []string
		unwanted string
	}{
		{language: "de", want: []string{"So funktionieren Filter", "Name ist nur die Bezeichnung", "Beispiel: today AND @Büro"}, unwanted: "How filters work"},
		{language: "en", want: []string{"How filters work", "Name is only the label", "Example: today AND @Office"}, unwanted: "So funktionieren Filter"},
	}

	for _, test := range tests {
		t.Run(test.language, func(t *testing.T) {
			var rendered bytes.Buffer
			ctx := WithUIPreferences(context.Background(), test.language, "system")
			if err := SavedFilterCreateForm(SavedFilterCreateFormView{}).Render(ctx, &rendered); err != nil {
				t.Fatalf("render saved filter help: %v", err)
			}
			output := rendered.String()
			for _, want := range test.want {
				if !strings.Contains(output, want) {
					t.Fatalf("help for %s missing %q in %s", test.language, want, output)
				}
			}
			if strings.Contains(output, test.unwanted) {
				t.Fatalf("help for %s contains %q in %s", test.language, test.unwanted, output)
			}
		})
	}
}

func TestAdvertisedSavedFilterSyntaxParsesAndCompiles(t *testing.T) {
	t.Parallel()

	expressions := []string{
		"today",
		"overdue",
		"upcoming",
		"no date",
		"#Projekt",
		"@Label",
		"priority:high",
		"completed:false",
		"text:foo",
		"before:2026-07-14",
		"after:2026-07-14",
		"today AND @Büro",
		"overdue OR upcoming",
		"NOT completed:true",
	}

	for _, expression := range expressions {
		t.Run(expression, func(t *testing.T) {
			ast, err := query.ParseFilter(query.LexFilter(expression))
			if err != nil {
				t.Fatalf("parse advertised expression %q: %v", expression, err)
			}
			if _, _, err := query.CompileFilter(ast, query.CompileOptions{Now: time.Date(2026, 7, 14, 12, 0, 0, 0, time.UTC)}); err != nil {
				t.Fatalf("compile advertised expression %q: %v", expression, err)
			}
		})
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
