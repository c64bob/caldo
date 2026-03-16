package caldav

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"caldo/internal/domain"
)

type TasksRepo struct {
	client *Client
}

func NewTasksRepo(client *Client) *TasksRepo {
	return &TasksRepo{client: client}
}

func (r *TasksRepo) ListTasks(ctx context.Context, serverURL, username, password string, collection Collection) ([]domain.Task, error) {
	_ = ctx
	if strings.TrimSpace(serverURL) == "" {
		return nil, fmt.Errorf("CalDAV server URL fehlt")
	}
	if !collection.SupportsVTODO {
		return nil, nil
	}

	now := time.Now().UTC()
	dueA := now.Add(48 * time.Hour)
	dueB := now.Add(96 * time.Hour)
	tasks := []domain.Task{
		{
			UID:             "demo-1",
			CollectionID:    collection.ID,
			CollectionHref:  collection.Href,
			Href:            collection.Href + "demo-1.ics",
			ETag:            "\"demo-1\"",
			Summary:         "CalDAV Discovery prüfen",
			Description:     "Principal/Home-Set/Collections erfolgreich ermitteln",
			Status:          "NEEDS-ACTION",
			Priority:        3,
			PercentComplete: 0,
			Categories:      []string{"setup", "caldav"},
			Due:             &dueA,
			DueKind:         "datetime",
			CreatedAt:       &now,
			UpdatedAt:       &now,
		},
		{
			UID:             "demo-2",
			CollectionID:    collection.ID,
			CollectionHref:  collection.Href,
			Href:            collection.Href + "demo-2.ics",
			ETag:            "\"demo-2\"",
			Summary:         "HTMX Taskliste anzeigen",
			Description:     "Sidebar + Tabelle als Partials rendern",
			Status:          "IN-PROCESS",
			Priority:        5,
			PercentComplete: 40,
			Categories:      []string{"ui"},
			Due:             &dueB,
			DueKind:         "datetime",
			CreatedAt:       &now,
			UpdatedAt:       &now,
		},
	}

	sort.Slice(tasks, func(i, j int) bool { return tasks[i].Summary < tasks[j].Summary })
	_ = username
	_ = password
	_ = r.client
	return tasks, nil
}
