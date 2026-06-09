package view

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestUndoNotificationRendersExpiryAndAction(t *testing.T) {
	t.Parallel()

	component := UndoNotification(UndoSnapshotView{
		ActionType:   "task_deleted",
		ExpiresAtISO: "2026-06-09T17:30:00Z",
	})

	var rendered bytes.Buffer
	if err := component.Render(context.Background(), &rendered); err != nil {
		t.Fatalf("render undo notification: %v", err)
	}

	output := rendered.String()
	for _, want := range []string{
		`data-undo-toast`,
		`data-undo-expires-at="2026-06-09T17:30:00Z"`,
		`Aufgabe gelöscht.`,
		`Wiederherstellung ist kurz verfügbar.`,
		`data-undo-countdown`,
		`data-undo-action`,
		`Rückgängig`,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected undo notification to include %q in %s", want, output)
		}
	}
}

func TestUndoResultNotificationRendersResultOnly(t *testing.T) {
	t.Parallel()

	component := UndoResultNotification("Rückgängig ausgeführt.")

	var rendered bytes.Buffer
	if err := component.Render(context.Background(), &rendered); err != nil {
		t.Fatalf("render undo result notification: %v", err)
	}

	output := rendered.String()
	if !strings.Contains(output, "Rückgängig ausgeführt.") {
		t.Fatalf("expected undo result text in %s", output)
	}
	if strings.Contains(output, `data-undo-action`) {
		t.Fatalf("undo result must not include an action button: %s", output)
	}
}
