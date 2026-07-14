package view

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/a-h/templ"
)

func TestTaskListPagesExposeFullWorkspaceMarker(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		component templ.Component
	}{
		{name: "date scoped", component: DateScopedTasksPage("Heute", "Keine Aufgaben", nil)},
		{name: "configurable", component: ConfigurableTaskListPage("Projekt", "Keine Aufgaben", nil)},
		{name: "search", component: SearchPage("Aufgabe", nil, SearchSaveFilterView{})},
		{name: "configurable search", component: ConfigurableSearchPage("Aufgabe", nil, SearchSaveFilterView{})},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			var rendered bytes.Buffer
			if err := test.component.Render(context.Background(), &rendered); err != nil {
				t.Fatalf("render task list page: %v", err)
			}
			output := rendered.String()
			if !strings.Contains(output, `class="caldo-page caldo-task-list-page" data-task-list-page`) {
				t.Fatalf("task list page missing full workspace marker in %s", output)
			}
		})
	}
}
