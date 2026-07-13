package view

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"caldo/internal/db"
	"github.com/a-h/templ"
)

func TestNormalPageContentDoesNotRepeatTopbarPageTitle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		component templ.Component
		unwanted  string
	}{
		{
			name:      "date scoped tasks",
			component: DateScopedTasksPage("Heute", "Keine Aufgaben", nil),
			unwanted:  `<h2 class="caldo-page-title">Heute</h2>`,
		},
		{
			name:      "projects overview",
			component: ProjectsOverviewPage(nil, ProjectFeedback{}),
			unwanted:  `<h2 class="caldo-page-title">Projekte</h2>`,
		},
		{
			name:      "navigation overview",
			component: NavigationOverviewPage("Labels", "Keine Labels", nil),
			unwanted:  `<h2 class="caldo-page-title">Labels</h2>`,
		},
		{
			name:      "search",
			component: SearchPage("", nil, SearchSaveFilterView{}),
			unwanted:  `<h2 class="caldo-page-title">Globale Suche</h2>`,
		},
		{
			name:      "settings",
			component: SettingsPageContent(SettingsPageView{}),
			unwanted:  `<h2 class="caldo-page-title">Einstellungen</h2>`,
		},
		{
			name:      "saved filters",
			component: SavedFiltersPage(nil, SavedFilterCreateFormView{}),
			unwanted:  `<h2 class="caldo-page-title">Filter</h2>`,
		},
		{
			name:      "conflict list",
			component: ConflictListPage(nil),
			unwanted:  `<h2 class="caldo-page-title">Konflikte</h2>`,
		},
		{
			name:      "conflict detail",
			component: ConflictDetailPage(db.ConflictDetail{}, nil),
			unwanted:  `<h2 class="caldo-page-title">Konfliktdetail</h2>`,
		},
		{
			name:      "quick add",
			component: QuickAddPage(nil, "", ""),
			unwanted:  `<h2 class="caldo-page-title">Quick Add</h2>`,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var rendered bytes.Buffer
			if err := tt.component.Render(context.Background(), &rendered); err != nil {
				t.Fatalf("render page content: %v", err)
			}

			if output := rendered.String(); strings.Contains(output, tt.unwanted) {
				t.Fatalf("page content must not repeat topbar title %q in %s", tt.unwanted, output)
			}
		})
	}
}
