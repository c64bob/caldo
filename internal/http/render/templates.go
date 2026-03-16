package render

import (
	"fmt"
	"html/template"
	"os"
	"path/filepath"
)

type Templates struct {
	taskPage       *template.Template
	tasksList      *template.Template
	sidebarPartial *template.Template
}

func LoadTemplates() (*Templates, error) {
	templateRoot, err := resolveTemplateRoot()
	if err != nil {
		return nil, err
	}

	taskPage, err := template.ParseFiles(
		templateFile(templateRoot, "layout.gohtml"),
		templateFile(templateRoot, "pages/tasks.gohtml"),
		templateFile(templateRoot, "partials/sidebar_lists.gohtml"),
		templateFile(templateRoot, "partials/task_table.gohtml"),
		templateFile(templateRoot, "partials/task_row.gohtml"),
		templateFile(templateRoot, "partials/flash.gohtml"),
	)
	if err != nil {
		return nil, fmt.Errorf("parse task page templates: %w", err)
	}

	tasksList, err := template.ParseFiles(
		templateFile(templateRoot, "partials/task_table.gohtml"),
		templateFile(templateRoot, "partials/task_row.gohtml"),
		templateFile(templateRoot, "partials/sidebar_lists.gohtml"),
		templateFile(templateRoot, "partials/tasks_list_response.gohtml"),
	)
	if err != nil {
		return nil, fmt.Errorf("parse tasks list templates: %w", err)
	}

	sidebarPartial, err := template.ParseFiles(templateFile(templateRoot, "partials/sidebar_lists.gohtml"))
	if err != nil {
		return nil, fmt.Errorf("parse sidebar template: %w", err)
	}

	return &Templates{taskPage: taskPage, tasksList: tasksList, sidebarPartial: sidebarPartial}, nil
}

func (t *Templates) RenderTasksPage(wr interface{ Write([]byte) (int, error) }, vm TaskPageViewModel) error {
	return t.taskPage.ExecuteTemplate(wr, "layout", vm)
}

func (t *Templates) RenderTasksList(wr interface{ Write([]byte) (int, error) }, vm TaskPageViewModel) error {
	return t.tasksList.ExecuteTemplate(wr, "tasks_list_response", vm)
}

func (t *Templates) RenderSidebar(wr interface{ Write([]byte) (int, error) }, vm TaskPageViewModel) error {
	return t.sidebarPartial.ExecuteTemplate(wr, "sidebar_lists", vm)
}

func templateFile(root, name string) string {
	return filepath.Join(root, filepath.FromSlash(name))
}

func resolveTemplateRoot() (string, error) {
	candidates := make([]string, 0, 4)
	if wd, err := os.Getwd(); err == nil {
		candidates = append(candidates, filepath.Join(wd, "web", "templates"))
	}
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		candidates = append(candidates,
			filepath.Join(exeDir, "web", "templates"),
			filepath.Join(exeDir, "..", "web", "templates"),
		)
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(filepath.Join(candidate, "layout.gohtml")); err == nil {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("template-Verzeichnis nicht gefunden")
}
