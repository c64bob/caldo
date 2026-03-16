package jobs

import (
	"context"
	"log"
	"time"

	"caldo/internal/service"
)

type SyncJob struct {
	Service     *service.SyncService
	PrincipalID string
}

func (j *SyncJob) Run(ctx context.Context) {
	if j == nil || j.Service == nil || j.PrincipalID == "" {
		return
	}
	result, err := j.Service.SyncNow(ctx, j.PrincipalID)
	if err != nil {
		log.Printf("sync job failed for %s: %v", j.PrincipalID, err)
		return
	}
	log.Printf("sync job completed for %s: %d/%d collections (%s) at %s", result.PrincipalID, result.SyncedCollections, result.Collections, result.Mode, result.SyncedAt.Format(time.RFC3339))
}
