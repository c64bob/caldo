package handler

import (
	"context"
	"database/sql"
	"net/http"

	"caldo/internal/db"
	"caldo/internal/view"
)

type syncDependencies struct {
	database     *db.Database
	broker       *eventBroker
	runner       manualSyncRunner
	lifecycleCtx context.Context
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
		_ = view.SyncStatusBadge(status.State, syncTimeView(status.LastSuccessAt)).Render(r.Context(), w)
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
			publishSyncStatusEvent(deps.broker)
			if deps.runner == nil {
				_ = deps.database.FinishManualSyncError(context.WithoutCancel(r.Context()), "sync_unavailable")
				publishSyncStatusEvent(deps.broker)
			} else {
				runCtx := deps.lifecycleCtx
				if runCtx == nil {
					runCtx = context.WithoutCancel(r.Context())
				}
				persistCtx := context.WithoutCancel(runCtx)
				go finishManualSyncRun(runCtx, persistCtx, deps)
			}
		}
		status, _ := deps.database.LoadSyncStatus(r.Context())
		_ = view.SyncStatusBadge(status.State, syncTimeView(status.LastSuccessAt)).Render(r.Context(), w)
	}
}

func finishManualSyncRun(runCtx context.Context, persistCtx context.Context, deps syncDependencies) {
	if err := deps.runner.Run(runCtx); err != nil {
		_ = deps.database.FinishManualSyncError(persistCtx, "sync_failed")
		publishSyncStatusEvent(deps.broker)
		return
	}
	_ = deps.database.FinishManualSyncSuccess(persistCtx)
	publishSyncStatusEvent(deps.broker)
}

func publishSyncStatusEvent(broker *eventBroker) {
	if broker == nil {
		return
	}
	broker.publish(appEvent{Type: "sync", Resource: "sync_status", Version: 0, OriginConnection: "server"})
}

func syncTimeView(ts sql.NullTime) view.LocalDateTimeView {
	return view.LocalDateTimeFromNull(ts, "nie")
}
