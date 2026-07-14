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
	for _, index := range []int{1, 3, 4} {
		if !groups[0].Tasks[index].ParentVisible {
			t.Fatalf("bundled child %q must be marked with a visible parent", groups[0].Tasks[index].ID)
		}
	}
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

func TestBuildTaskListGroupsBundlesParentAndChildInDefaultOrder(t *testing.T) {
	t.Parallel()
	tasks := []TaskRowView{{ID: "child", ParentID: "parent", IsSubtask: true}, {ID: "parent"}}
	display := TaskListDisplayView{Preference: model.DefaultTaskViewPreference(model.TaskViewSearch, "")}
	groups := BuildTaskListGroups(tasks, display, time.Time{})
	assertTaskIDs(t, groups[0].Tasks, []string{"parent", "child"})
	if !groups[0].Tasks[1].ParentVisible {
		t.Fatal("child bundled with its parent must be marked with a visible parent")
	}
}

func TestBuildTaskListGroupsLeavesOrphanedResultsAtTopLevel(t *testing.T) {
	t.Parallel()
	tasks := []TaskRowView{
		{ID: "child-2", ParentID: "filtered-parent", ParentTitle: "Filtered parent", IsSubtask: true},
		{ID: "child-1", ParentID: "filtered-parent", ParentTitle: "Filtered parent", IsSubtask: true},
		{ID: "normal"},
	}
	display := TaskListDisplayView{Preference: model.TaskViewPreference{SortBy: model.TaskSortName, SortOrder: model.TaskSortAscending, GroupBy: model.TaskGroupPriority}}

	groups := BuildTaskListGroups(tasks, display, time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC))
	if len(groups) != 1 {
		t.Fatalf("group count=%d want=1", len(groups))
	}
	assertTaskIDs(t, groups[0].Tasks, []string{"child-2", "child-1", "normal"})
	for _, task := range groups[0].Tasks[:2] {
		if task.ParentVisible {
			t.Fatalf("orphaned child %q must not be marked with a visible parent", task.ID)
		}
	}
}

func TestBuildTaskListGroupsGroupsOrphanedSiblingsByTheirOwnPriority(t *testing.T) {
	t.Parallel()
	tasks := []TaskRowView{
		{ID: "child-none", ParentID: "filtered-parent", ParentTitle: "Filtered parent", IsSubtask: true},
		{ID: "child-p1", ParentID: "filtered-parent", ParentTitle: "Filtered parent", IsSubtask: true, Priority: 1, HasPriority: true},
		{ID: "child-p3", ParentID: "filtered-parent", ParentTitle: "Filtered parent", IsSubtask: true, Priority: 9, HasPriority: true},
	}
	display := TaskListDisplayView{Preference: model.TaskViewPreference{SortBy: model.TaskSortDefault, SortOrder: model.TaskSortAscending, GroupBy: model.TaskGroupPriority}}

	groups := BuildTaskListGroups(tasks, display, time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC))
	if len(groups) != 3 {
		t.Fatalf("group count=%d want=3: %#v", len(groups), groups)
	}
	if groups[0].Label != "P1" || groups[1].Label != "P3" || groups[2].Label != "Keine Priorität" {
		t.Fatalf("unexpected group labels: %#v", groups)
	}
	assertTaskIDs(t, groups[0].Tasks, []string{"child-p1"})
	assertTaskIDs(t, groups[1].Tasks, []string{"child-p3"})
	assertTaskIDs(t, groups[2].Tasks, []string{"child-none"})
	for _, group := range groups {
		if group.Tasks[0].ParentVisible {
			t.Fatalf("orphaned child %q must remain top-level", group.Tasks[0].ID)
		}
	}
}

func TestBuildTaskListGroupsSortsOrphanedSiblingsIndependently(t *testing.T) {
	t.Parallel()
	tasks := []TaskRowView{
		{ID: "child-later", ParentID: "filtered-parent", IsSubtask: true, DueISODate: "2026-07-12"},
		{ID: "child-earlier", ParentID: "filtered-parent", IsSubtask: true, DueISODate: "2026-07-11"},
	}
	display := TaskListDisplayView{Preference: model.TaskViewPreference{SortBy: model.TaskSortDue, SortOrder: model.TaskSortAscending, GroupBy: model.TaskGroupNone}}

	groups := BuildTaskListGroups(tasks, display, time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC))
	assertTaskIDs(t, groups[0].Tasks, []string{"child-earlier", "child-later"})
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
