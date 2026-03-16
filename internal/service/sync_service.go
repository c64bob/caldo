package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"caldo/internal/caldav"
	"caldo/internal/security"
	"caldo/internal/store/sqlite"
)

type SyncService struct {
	accountsRepo  *sqlite.DAVAccountsRepo
	syncStateRepo *sqlite.SyncStateRepo
	key           []byte
	defaultList   string
	caldavClient  *caldav.Client
	tasksRepo     *caldav.TasksRepo
}

type SyncResult struct {
	PrincipalID       string
	Collections       int
	SyncedCollections int
	Mode              string
	SyncedAt          time.Time
}

func NewSyncService(accountsRepo *sqlite.DAVAccountsRepo, syncStateRepo *sqlite.SyncStateRepo, key []byte, defaultList string) *SyncService {
	client := caldav.NewClient()
	return &SyncService{
		accountsRepo:  accountsRepo,
		syncStateRepo: syncStateRepo,
		key:           key,
		defaultList:   defaultList,
		caldavClient:  client,
		tasksRepo:     caldav.NewTasksRepo(client),
	}
}

func (s *SyncService) SyncNow(ctx context.Context, principalID string) (SyncResult, error) {
	principalID = strings.TrimSpace(principalID)
	if principalID == "" {
		return SyncResult{}, fmt.Errorf("missing principal")
	}
	account, ok, err := s.accountsRepo.GetByPrincipal(ctx, principalID)
	if err != nil {
		return SyncResult{}, fmt.Errorf("load dav account: %w", err)
	}
	if !ok {
		return SyncResult{}, fmt.Errorf("kein DAV-Account hinterlegt")
	}
	password, err := security.DecryptAESGCM(s.key, account.PasswordEncrypted)
	if err != nil {
		return SyncResult{}, fmt.Errorf("decrypt dav password: %w", err)
	}
	discovery, err := s.caldavClient.DiscoverTaskCollections(ctx, account.ServerURL, account.Username, string(password), s.defaultList)
	if err != nil {
		return SyncResult{}, err
	}
	result := SyncResult{PrincipalID: principalID, SyncedAt: time.Now().UTC()}
	for _, collection := range discovery.Collections {
		if !collection.SupportsVTODO {
			continue
		}
		result.Collections++
		state, _, err := s.syncStateRepo.Get(ctx, principalID, collection.ID)
		if err != nil {
			return SyncResult{}, fmt.Errorf("load sync-state: %w", err)
		}
		snapshot, err := s.tasksRepo.SyncCollection(ctx, account.ServerURL, account.Username, string(password), collection, state.SyncToken)
		if err != nil {
			_ = s.syncStateRepo.SaveError(ctx, principalID, collection.ID, err.Error(), time.Now())
			return SyncResult{}, fmt.Errorf("sync collection %q: %w", collection.DisplayName, err)
		}
		result.SyncedCollections++
		result.Mode = snapshot.Mode
		if err := s.syncStateRepo.Upsert(ctx, sqlite.SyncState{
			PrincipalID:   principalID,
			CollectionID:  collection.ID,
			SyncToken:     snapshot.SyncToken,
			ETagDigest:    snapshot.ETagDigest,
			ResourceCount: snapshot.ResourceCount,
			LastMode:      snapshot.Mode,
			LastSyncedAt:  result.SyncedAt,
		}); err != nil {
			return SyncResult{}, fmt.Errorf("persist sync-state: %w", err)
		}
	}
	return result, nil
}
