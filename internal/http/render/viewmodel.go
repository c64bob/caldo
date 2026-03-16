package render

import (
	"fmt"
	"strings"
	"time"

	"caldo/internal/domain"
)

type TaskPageViewModel struct {
	PrincipalID    string
	Lists          []TaskListItem
	ActiveListID   string
	Rows           []TaskRow
	HasCredentials bool
	Error          string
}

type TaskListItem struct {
	ID          string
	DisplayName string
	Href        string
	IsActive    bool
}

type TaskRow struct {
	UID             string
	Summary         string
	Status          string
	Priority        int
	PercentComplete int
	Due             string
	Categories      string
}

func BuildTaskRows(tasks []domain.Task) []TaskRow {
	rows := make([]TaskRow, 0, len(tasks))
	for _, t := range tasks {
		rows = append(rows, TaskRow{
			UID:             t.UID,
			Summary:         t.Summary,
			Status:          t.Status,
			Priority:        t.Priority,
			PercentComplete: t.PercentComplete,
			Due:             formatDue(t.Due, t.DueKind),
			Categories:      strings.Join(t.Categories, ", "),
		})
	}
	return rows
}

func formatDue(due *time.Time, kind string) string {
	if due == nil {
		return ""
	}
	if kind == "date" {
		return due.Format("2006-01-02")
	}
	return due.Local().Format("2006-01-02 15:04")
}

func BuildTaskLists(lists []domain.List, activeListID string) []TaskListItem {
	out := make([]TaskListItem, 0, len(lists))
	for _, l := range lists {
		display := strings.TrimSpace(l.DisplayName)
		if display == "" {
			display = l.ID
		}
		out = append(out, TaskListItem{ID: l.ID, DisplayName: display, Href: l.Href, IsActive: l.ID == activeListID})
	}
	if len(out) == 0 {
		out = append(out, TaskListItem{ID: "tasks", DisplayName: "Tasks", IsActive: true})
	}
	return out
}

func (vm TaskPageViewModel) Title() string {
	for _, l := range vm.Lists {
		if l.IsActive {
			return fmt.Sprintf("Aufgaben – %s", l.DisplayName)
		}
	}
	return "Aufgaben"
}
