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
		`hx-post="/sync/manual"`,
		`Status: idle`,
		`Letzter Sync:`,
		`02.01.2026 03:04`,
		`datetime="2026-01-02T03:04:00Z"`,
		`data-local-date-time`,
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
		`Status: running`,
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
