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
