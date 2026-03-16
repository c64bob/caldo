package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

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
	ListID      string
	UID         string
	Href        string
	ETag        string
	Summary     string
	Status      string
	Priority    int
	Description string
	Due         *time.Time
	DueKind     string
	Categories  []string
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
	task.Description = strings.TrimSpace(in.Description)
	task.Due = in.Due
	task.DueKind = strings.TrimSpace(in.DueKind)
	task.Categories = in.Categories
	return s.tasksRepo.CreateTask(ctx, account.ServerURL, account.Username, string(password), collection, task)
}

func (s *TaskService) UpdateTask(ctx context.Context, principalID string, in TaskMutationInput) (domain.Task, error) {
	account, password, collection, err := s.loadCredentialsAndCollection(ctx, principalID, in.ListID)
	if err != nil {
		return domain.Task{}, err
	}
	if strings.TrimSpace(in.ETag) == "" {
		return domain.Task{}, caldav.ErrMissingETag
	}
	current, err := s.loadTaskForUpdate(ctx, account, password, collection, in)
	if err != nil {
		return domain.Task{}, err
	}
	if summary := strings.TrimSpace(in.Summary); summary != "" {
		current.Summary = summary
	}
	if status := strings.TrimSpace(in.Status); status != "" {
		current.Status = status
	}
	current.Priority = in.Priority
	if description := strings.TrimSpace(in.Description); description != "" {
		current.Description = description
	}
	if in.Due != nil && strings.TrimSpace(in.DueKind) != "" {
		current.Due = in.Due
		current.DueKind = strings.TrimSpace(in.DueKind)
	}
	if len(in.Categories) > 0 {
		current.Categories = in.Categories
	}
	current.ETag = strings.TrimSpace(in.ETag)
	return s.tasksRepo.UpdateTask(ctx, account.ServerURL, account.Username, string(password), current)
}

func (s *TaskService) DeleteTask(ctx context.Context, principalID string, in TaskMutationInput) error {
	account, password, collection, err := s.loadCredentialsAndCollection(ctx, principalID, in.ListID)
	if err != nil {
		return err
	}
	if strings.TrimSpace(in.ETag) == "" {
		return caldav.ErrMissingETag
	}
	current, err := s.loadTaskForUpdate(ctx, account, password, collection, in)
	if err != nil {
		return err
	}
	return s.tasksRepo.DeleteTask(ctx, account.ServerURL, account.Username, string(password), strings.TrimSpace(current.Href), strings.TrimSpace(in.ETag))
}

func (s *TaskService) loadTaskForUpdate(ctx context.Context, account sqlite.DAVAccount, password []byte, collection caldav.Collection, in TaskMutationInput) (domain.Task, error) {
	tasks, err := s.tasksRepo.ListTasks(ctx, account.ServerURL, account.Username, string(password), collection)
	if err != nil {
		return domain.Task{}, err
	}
	targetHref := strings.TrimSpace(in.Href)
	targetUID := strings.TrimSpace(in.UID)
	for _, task := range tasks {
		if targetHref != "" && strings.TrimSpace(task.Href) == targetHref {
			return task, nil
		}
		if targetUID != "" && strings.TrimSpace(task.UID) == targetUID {
			return task, nil
		}
	}
	return domain.Task{}, errors.New("Task zum Aktualisieren nicht gefunden")
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

func ParseCategories(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	seen := map[string]struct{}{}
	for _, part := range parts {
		c := strings.TrimSpace(part)
		if c == "" {
			continue
		}
		k := strings.ToLower(c)
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		out = append(out, c)
	}
	return out
}

func ParseDue(raw string) (*time.Time, string) {
	v := strings.TrimSpace(raw)
	if v == "" {
		return nil, ""
	}
	if t, err := time.Parse("2006-01-02", v); err == nil {
		return &t, "date"
	}
	if t, err := time.ParseInLocation("2006-01-02T15:04", v, time.Local); err == nil {
		return &t, "datetime"
	}
	return nil, ""
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
