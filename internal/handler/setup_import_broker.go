package handler

import (
	"sync"

	"github.com/google/uuid"
)

type setupImportEvent struct {
	Event string
	Data  string
}

type setupImportSubscriber struct {
	id string
	ch chan setupImportEvent
}

type setupImportEventBroker struct {
	mu          sync.Mutex
	subscribers map[string]chan setupImportEvent
	running     bool
	started     bool
	succeeded   bool
	failed      bool
}

func newSetupImportEventBroker() *setupImportEventBroker {
	return &setupImportEventBroker{subscribers: make(map[string]chan setupImportEvent)}
}

func (b *setupImportEventBroker) StartRun() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.running {
		return false
	}
	b.running = true
	b.started = true
	b.succeeded = false
	b.failed = false
	return true
}

func (b *setupImportEventBroker) FinishRun() {
	b.mu.Lock()
	b.running = false
	b.mu.Unlock()
}

func (b *setupImportEventBroker) CanCompleteSetup() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.started && b.succeeded && !b.running && !b.failed
}

func (b *setupImportEventBroker) Subscribe() setupImportSubscriber {
	b.mu.Lock()
	defer b.mu.Unlock()
	id := uuid.NewString()
	ch := make(chan setupImportEvent, 16)
	b.subscribers[id] = ch
	return setupImportSubscriber{id: id, ch: ch}
}

func (b *setupImportEventBroker) Unsubscribe(id string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	ch, ok := b.subscribers[id]
	if !ok {
		return
	}
	delete(b.subscribers, id)
	close(ch)
}

func (b *setupImportEventBroker) Publish(event setupImportEvent) {
	b.mu.Lock()
	defer b.mu.Unlock()
	switch event.Event {
	case "done":
		b.running = false
		b.succeeded = true
		b.failed = false
	case "error":
		b.running = false
		b.succeeded = false
		b.failed = true
	}
	for _, subscriber := range b.subscribers {
		select {
		case subscriber <- event:
		default:
		}
	}
}
