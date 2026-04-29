package handler

import (
	"context"
	"database/sql"
	"net/http"

	"caldo/internal/db"
	"caldo/internal/view"
)

type syncDependencies struct {
	database *db.Database
	broker   *eventBroker
	runner   manualSyncRunner
}

type manualSyncRunner interface {
	Run(ctx context.Context) error
}

func SyncStatus(deps syncDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		status, err := deps.database.LoadSyncStatus(r.Context())
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		_ = view.SyncStatusBadge(status.State, formatSyncTime(status.LastSuccessAt)).Render(r.Context(), w)
	}
}

func ManualSync(deps syncDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		started, err := deps.database.TryStartManualSync(r.Context())
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		if started {
			deps.broker.publish(appEvent{Type: "sync", Resource: "sync_status", Version: 0, OriginConnection: "server"})
			if deps.runner == nil {
				_ = deps.database.FinishManualSyncError(r.Context(), "sync_unavailable")
			} else if err := deps.runner.Run(r.Context()); err != nil {
				_ = deps.database.FinishManualSyncError(r.Context(), "sync_failed")
			} else {
				_ = deps.database.FinishManualSyncSuccess(r.Context())
			}
			deps.broker.publish(appEvent{Type: "sync", Resource: "sync_status", Version: 0, OriginConnection: "server"})
		}
		status, _ := deps.database.LoadSyncStatus(r.Context())
		_ = view.SyncStatusBadge(status.State, formatSyncTime(status.LastSuccessAt)).Render(r.Context(), w)
	}
}

func formatSyncTime(ts sql.NullTime) string {
	if !ts.Valid {
		return "nie"
	}
	return ts.Time.Local().Format("02.01.2006 15:04")
}
