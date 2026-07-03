package model

import (
	"reflect"
	"testing"
)

func TestNormalizeLabelName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     string
		want      string
		wantError bool
	}{
		{name: "trimmed", input: "  Home  ", want: "Home"},
		{name: "empty", input: "   ", wantError: true},
		{name: "reserved starred", input: "STARRED", wantError: true},
		{name: "reserved case insensitive", input: "starred", wantError: true},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := NormalizeLabelName(tc.input)
			if tc.wantError {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("unexpected normalized name: got %q want %q", got, tc.want)
			}
		})
	}
}

func TestCategoriesToLabelsAndFavorite(t *testing.T) {
	t.Parallel()

	labels, isFavorite := CategoriesToLabelsAndFavorite([]string{
		"  STARRED ",
		"home",
		"Errands",
		"HOME",
		"",
	})

	if !isFavorite {
		t.Fatal("expected favorite to be true")
	}

	want := []string{"Errands", "home"}
	if !reflect.DeepEqual(labels, want) {
		t.Fatalf("unexpected labels: got %v want %v", labels, want)
	}
}

func TestLabelsAndFavoriteToCategories(t *testing.T) {
	t.Parallel()

	categories, err := LabelsAndFavoriteToCategories([]string{" home ", "Errands", "HOME"}, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []string{"Errands", "home", ReservedFavoriteCategory}
	if !reflect.DeepEqual(categories, want) {
		t.Fatalf("unexpected categories: got %v want %v", categories, want)
	}
}

func TestLabelsAndFavoriteToCategoriesRejectsReservedLabel(t *testing.T) {
	t.Parallel()

	if _, err := LabelsAndFavoriteToCategories([]string{"STARRED"}, false); err == nil {
		t.Fatal("expected reserved label error")
	}
}

func TestNormalizeFavoritePriorityFields(t *testing.T) {
	t.Parallel()

	medium := 5
	categories, priority, err := NormalizeFavoritePriorityFields([]string{"home", ReservedFavoriteCategory}, &medium)
	if err != nil {
		t.Fatalf("normalize favorite priority: %v", err)
	}
	if priority == nil || *priority != FavoritePriorityValue {
		t.Fatalf("expected favorite to set high priority, got %#v", priority)
	}
	if !reflect.DeepEqual(categories, []string{"home", ReservedFavoriteCategory}) {
		t.Fatalf("unexpected categories: %v", categories)
	}

	high := 4
	categories, priority, err = NormalizeFavoritePriorityFields([]string{"home"}, &high)
	if err != nil {
		t.Fatalf("normalize high priority: %v", err)
	}
	if priority == nil || *priority != 4 {
		t.Fatalf("expected high priority to be preserved, got %#v", priority)
	}
	if !reflect.DeepEqual(categories, []string{"home", ReservedFavoriteCategory}) {
		t.Fatalf("expected high priority to set favorite, got %v", categories)
	}
}

func TestCategoriesWithFavoriteFromPriority(t *testing.T) {
	t.Parallel()

	medium := 5
	categories, err := CategoriesWithFavoriteFromPriority([]string{"home", ReservedFavoriteCategory}, &medium)
	if err != nil {
		t.Fatalf("priority category normalization: %v", err)
	}
	if !reflect.DeepEqual(categories, []string{"home"}) {
		t.Fatalf("expected non-high priority to remove favorite, got %v", categories)
	}

	high := 1
	categories, err = CategoriesWithFavoriteFromPriority([]string{"home"}, &high)
	if err != nil {
		t.Fatalf("priority category normalization: %v", err)
	}
	if !reflect.DeepEqual(categories, []string{"home", ReservedFavoriteCategory}) {
		t.Fatalf("expected high priority to add favorite, got %v", categories)
	}
}
