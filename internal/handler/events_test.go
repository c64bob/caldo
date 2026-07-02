package handler

import (
	"bufio"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type sseBlockResult struct {
	block string
	err   error
}

func TestEventsPublishesSyncStatusHTML(t *testing.T) {
	database := openSQLiteForSyncHandlerTest(t)
	if err := database.FinishManualSyncSuccess(context.Background()); err != nil {
		t.Fatalf("finish manual sync: %v", err)
	}
	broker := newEventBroker()
	server := httptest.NewServer(Events(syncDependencies{database: database, broker: broker}))
	t.Cleanup(server.Close)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("open events stream: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status code: got %d want %d", response.StatusCode, http.StatusOK)
	}

	reader := bufio.NewReader(response.Body)
	connected := readSSEBlockBefore(t, reader, cancel)
	if !strings.Contains(connected, "event: connected") {
		t.Fatalf("expected connected event, got %q", connected)
	}

	broker.publish(appEvent{Type: "sync", Resource: "sync_status", Version: 0, OriginConnection: "server"})
	appEventBlock := readSSEBlockBefore(t, reader, cancel)
	if !strings.Contains(appEventBlock, "event: app-event") || !strings.Contains(appEventBlock, `"resource":"sync_status"`) {
		t.Fatalf("expected app-event sync status block, got %q", appEventBlock)
	}
	syncStatusBlock := readSSEBlockBefore(t, reader, cancel)
	for _, want := range []string{
		"event: sync-status",
		`hx-target="#sync-status"`,
		`hx-swap="innerHTML"`,
		`data-sync-request`,
		"Letzter erfolgreicher Sync: ",
		`data-sync-tooltip-template`,
	} {
		if !strings.Contains(syncStatusBlock, want) {
			t.Fatalf("expected sync-status block to include %q in %q", want, syncStatusBlock)
		}
	}
	if strings.Contains(syncStatusBlock, "Letzter erfolgreicher Sync: nie") {
		t.Fatalf("expected sync-status block to include persisted success time, got %q", syncStatusBlock)
	}
}

func readSSEBlockBefore(t *testing.T, reader *bufio.Reader, cancel context.CancelFunc) string {
	t.Helper()
	resultCh := make(chan sseBlockResult, 1)
	go func() {
		block, err := readSSEBlock(reader)
		resultCh <- sseBlockResult{block: block, err: err}
	}()

	select {
	case result := <-resultCh:
		if result.err != nil {
			t.Fatalf("read sse block: %v", result.err)
		}
		return result.block
	case <-time.After(2 * time.Second):
		cancel()
		t.Fatal("timed out waiting for sse block")
	}
	return ""
}

func readSSEBlock(reader *bufio.Reader) (string, error) {
	var block strings.Builder
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return "", err
		}
		if line == "\n" || line == "\r\n" {
			return block.String(), nil
		}
		block.WriteString(line)
	}
}
