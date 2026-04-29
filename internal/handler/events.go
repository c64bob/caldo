package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

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
		fmt.Fprintf(w, "event: connected\ndata: %s\n\n", connected)
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
				fmt.Fprintf(w, "event: app-event\ndata: %s\n\n", encoded)
				flusher.Flush()
			}
		}
	}
}
