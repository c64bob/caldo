package model

import "testing"

func TestValidateTaskViewPreference(t *testing.T) {
	t.Parallel()

	valid := []TaskViewPreference{
		DefaultTaskViewPreference(TaskViewToday, ""),
		{ViewKind: TaskViewProject, ViewID: "project-1", SortBy: TaskSortPriority, SortOrder: TaskSortDescending, GroupBy: TaskGroupDue},
		{ViewKind: TaskViewSearch, SortBy: TaskSortName, SortOrder: TaskSortAscending, GroupBy: TaskGroupProject},
	}
	for _, preference := range valid {
		if err := ValidateTaskViewPreference(preference); err != nil {
			t.Fatalf("valid preference rejected: %#v: %v", preference, err)
		}
	}

	invalid := []TaskViewPreference{
		{ViewKind: "unknown", SortBy: TaskSortDefault, SortOrder: TaskSortAscending, GroupBy: TaskGroupNone},
		{ViewKind: TaskViewProject, SortBy: TaskSortDefault, SortOrder: TaskSortAscending, GroupBy: TaskGroupNone},
		{ViewKind: TaskViewToday, ViewID: "unexpected", SortBy: TaskSortDefault, SortOrder: TaskSortAscending, GroupBy: TaskGroupNone},
		{ViewKind: TaskViewToday, SortBy: "sql", SortOrder: TaskSortAscending, GroupBy: TaskGroupNone},
	}
	for _, preference := range invalid {
		if err := ValidateTaskViewPreference(preference); err == nil {
			t.Fatalf("invalid preference accepted: %#v", preference)
		}
	}
}
