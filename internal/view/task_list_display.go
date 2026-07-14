package view

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"time"

	"caldo/internal/model"
)

// TaskListDisplayView describes the display controls for one task-list scope.
type TaskListDisplayView struct {
	Preference           model.TaskViewPreference
	SearchQuery          string
	Language             string
	AllowProjectGrouping bool
	AllowDueDateGrouping bool
}

// TaskListGroupView is one optional display group containing ordered task rows.
type TaskListGroupView struct {
	Key   string
	Label string
	Tasks []TaskRowView
}

type taskListBundle struct {
	representative TaskRowView
	parent         *TaskRowView
	children       []TaskRowView
	firstIndex     int
}

type taskListGroup struct {
	TaskListGroupView
	rank  int
	value string
}

func taskListSortValues() []string {
	return []string{model.TaskSortDefault, model.TaskSortDue, model.TaskSortPriority, model.TaskSortName, model.TaskSortAdded}
}

func taskListGroupValues(display TaskListDisplayView) []string {
	values := []string{model.TaskGroupNone}
	if display.AllowProjectGrouping {
		values = append(values, model.TaskGroupProject)
	}
	if display.AllowDueDateGrouping {
		values = append(values, model.TaskGroupDue)
	}
	return append(values, model.TaskGroupAdded, model.TaskGroupPriority)
}

func taskListDisplayLabel(ctx context.Context, value string) string {
	english := UILanguage(ctx) == "en"
	labelsDE := map[string]string{
		model.TaskSortDefault:    "Standard",
		model.TaskSortDue:        "Fälligkeitsdatum",
		model.TaskSortPriority:   "Priorität",
		model.TaskSortName:       "Name",
		model.TaskSortAdded:      "Hinzufügedatum",
		model.TaskGroupNone:      "Keine",
		model.TaskGroupProject:   "Projekt",
		model.TaskSortAscending:  "Aufsteigend",
		model.TaskSortDescending: "Absteigend",
	}
	labelsEN := map[string]string{
		model.TaskSortDefault:    "Default",
		model.TaskSortDue:        "Due date",
		model.TaskSortPriority:   "Priority",
		model.TaskSortName:       "Name",
		model.TaskSortAdded:      "Date added",
		model.TaskGroupNone:      "None",
		model.TaskGroupProject:   "Project",
		model.TaskSortAscending:  "Ascending",
		model.TaskSortDescending: "Descending",
	}
	if english {
		if label := labelsEN[value]; label != "" {
			return label
		}
	}
	if label := labelsDE[value]; label != "" {
		return label
	}
	return value
}

func taskListDisplaySummary(ctx context.Context, display TaskListDisplayView) string {
	parts := make([]string, 0, 3)
	if display.Preference.GroupBy != model.TaskGroupNone {
		parts = append(parts, taskListDisplayLabel(ctx, display.Preference.GroupBy))
	}
	if display.Preference.SortBy != model.TaskSortDefault {
		parts = append(parts, taskListDisplayLabel(ctx, display.Preference.SortBy), taskListDisplayLabel(ctx, display.Preference.SortOrder))
	}
	return strings.Join(parts, " · ")
}

func taskListDisplayIsDefault(display TaskListDisplayView) bool {
	return display.Preference.SortBy == model.TaskSortDefault && display.Preference.GroupBy == model.TaskGroupNone
}

func taskListDisplayOrderVisible(display TaskListDisplayView) bool {
	return display.Preference.SortBy != model.TaskSortDefault
}

func taskListDisplayControlTitle(ctx context.Context) string {
	if UILanguage(ctx) == "en" {
		return "Display options"
	}
	return "Anzeigeoptionen"
}

func taskListDisplayFieldLabel(ctx context.Context, field string) string {
	english := UILanguage(ctx) == "en"
	switch field {
	case "group":
		if english {
			return "Group by"
		}
		return "Gruppieren nach"
	case "sort":
		if english {
			return "Sort by"
		}
		return "Sortieren nach"
	case "order":
		if english {
			return "Order"
		}
		return "Reihenfolge"
	default:
		return field
	}
}

func taskListDisplayActionLabel(ctx context.Context, action string) string {
	english := UILanguage(ctx) == "en"
	switch action {
	case "apply":
		if english {
			return "Apply"
		}
		return "Anwenden"
	case "reset":
		if english {
			return "Reset"
		}
		return "Zurücksetzen"
	default:
		return action
	}
}

// BuildTaskListGroups applies display-only sorting and grouping to parent-task bundles.
func BuildTaskListGroups(tasks []TaskRowView, display TaskListDisplayView, referenceDate time.Time) []TaskListGroupView {
	if len(tasks) == 0 {
		return nil
	}
	preference := display.Preference
	bundles := buildTaskListBundles(tasks)
	if preference.SortBy == model.TaskSortDefault && preference.GroupBy == model.TaskGroupNone {
		return []TaskListGroupView{{Tasks: flattenTaskListBundles(bundles)}}
	}

	if preference.SortBy != model.TaskSortDefault {
		sort.SliceStable(bundles, func(i, j int) bool {
			comparison := compareTaskListBundles(bundles[i], bundles[j], preference.SortBy)
			if preference.SortOrder == model.TaskSortDescending {
				comparison = -comparison
			}
			if comparison == 0 {
				return bundles[i].firstIndex < bundles[j].firstIndex
			}
			return comparison < 0
		})
	}

	if preference.GroupBy == model.TaskGroupNone {
		return []TaskListGroupView{{Tasks: flattenTaskListBundles(bundles)}}
	}

	groupsByKey := make(map[string]*taskListGroup)
	groups := make([]*taskListGroup, 0)
	for _, bundle := range bundles {
		identity := taskListGroupFor(bundle.representative, preference.GroupBy, referenceDate, display.Language == "en")
		group := groupsByKey[identity.Key]
		if group == nil {
			group = &identity
			groupsByKey[identity.Key] = group
			groups = append(groups, group)
		}
		group.Tasks = append(group.Tasks, bundleRows(bundle)...)
	}

	sort.SliceStable(groups, func(i, j int) bool {
		if groups[i].rank != groups[j].rank {
			return groups[i].rank < groups[j].rank
		}
		if preference.GroupBy == model.TaskGroupAdded {
			return groups[i].value > groups[j].value
		}
		return groups[i].value < groups[j].value
	})

	result := make([]TaskListGroupView, 0, len(groups))
	for _, group := range groups {
		result = append(result, group.TaskListGroupView)
	}
	return result
}

func buildTaskListBundles(tasks []TaskRowView) []taskListBundle {
	bundles := make([]taskListBundle, 0, len(tasks))
	indexByKey := make(map[string]int, len(tasks))
	visibleParents := make(map[string]struct{}, len(tasks))
	for _, task := range tasks {
		if !task.IsSubtask && strings.TrimSpace(task.ID) != "" {
			visibleParents[task.ID] = struct{}{}
		}
	}
	for index, task := range tasks {
		key := task.ID
		if task.IsSubtask && strings.TrimSpace(task.ParentID) != "" {
			if _, ok := visibleParents[task.ParentID]; ok {
				key = task.ParentID
			}
		}
		bundleIndex, ok := indexByKey[key]
		if !ok {
			bundleIndex = len(bundles)
			indexByKey[key] = bundleIndex
			bundles = append(bundles, taskListBundle{representative: task, firstIndex: index})
		}
		bundle := &bundles[bundleIndex]
		if task.IsSubtask {
			bundle.children = append(bundle.children, task)
			continue
		}
		parent := task
		bundle.parent = &parent
		bundle.representative = task
	}
	return bundles
}

func bundleRows(bundle taskListBundle) []TaskRowView {
	rows := make([]TaskRowView, 0, len(bundle.children)+1)
	parentVisible := bundle.parent != nil
	if bundle.parent != nil {
		rows = append(rows, *bundle.parent)
	}
	for _, child := range bundle.children {
		child.ParentVisible = parentVisible
		rows = append(rows, child)
	}
	return rows
}

func flattenTaskListBundles(bundles []taskListBundle) []TaskRowView {
	rows := make([]TaskRowView, 0)
	for _, bundle := range bundles {
		rows = append(rows, bundleRows(bundle)...)
	}
	return rows
}

func compareTaskListBundles(left, right taskListBundle, sortBy string) int {
	a := left.representative
	b := right.representative
	switch sortBy {
	case model.TaskSortDue:
		return compareOptionalStrings(normalizedISODate(a.DueISODate), normalizedISODate(b.DueISODate))
	case model.TaskSortPriority:
		return compareOptionalInts(a.Priority, a.HasPriority, b.Priority, b.HasPriority)
	case model.TaskSortName:
		return strings.Compare(strings.ToLower(strings.TrimSpace(a.Title)), strings.ToLower(strings.TrimSpace(b.Title)))
	case model.TaskSortAdded:
		return compareOptionalStrings(normalizedCreatedAt(a.CreatedAt), normalizedCreatedAt(b.CreatedAt))
	default:
		return 0
	}
}

func compareOptionalStrings(left, right string) int {
	if left == "" && right == "" {
		return 0
	}
	if left == "" {
		return 1
	}
	if right == "" {
		return -1
	}
	return strings.Compare(left, right)
}

func compareOptionalInts(left int, leftSet bool, right int, rightSet bool) int {
	if !leftSet && !rightSet {
		return 0
	}
	if !leftSet {
		return 1
	}
	if !rightSet {
		return -1
	}
	return left - right
}

func taskListGroupFor(task TaskRowView, groupBy string, referenceDate time.Time, english bool) taskListGroup {
	switch groupBy {
	case model.TaskGroupProject:
		label := strings.TrimSpace(task.ProjectName)
		key := strings.TrimSpace(task.ProjectID)
		if label == "" {
			label = taskListGroupStaticLabel(english, "without_project")
		}
		if key == "" {
			key = "none:" + strings.ToLower(label)
		}
		return newTaskListGroup("project:"+key, label, 0, strings.ToLower(label))
	case model.TaskGroupDue:
		return taskListDueGroup(task.DueISODate, referenceDate, english)
	case model.TaskGroupAdded:
		return taskListAddedGroup(task.CreatedAt, referenceDate, english)
	case model.TaskGroupPriority:
		return taskListPriorityGroup(task, english)
	default:
		return newTaskListGroup("all", "", 0, "")
	}
}

func taskListDueGroup(rawDate string, referenceDate time.Time, english bool) taskListGroup {
	date := normalizedISODate(rawDate)
	today := referenceDate.UTC().Format("2006-01-02")
	tomorrow := referenceDate.UTC().AddDate(0, 0, 1).Format("2006-01-02")
	switch {
	case date == "":
		return newTaskListGroup("due:none", taskListGroupStaticLabel(english, "no_date"), 4, "")
	case date < today:
		return newTaskListGroup("due:overdue", taskListGroupStaticLabel(english, "overdue"), 0, "")
	case date == today:
		return newTaskListGroup("due:today", taskListGroupStaticLabel(english, "today"), 1, date)
	case date == tomorrow:
		return newTaskListGroup("due:tomorrow", taskListGroupStaticLabel(english, "tomorrow"), 2, date)
	default:
		return newTaskListGroup("due:"+date, germanDateLabel(date), 3, date)
	}
}

func taskListAddedGroup(rawCreatedAt string, referenceDate time.Time, english bool) taskListGroup {
	date := normalizedCreatedDate(rawCreatedAt)
	today := referenceDate.UTC().Format("2006-01-02")
	yesterday := referenceDate.UTC().AddDate(0, 0, -1).Format("2006-01-02")
	switch date {
	case "":
		return newTaskListGroup("added:none", taskListGroupStaticLabel(english, "unknown"), 1, "")
	case today:
		return newTaskListGroup("added:"+date, taskListGroupStaticLabel(english, "today"), 0, date)
	case yesterday:
		return newTaskListGroup("added:"+date, taskListGroupStaticLabel(english, "yesterday"), 0, date)
	default:
		return newTaskListGroup("added:"+date, germanDateLabel(date), 0, date)
	}
}

func taskListPriorityGroup(task TaskRowView, english bool) taskListGroup {
	switch {
	case !task.HasPriority || task.Priority <= 0:
		return newTaskListGroup("priority:none", taskListGroupStaticLabel(english, "no_priority"), 3, "")
	case task.Priority <= 4:
		return newTaskListGroup("priority:p1", "P1", 0, "")
	case task.Priority <= 6:
		return newTaskListGroup("priority:p2", "P2", 1, "")
	default:
		return newTaskListGroup("priority:p3", "P3", 2, "")
	}
}

func taskListGroupStaticLabel(english bool, key string) string {
	labelsDE := map[string]string{
		"without_project": "Ohne Projekt",
		"no_date":         "Kein Datum",
		"overdue":         "Überfällig",
		"today":           "Heute",
		"tomorrow":        "Morgen",
		"yesterday":       "Gestern",
		"unknown":         "Unbekannt",
		"no_priority":     "Keine Priorität",
	}
	labelsEN := map[string]string{
		"without_project": "No project",
		"no_date":         "No date",
		"overdue":         "Overdue",
		"today":           "Today",
		"tomorrow":        "Tomorrow",
		"yesterday":       "Yesterday",
		"unknown":         "Unknown",
		"no_priority":     "No priority",
	}
	if english {
		return labelsEN[key]
	}
	return labelsDE[key]
}

func newTaskListGroup(key, label string, rank int, value string) taskListGroup {
	return taskListGroup{TaskListGroupView: TaskListGroupView{Key: key, Label: label}, rank: rank, value: value}
}

func normalizedISODate(value string) string {
	trimmed := strings.TrimSpace(value)
	if len(trimmed) >= 10 {
		trimmed = trimmed[:10]
	}
	if _, err := time.Parse("2006-01-02", trimmed); err != nil {
		return ""
	}
	return trimmed
}

func normalizedCreatedAt(value string) string {
	return strings.TrimSpace(strings.ReplaceAll(value, " ", "T"))
}

func normalizedCreatedDate(value string) string {
	createdAt := normalizedCreatedAt(value)
	if len(createdAt) < 10 {
		return ""
	}
	return normalizedISODate(createdAt[:10])
}

func germanDateLabel(isoDate string) string {
	parsed, err := time.Parse("2006-01-02", isoDate)
	if err != nil {
		return isoDate
	}
	return parsed.Format("02.01.2006")
}

func taskListGroupCountLabel(group TaskListGroupView) string {
	return strconv.Itoa(len(group.Tasks))
}
