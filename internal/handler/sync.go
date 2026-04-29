package handler

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"sync"

	"caldo/internal/db"
	"caldo/internal/view"
)

type syncDependencies struct {
	database *db.Database
	broker   *syncEventBroker
	runner   manualSyncRunner
}

type manualSyncRunner interface{ Run(ctx context.Context) error }

type syncEventBroker struct {
	mu          sync.Mutex
	subscribers map[chan string]struct{}
}

func newSyncEventBroker() *syncEventBroker { return &syncEventBroker{subscribers: map[chan string]struct{}{}} }
func (b *syncEventBroker) subscribe() chan string {
	b.mu.Lock(); defer b.mu.Unlock(); ch := make(chan string, 1); b.subscribers[ch] = struct{}{}; return ch
}
func (b *syncEventBroker) unsubscribe(ch chan string) { b.mu.Lock(); defer b.mu.Unlock(); delete(b.subscribers, ch); close(ch) }
func (b *syncEventBroker) publish(msg string) {
	b.mu.Lock(); defer b.mu.Unlock(); for ch := range b.subscribers { select { case ch <- msg: default: } }
}

func SyncStatus(deps syncDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		status, err := deps.database.LoadSyncStatus(r.Context())
		if err != nil { http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError); return }
		_ = view.SyncStatusBadge(status.State, formatSyncTime(status.LastSuccessAt)).Render(r.Context(), w)
	}
}

func ManualSync(deps syncDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		started, err := deps.database.TryStartManualSync(r.Context())
		if err != nil { http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError); return }
		if started {
			deps.broker.publish("running")
			if deps.runner == nil {
				_ = deps.database.FinishManualSyncError(r.Context(), "sync_unavailable")
			} else if err := deps.runner.Run(r.Context()); err != nil {
				_ = deps.database.FinishManualSyncError(r.Context(), "sync_failed")
			} else {
				_ = deps.database.FinishManualSyncSuccess(r.Context())
			}
			deps.broker.publish("idle")
		}
		status, _ := deps.database.LoadSyncStatus(r.Context())
		_ = view.SyncStatusBadge(status.State, formatSyncTime(status.LastSuccessAt)).Render(r.Context(), w)
	}
}

func SyncEvents(deps syncDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		flusher, ok := w.(http.Flusher); if !ok { http.Error(w, "streaming unsupported", http.StatusInternalServerError); return }
		ch := deps.broker.subscribe(); defer deps.broker.unsubscribe(ch)
		for {
			select {
			case <-r.Context().Done(): return
			case msg := <-ch:
				fmt.Fprintf(w, "event: sync-status\ndata: %s\n\n", msg)
				flusher.Flush()
			}
		}
	}
}

func formatSyncTime(ts sql.NullTime) string {
	if !ts.Valid {
		return "nie"
	}
	return ts.Time.Local().Format("02.01.2006 15:04")
}
