package view

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestSyncStatusFrameKeepsStableSwapTargetOutsideSwappedBadge(t *testing.T) {
	t.Parallel()

	ctx := WithCSRFToken(context.Background(), "token-123")
	component := SyncStatusFrame("idle", LocalDateTimeView{Text: "02.01.2026 03:04", ISO: "2026-01-02T03:04:00Z"})

	var rendered bytes.Buffer
	if err := component.Render(ctx, &rendered); err != nil {
		t.Fatalf("render sync status frame: %v", err)
	}

	output := rendered.String()
	for _, want := range []string{
		`class="caldo-sync-status" id="sync-status"`,
		`id="sync-status"`,
		`hx-swap="innerHTML"`,
		`data-sync-request`,
		`data-sync-state="idle"`,
		`data-sync-tooltip-template="Jetzt synchronisieren. Status: bereit. Letzter erfolgreicher Sync: {last}"`,
		`hx-post="/sync/manual"`,
		`aria-label="Jetzt synchronisieren. Status: bereit. Letzter erfolgreicher Sync: 02.01.2026 03:04"`,
		`title="Jetzt synchronisieren. Status: bereit. Letzter erfolgreicher Sync: 02.01.2026 03:04"`,
		`Status: bereit`,
		`Letzter erfolgreicher Sync:`,
		`02.01.2026 03:04`,
		`datetime="2026-01-02T03:04:00Z"`,
		`data-local-date-time`,
		`data-sync-last-value`,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected sync stream to include %q in %s", want, output)
		}
	}
}

func TestSyncStatusBadgePollsWhileRunning(t *testing.T) {
	t.Parallel()

	component := SyncStatusBadge("running", LocalDateTimeView{Text: "nie"})

	var rendered bytes.Buffer
	if err := component.Render(context.Background(), &rendered); err != nil {
		t.Fatalf("render sync status badge: %v", err)
	}

	output := rendered.String()
	for _, want := range []string{
		`caldo-sync-action-running`,
		`data-sync-state="running"`,
		`Status: läuft`,
		`hx-get="/sync/status"`,
		`hx-trigger="load delay:1s"`,
		`hx-target="#sync-status"`,
		`hx-swap="innerHTML"`,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected running sync badge to include %q in %s", want, output)
		}
	}
}
