package service

import (
	"context"
	"fmt"
	"strconv"
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

type TaskMutationInput struct {
	ListID   string
	UID      string
	Href     string
	ETag     string
	Summary  string
	Status   string
	Priority int
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

func (s *TaskService) CreateTask(ctx context.Context, principalID string, in TaskMutationInput) (domain.Task, error) {
	account, password, collection, err := s.loadCredentialsAndCollection(ctx, principalID, in.ListID)
	if err != nil {
		return domain.Task{}, err
	}
	prio := in.Priority
	if prio < 0 {
		prio = 0
	}
	task := domain.Task{UID: in.UID, Summary: strings.TrimSpace(in.Summary), Status: strings.TrimSpace(in.Status), Priority: prio}
	return s.tasksRepo.CreateTask(ctx, account.ServerURL, account.Username, string(password), collection, task)
}

func (s *TaskService) UpdateTask(ctx context.Context, principalID string, in TaskMutationInput) (domain.Task, error) {
	account, password, _, err := s.loadCredentialsAndCollection(ctx, principalID, in.ListID)
	if err != nil {
		return domain.Task{}, err
	}
	task := domain.Task{UID: strings.TrimSpace(in.UID), Href: strings.TrimSpace(in.Href), ETag: strings.TrimSpace(in.ETag), Summary: strings.TrimSpace(in.Summary), Status: strings.TrimSpace(in.Status), Priority: in.Priority}
	return s.tasksRepo.UpdateTask(ctx, account.ServerURL, account.Username, string(password), task)
}

func (s *TaskService) DeleteTask(ctx context.Context, principalID string, in TaskMutationInput) error {
	account, password, _, err := s.loadCredentialsAndCollection(ctx, principalID, in.ListID)
	if err != nil {
		return err
	}
	return s.tasksRepo.DeleteTask(ctx, account.ServerURL, account.Username, string(password), strings.TrimSpace(in.Href), strings.TrimSpace(in.ETag))
}

func ParsePriority(raw string) int {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return 0
	}
	if value < 0 {
		return 0
	}
	if value > 9 {
		return 9
	}
	return value
}

func (s *TaskService) loadCredentialsAndCollection(ctx context.Context, principalID, listID string) (sqlite.DAVAccount, []byte, caldav.Collection, error) {
	if strings.TrimSpace(principalID) == "" {
		return sqlite.DAVAccount{}, nil, caldav.Collection{}, fmt.Errorf("missing principal")
	}
	account, ok, err := s.accountsRepo.GetByPrincipal(ctx, principalID)
	if err != nil {
		return sqlite.DAVAccount{}, nil, caldav.Collection{}, fmt.Errorf("load dav account: %w", err)
	}
	if !ok {
		return sqlite.DAVAccount{}, nil, caldav.Collection{}, fmt.Errorf("kein DAV-Account hinterlegt")
	}
	password, err := security.DecryptAESGCM(s.key, account.PasswordEncrypted)
	if err != nil {
		return sqlite.DAVAccount{}, nil, caldav.Collection{}, fmt.Errorf("decrypt dav password: %w", err)
	}
	discovery, err := s.caldavClient.DiscoverTaskCollections(ctx, account.ServerURL, account.Username, string(password), s.defaultList)
	if err != nil {
		return sqlite.DAVAccount{}, nil, caldav.Collection{}, err
	}
	for _, c := range discovery.Collections {
		if !c.SupportsVTODO {
			continue
		}
		if strings.TrimSpace(listID) == "" || c.ID == listID {
			return account, password, c, nil
		}
	}
	return sqlite.DAVAccount{}, nil, caldav.Collection{}, fmt.Errorf("Task-Liste nicht gefunden")
}
