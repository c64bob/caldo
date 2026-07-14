package view

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/a-h/templ"
)

func TestEmptyTaskViewsOmitTaskCreationAction(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		component templ.Component
		wantTitle string
		wantText  string
	}{
		{
			name:      "date scoped view",
			component: DateScopedTasksPage("Heute", "Keine fälligen Aufgaben.", nil),
			wantTitle: "Keine fälligen Aufgaben.",
			wantText:  "Es gibt in dieser Ansicht gerade nichts zu bearbeiten.",
		},
		{
			name:      "configurable project label filter or completed view",
			component: ConfigurableTaskListPage("Projekt", "Keine offenen Aufgaben in diesem Projekt.", nil),
			wantTitle: "Keine offenen Aufgaben in diesem Projekt.",
			wantText:  "Es gibt in dieser Ansicht gerade nichts zu bearbeiten.",
		},
		{
			name:      "blank search",
			component: SearchLiveResults("", nil, SearchSaveFilterView{}),
			wantTitle: "Noch keine Suche",
			wantText:  "Gib einen Suchbegriff ein.",
		},
		{
			name:      "blank configurable search",
			component: ConfigurableSearchLiveResults("  ", nil, SearchSaveFilterView{}),
			wantTitle: "Noch keine Suche",
			wantText:  "Gib einen Suchbegriff ein.",
		},
		{
			name:      "search without results",
			component: ConfigurableSearchLiveResults("missing", nil, SearchSaveFilterView{}),
			wantTitle: "Keine aktiven Aufgaben gefunden",
			wantText:  "Passe den Suchbegriff oder Projekt- und Labeltokens an.",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			var rendered bytes.Buffer
			if err := test.component.Render(context.Background(), &rendered); err != nil {
				t.Fatalf("render empty task view: %v", err)
			}

			output := rendered.String()
			for _, want := range []string{`data-empty-state`, test.wantTitle, test.wantText, `caldo-empty-state-icon`} {
				if !strings.Contains(output, want) {
					t.Fatalf("empty task view missing %q in %s", want, output)
				}
			}
			for _, unwanted := range []string{`Aufgabe erstellen`, `href="/quick-add"`, `data-quick-add-open`} {
				if strings.Contains(output, unwanted) {
					t.Fatalf("empty task view contains task creation action %q in %s", unwanted, output)
				}
			}
		})
	}
}
