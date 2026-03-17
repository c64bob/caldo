package render

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"caldo/internal/domain"
)

type TaskPageViewModel struct {
	PrincipalID    string
	Lists          []TaskListItem
	Contexts       []string
	Goals          []string
	ActiveListID   string
	ActiveView     string
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
	ParentUID       string
	Goal            string
	ListID          string
	Href            string
	ETag            string
	Summary         string
	Description     string
	Status          string
	Priority        int
	PercentComplete int
	Due             string
	DueInput        string
	Categories      string
	CategoryItems   []string
	Folder          string
	Context         string
	IsCompleted     bool
	IsStarred       bool
	IsSubtask       bool
	SubtaskTotal    int
	SubtaskDone     int
	PriorityClass   string
}

func BuildTaskRows(tasks []domain.Task, lists []domain.List) []TaskRow {
	foldersByID := make(map[string]string, len(lists))
	for _, l := range lists {
		name := strings.TrimSpace(l.DisplayName)
		if name == "" {
			name = l.ID
		}
		foldersByID[l.ID] = name
	}

	flatRows := make([]TaskRow, 0, len(tasks))
	byParent := make(map[string][]TaskRow)
	for _, t := range tasks {
		starred, categories := splitStarCategory(t.Categories)
		row := TaskRow{
			UID:             t.UID,
			ParentUID:       strings.TrimSpace(t.ParentUID),
			Goal:            strings.TrimSpace(t.Goal),
			ListID:          t.CollectionID,
			Href:            t.Href,
			ETag:            t.ETag,
			Summary:         t.Summary,
			Description:     t.Description,
			Status:          t.Status,
			Priority:        t.Priority,
			PercentComplete: t.PercentComplete,
			Due:             formatDue(t.Due, t.DueKind),
			DueInput:        formatDueInput(t.Due, t.DueKind),
			Categories:      strings.Join(categories, ", "),
			CategoryItems:   categories,
			Folder:          foldersByID[t.CollectionID],
			Context:         deriveContext(categories),
			IsCompleted:     strings.EqualFold(t.Status, "completed"),
			IsStarred:       starred,
			IsSubtask:       strings.TrimSpace(t.ParentUID) != "",
			PriorityClass:   priorityClass(t.Priority),
		}
		flatRows = append(flatRows, row)
		if row.IsSubtask {
			byParent[row.ParentUID] = append(byParent[row.ParentUID], row)
		}
	}

	for i := range flatRows {
		flatRows[i].SubtaskTotal = len(byParent[flatRows[i].UID])
		done := 0
		for _, sub := range byParent[flatRows[i].UID] {
			if sub.IsCompleted {
				done++
			}
		}
		flatRows[i].SubtaskDone = done
	}

	sort.Slice(flatRows, func(i, j int) bool {
		if flatRows[i].Summary == flatRows[j].Summary {
			return flatRows[i].UID < flatRows[j].UID
		}
		return flatRows[i].Summary < flatRows[j].Summary
	})

	ordered := make([]TaskRow, 0, len(flatRows))
	for _, row := range flatRows {
		if row.IsSubtask {
			continue
		}
		ordered = append(ordered, row)
		children := byParent[row.UID]
		sort.Slice(children, func(i, j int) bool { return children[i].Summary < children[j].Summary })
		ordered = append(ordered, children...)
	}
	for _, row := range flatRows {
		if row.IsSubtask {
			if _, hasParent := byParent[row.ParentUID]; !hasParent {
				ordered = append(ordered, row)
			}
		}
	}
	return ordered
}

func BuildContexts(rows []TaskRow) []string {
	return uniqueFromRows(rows, func(r TaskRow) string { return r.Context })
}
func BuildGoals(rows []TaskRow) []string {
	return uniqueFromRows(rows, func(r TaskRow) string { return r.Goal })
}

func uniqueFromRows(rows []TaskRow, getter func(TaskRow) string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0)
	for _, row := range rows {
		v := strings.TrimSpace(getter(row))
		if v == "" || v == "—" {
			continue
		}
		if _, ok := seen[strings.ToLower(v)]; ok {
			continue
		}
		seen[strings.ToLower(v)] = struct{}{}
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}

func splitStarCategory(categories []string) (bool, []string) {
	out := make([]string, 0, len(categories))
	starred := false
	for _, c := range categories {
		trimmed := strings.TrimSpace(c)
		if strings.EqualFold(trimmed, "starred") {
			starred = true
			continue
		}
		out = append(out, trimmed)
	}
	return starred, out
}

func deriveContext(categories []string) string {
	for _, c := range categories {
		trimmed := strings.TrimSpace(c)
		if strings.HasPrefix(trimmed, "@") {
			return trimmed
		}
	}
	return "—"
}

func priorityClass(priority int) string {
	switch {
	case priority >= 8:
		return "priority-top"
	case priority >= 6:
		return "priority-high"
	case priority >= 4:
		return "priority-medium"
	case priority >= 2:
		return "priority-low"
	default:
		return "priority-negative"
	}
}

func formatDueInput(due *time.Time, kind string) string {
	if due == nil {
		return ""
	}
	if kind == "date" {
		return due.Format("2006-01-02")
	}
	return due.Local().Format("2006-01-02T15:04")
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
	out := make([]TaskListItem, 0, len(lists)+1)
	out = append(out, TaskListItem{ID: "all", DisplayName: "Alle Tasks", IsActive: activeListID == "all"})
	for _, l := range lists {
		display := strings.TrimSpace(l.DisplayName)
		if display == "" {
			display = l.ID
		}
		out = append(out, TaskListItem{ID: l.ID, DisplayName: display, Href: l.Href, IsActive: l.ID == activeListID})
	}
	if len(out) == 1 {
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
