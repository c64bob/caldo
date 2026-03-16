package service

import (
	"context"
	"fmt"
	"strings"

	"caldo/internal/caldav"
	"caldo/internal/domain"
	"caldo/internal/security"
	"caldo/internal/store/sqlite"
)

type TaskService struct {
	accountsRepo *sqlite.DAVAccountsRepo
	key          []byte
	defaultList  string
	caldavClient *caldav.Client
	tasksRepo    *caldav.TasksRepo
}

func NewTaskService(accountsRepo *sqlite.DAVAccountsRepo, key []byte, defaultList string) *TaskService {
	client := caldav.NewClient()
	return &TaskService{
		accountsRepo: accountsRepo,
		key:          key,
		defaultList:  defaultList,
		caldavClient: client,
		tasksRepo:    caldav.NewTasksRepo(client),
	}
}

type TaskPageData struct {
	Lists          []domain.List
	Tasks          []domain.Task
	ActiveListID   string
	HasCredentials bool
}

func (s *TaskService) LoadTaskPage(ctx context.Context, principalID string, selectedListID string) (TaskPageData, error) {
	if strings.TrimSpace(principalID) == "" {
		return TaskPageData{}, fmt.Errorf("missing principal")
	}
	account, ok, err := s.accountsRepo.GetByPrincipal(ctx, principalID)
	if err != nil {
		return TaskPageData{}, fmt.Errorf("load dav account: %w", err)
	}
	if !ok {
		return TaskPageData{HasCredentials: false}, nil
	}

	password, err := security.DecryptAESGCM(s.key, account.PasswordEncrypted)
	if err != nil {
		return TaskPageData{}, fmt.Errorf("decrypt dav password: %w", err)
	}

	discovery, err := s.caldavClient.DiscoverTaskCollections(ctx, account.ServerURL, account.Username, string(password), s.defaultList)
	if err != nil {
		return TaskPageData{}, err
	}

	lists := make([]domain.List, 0, len(discovery.Collections))
	for idx, c := range discovery.Collections {
		if !c.SupportsVTODO {
			continue
		}
		isDefault := idx == 0
		if strings.TrimSpace(c.DisplayName) == strings.TrimSpace(s.defaultList) {
			isDefault = true
		}
		lists = append(lists, domain.List{ID: c.ID, DisplayName: c.DisplayName, Href: c.Href, IsDefault: isDefault})
	}
	if len(lists) == 0 {
		return TaskPageData{HasCredentials: true}, nil
	}

	activeListID := selectedListID
	if strings.TrimSpace(activeListID) == "" {
		activeListID = lists[0].ID
	}

	activeCollection := discovery.Collections[0]
	for _, c := range discovery.Collections {
		if c.ID == activeListID {
			activeCollection = c
			break
		}
	}

	tasks, err := s.tasksRepo.ListTasks(ctx, account.ServerURL, account.Username, string(password), activeCollection)
	if err != nil {
		return TaskPageData{}, err
	}

	return TaskPageData{
		Lists:          lists,
		Tasks:          tasks,
		ActiveListID:   activeListID,
		HasCredentials: true,
	}, nil
}
