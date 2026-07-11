package view

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"caldo/internal/model"
)

func TestBuildTaskListGroupsSortsParentsAndKeepsChildrenTogether(t *testing.T) {
	t.Parallel()
	tasks := []TaskRowView{
		{ID: "parent-z", Title: "Zulu", Priority: 7, HasPriority: true},
		{ID: "child-z-2", ParentID: "parent-z", IsSubtask: true, Title: "Second child", Priority: 1, HasPriority: true},
		{ID: "child-z-1", ParentID: "parent-z", IsSubtask: true, Title: "First child"},
		{ID: "parent-a", Title: "Alpha", Priority: 1, HasPriority: true},
		{ID: "child-a", ParentID: "parent-a", IsSubtask: true, Title: "Child"},
	}
	display := TaskListDisplayView{Preference: model.TaskViewPreference{SortBy: model.TaskSortName, SortOrder: model.TaskSortAscending, GroupBy: model.TaskGroupNone}}

	groups := BuildTaskListGroups(tasks, display, time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC))
	assertTaskIDs(t, groups[0].Tasks, []string{"parent-a", "child-a", "parent-z", "child-z-2", "child-z-1"})
}

func TestBuildTaskListGroupsUsesParentPriority(t *testing.T) {
	t.Parallel()
	tasks := []TaskRowView{
		{ID: "parent", Title: "Parent", Priority: 1, HasPriority: true},
		{ID: "child", ParentID: "parent", IsSubtask: true, Priority: 9, HasPriority: true},
		{ID: "normal", Title: "Normal"},
	}
	display := TaskListDisplayView{Preference: model.TaskViewPreference{SortBy: model.TaskSortDefault, SortOrder: model.TaskSortAscending, GroupBy: model.TaskGroupPriority}}

	groups := BuildTaskListGroups(tasks, display, time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC))
	if len(groups) != 2 || groups[0].Label != "P1" || groups[1].Label != "Keine Priorität" {
		t.Fatalf("unexpected groups: %#v", groups)
	}
	assertTaskIDs(t, groups[0].Tasks, []string{"parent", "child"})
}

func TestBuildTaskListGroupsKeepsProjectsWithDuplicateNamesSeparate(t *testing.T) {
	t.Parallel()
	tasks := []TaskRowView{
		{ID: "one", ProjectID: "project-1", ProjectName: "Work"},
		{ID: "two", ProjectID: "project-2", ProjectName: "Work"},
	}
	display := TaskListDisplayView{Preference: model.TaskViewPreference{SortBy: model.TaskSortDefault, SortOrder: model.TaskSortAscending, GroupBy: model.TaskGroupProject}}

	groups := BuildTaskListGroups(tasks, display, time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC))
	if len(groups) != 2 {
		t.Fatalf("group count=%d want=2", len(groups))
	}
}

func TestBuildTaskListGroupsBuildsDueDateBuckets(t *testing.T) {
	t.Parallel()
	tasks := []TaskRowView{
		{ID: "none"},
		{ID: "later", DueISODate: "2026-07-14"},
		{ID: "tomorrow", DueISODate: "2026-07-11"},
		{ID: "today", DueISODate: "2026-07-10"},
		{ID: "overdue", DueISODate: "2026-07-09"},
	}
	display := TaskListDisplayView{Preference: model.TaskViewPreference{SortBy: model.TaskSortDefault, SortOrder: model.TaskSortAscending, GroupBy: model.TaskGroupDue}}

	groups := BuildTaskListGroups(tasks, display, time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC))
	labels := make([]string, 0, len(groups))
	for _, group := range groups {
		labels = append(labels, group.Label)
	}
	want := []string{"Überfällig", "Heute", "Morgen", "14.07.2026", "Kein Datum"}
	if len(labels) != len(want) {
		t.Fatalf("labels=%#v want=%#v", labels, want)
	}
	for index := range want {
		if labels[index] != want[index] {
			t.Fatalf("labels=%#v want=%#v", labels, want)
		}
	}
}

func TestBuildTaskListGroupsLeavesDefaultOrderUntouched(t *testing.T) {
	t.Parallel()
	tasks := []TaskRowView{{ID: "child", ParentID: "parent", IsSubtask: true}, {ID: "parent"}}
	display := TaskListDisplayView{Preference: model.DefaultTaskViewPreference(model.TaskViewSearch, "")}
	groups := BuildTaskListGroups(tasks, display, time.Time{})
	assertTaskIDs(t, groups[0].Tasks, []string{"child", "parent"})
}

func TestBuildTaskListGroupsReversesOnlyPrimarySortKey(t *testing.T) {
	t.Parallel()
	tasks := []TaskRowView{
		{ID: "none-a"},
		{ID: "later", DueISODate: "2026-07-12"},
		{ID: "none-b"},
		{ID: "earlier", DueISODate: "2026-07-11"},
	}
	display := TaskListDisplayView{Preference: model.TaskViewPreference{SortBy: model.TaskSortDue, SortOrder: model.TaskSortDescending, GroupBy: model.TaskGroupNone}}

	groups := BuildTaskListGroups(tasks, display, time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC))
	assertTaskIDs(t, groups[0].Tasks, []string{"none-a", "none-b", "later", "earlier"})
}

func TestTaskListDisplayControlsRenderScopeAndHideIrrelevantGrouping(t *testing.T) {
	t.Parallel()
	display := TaskListDisplayView{
		Preference: model.TaskViewPreference{
			ViewKind:  model.TaskViewProject,
			ViewID:    "project-1",
			SortBy:    model.TaskSortName,
			SortOrder: model.TaskSortDescending,
			GroupBy:   model.TaskGroupPriority,
		},
		AllowProjectGrouping: false,
		AllowDueDateGrouping: true,
	}

	var rendered bytes.Buffer
	ctx := WithCSRFToken(context.Background(), "token-123")
	if err := TaskListDisplayControls(display).Render(ctx, &rendered); err != nil {
		t.Fatalf("render display controls: %v", err)
	}
	output := rendered.String()
	for _, want := range []string{
		`data-task-display`,
		`action="/task-view-preferences"`,
		`name="view_kind" value="project"`,
		`name="view_id" value="project-1"`,
		`name="sort_by"`,
		`value="name" selected`,
		`name="sort_order"`,
		`value="desc" selected`,
		`name="group_by"`,
		`value="priority" selected`,
		`Name · Absteigend`,
		`X-CSRF-Token&#34;:&#34;token-123`,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected controls to include %q in %s", want, output)
		}
	}
	if strings.Contains(output, `<option value="project"`) {
		t.Fatalf("project view must not offer project grouping: %s", output)
	}
}

func assertTaskIDs(t *testing.T, tasks []TaskRowView, want []string) {
	t.Helper()
	if len(tasks) != len(want) {
		t.Fatalf("task count=%d want=%d", len(tasks), len(want))
	}
	for index := range want {
		if tasks[index].ID != want[index] {
			t.Fatalf("task ids at %d=%q want=%q", index, tasks[index].ID, want[index])
		}
	}
}
