package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"caldo/internal/db"
	"caldo/internal/model"
)

func TestTaskViewPreferenceUpdatePersistsAndRedirects(t *testing.T) {
	t.Parallel()
	database, err := db.OpenSQLite(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })

	form := url.Values{
		"view_kind":  {model.TaskViewLabel},
		"view_id":    {"label-1"},
		"sort_by":    {model.TaskSortPriority},
		"sort_order": {model.TaskSortDescending},
		"group_by":   {model.TaskGroupProject},
	}
	req := httptest.NewRequest(http.MethodPost, "/task-view-preferences", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	TaskViewPreferenceUpdate(database).ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther || rr.Header().Get("Location") != "/labels/label-1" {
		t.Fatalf("status=%d location=%q", rr.Code, rr.Header().Get("Location"))
	}
	preference, err := database.LoadTaskViewPreference(context.Background(), model.TaskViewLabel, "label-1")
	if err != nil {
		t.Fatal(err)
	}
	if preference.SortBy != model.TaskSortPriority || preference.SortOrder != model.TaskSortDescending || preference.GroupBy != model.TaskGroupProject {
		t.Fatalf("unexpected preference: %#v", preference)
	}
}

func TestTaskViewPreferenceUpdateRejectsProjectGroupingInsideProject(t *testing.T) {
	t.Parallel()
	database, err := db.OpenSQLite(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })

	form := url.Values{
		"view_kind":  {model.TaskViewProject},
		"view_id":    {"project-1"},
		"sort_by":    {model.TaskSortDefault},
		"sort_order": {model.TaskSortAscending},
		"group_by":   {model.TaskGroupProject},
	}
	req := httptest.NewRequest(http.MethodPost, "/task-view-preferences", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	TaskViewPreferenceUpdate(database).ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%q", rr.Code, rr.Body.String())
	}
}

func TestTaskViewPreferenceUpdateResetsAndPreservesSearchQuery(t *testing.T) {
	t.Parallel()
	database, err := db.OpenSQLite(filepath.Join(t.TempDir(), "caldo.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	if err := database.SaveTaskViewPreference(context.Background(), model.TaskViewPreference{
		ViewKind: model.TaskViewSearch, SortBy: model.TaskSortName, SortOrder: model.TaskSortAscending, GroupBy: model.TaskGroupProject,
	}); err != nil {
		t.Fatal(err)
	}

	form := url.Values{
		"view_kind":    {model.TaskViewSearch},
		"sort_by":      {model.TaskSortName},
		"sort_order":   {model.TaskSortAscending},
		"group_by":     {model.TaskGroupProject},
		"search_query": {"invoice #Work"},
		"action":       {"reset"},
	}
	req := httptest.NewRequest(http.MethodPost, "/task-view-preferences", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	rr := httptest.NewRecorder()
	TaskViewPreferenceUpdate(database).ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent || rr.Header().Get("HX-Redirect") != "/search?q=invoice+%23Work" {
		t.Fatalf("status=%d redirect=%q", rr.Code, rr.Header().Get("HX-Redirect"))
	}
	preference, err := database.LoadTaskViewPreference(context.Background(), model.TaskViewSearch, "")
	if err != nil {
		t.Fatal(err)
	}
	if preference.SortBy != model.TaskSortDefault || preference.GroupBy != model.TaskGroupNone {
		t.Fatalf("unexpected reset preference: %#v", preference)
	}
}
