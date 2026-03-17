package render

import "testing"

func TestTaskPageViewModel_ShowColumn_MissingColumnHiddenWhenPrefsPresent(t *testing.T) {
	vm := TaskPageViewModel{VisibleColumns: map[string]bool{"name": true, "due": true}}
	if vm.ShowColumn("priority") {
		t.Fatal("expected missing column to be hidden when visibility prefs are present")
	}
	if !vm.ShowColumn("name") {
		t.Fatal("expected explicitly enabled column to be visible")
	}
}

func TestTaskPageViewModel_ShowColumn_DefaultsVisibleWithoutPrefs(t *testing.T) {
	vm := TaskPageViewModel{}
	if !vm.ShowColumn("priority") {
		t.Fatal("expected visible when no visibility prefs are present")
	}
}
