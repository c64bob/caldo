package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"

	"caldo/internal/db"
	"caldo/internal/view"
	"github.com/google/uuid"
)

type appEvent struct {
	Type             string `json:"type"`
	Resource         string `json:"resource"`
	Version          int    `json:"version"`
	OriginConnection string `json:"origin_connection"`
}

type eventSubscription struct {
	id string
	ch chan appEvent
}

type eventBroker struct {
	mu            sync.Mutex
	subscribers   map[string]chan appEvent
	connectionSeq uint64
}

func newEventBroker() *eventBroker {
	return &eventBroker{subscribers: map[string]chan appEvent{}}
}

func (b *eventBroker) subscribe() eventSubscription {
	b.mu.Lock()
	defer b.mu.Unlock()
	id := uuid.NewString()
	ch := make(chan appEvent, 8)
	b.subscribers[id] = ch
	return eventSubscription{id: id, ch: ch}
}

func (b *eventBroker) unsubscribe(subscription eventSubscription) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if ch, ok := b.subscribers[subscription.id]; ok {
		delete(b.subscribers, subscription.id)
		close(ch)
	}
}

func (b *eventBroker) publish(event appEvent) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, ch := range b.subscribers {
		select {
		case ch <- event:
		default:
		}
	}
}

func Events(deps syncDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unsupported", http.StatusInternalServerError)
			return
		}

		subscription := deps.broker.subscribe()
		defer deps.broker.unsubscribe(subscription)

		connected, _ := json.Marshal(map[string]string{"connection_id": subscription.id})
		if err := writeSSEEvent(w, "connected", connected); err != nil {
			return
		}
		flusher.Flush()

		for {
			select {
			case <-r.Context().Done():
				return
			case event := <-subscription.ch:
				encoded, err := json.Marshal(event)
				if err != nil {
					continue
				}
				if err := writeSSEEvent(w, "app-event", encoded); err != nil {
					return
				}
				if event.Type == "sync" && event.Resource == "sync_status" {
					if html, ok := syncStatusBadgeHTML(r.Context(), deps.database); ok {
						if err := writeSSEEvent(w, "sync-status", []byte(html)); err != nil {
							return
						}
					}
				}
				flusher.Flush()
			}
		}
	}
}

func writeSSEEvent(w io.Writer, eventName string, data []byte) error {
	if _, err := fmt.Fprintf(w, "event: %s\n", eventName); err != nil {
		return err
	}
	lines := bytes.Split(data, []byte("\n"))
	if len(lines) > 0 && len(lines[len(lines)-1]) == 0 {
		lines = lines[:len(lines)-1]
	}
	if len(lines) == 0 {
		lines = [][]byte{{}}
	}
	for _, line := range lines {
		if _, err := fmt.Fprintf(w, "data: %s\n", line); err != nil {
			return err
		}
	}
	_, err := fmt.Fprint(w, "\n")
	return err
}

func syncStatusBadgeHTML(ctx context.Context, database *db.Database) (string, bool) {
	if database == nil {
		return "", false
	}
	status, err := database.LoadSyncStatus(ctx)
	if err != nil {
		return "", false
	}
	var rendered bytes.Buffer
	if err := view.SyncStatusBadge(status.State, syncTimeView(status.LastSuccessAt)).Render(ctx, &rendered); err != nil {
		return "", false
	}
	return rendered.String(), true
}
