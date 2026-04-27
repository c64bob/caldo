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
	return true
}

func (b *setupImportEventBroker) FinishRun() {
	b.mu.Lock()
	b.running = false
	b.mu.Unlock()
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
	for _, subscriber := range b.subscribers {
		select {
		case subscriber <- event:
		default:
		}
	}
}
