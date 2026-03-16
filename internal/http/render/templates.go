package render

import (
	"fmt"
	"html/template"
)

type Templates struct {
	taskPage       *template.Template
	tasksList      *template.Template
	sidebarPartial *template.Template
}

func LoadTemplates() (*Templates, error) {
	taskPage, err := template.ParseFiles(
		"web/templates/layout.gohtml",
		"web/templates/pages/tasks.gohtml",
		"web/templates/partials/sidebar_lists.gohtml",
		"web/templates/partials/task_table.gohtml",
		"web/templates/partials/task_row.gohtml",
		"web/templates/partials/flash.gohtml",
	)
	if err != nil {
		return nil, fmt.Errorf("parse task page templates: %w", err)
	}

	tasksList, err := template.ParseFiles(
		"web/templates/partials/task_table.gohtml",
		"web/templates/partials/task_row.gohtml",
	)
	if err != nil {
		return nil, fmt.Errorf("parse tasks list templates: %w", err)
	}

	sidebarPartial, err := template.ParseFiles("web/templates/partials/sidebar_lists.gohtml")
	if err != nil {
		return nil, fmt.Errorf("parse sidebar template: %w", err)
	}

	return &Templates{taskPage: taskPage, tasksList: tasksList, sidebarPartial: sidebarPartial}, nil
}

func (t *Templates) RenderTasksPage(wr interface{ Write([]byte) (int, error) }, vm TaskPageViewModel) error {
	return t.taskPage.ExecuteTemplate(wr, "layout", vm)
}

func (t *Templates) RenderTasksList(wr interface{ Write([]byte) (int, error) }, vm TaskPageViewModel) error {
	return t.tasksList.ExecuteTemplate(wr, "task_table", vm)
}

func (t *Templates) RenderSidebar(wr interface{ Write([]byte) (int, error) }, vm TaskPageViewModel) error {
	return t.sidebarPartial.ExecuteTemplate(wr, "sidebar_lists", vm)
}
