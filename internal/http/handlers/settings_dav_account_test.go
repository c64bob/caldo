package handlers

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSettingsTemplate_UsesResolvedTemplateRoot(t *testing.T) {
	base := t.TempDir()
	templatesDir := filepath.Join(base, "web", "templates")
	if err := os.MkdirAll(filepath.Join(templatesDir, "pages"), 0o755); err != nil {
		t.Fatalf("mkdir templates: %v", err)
	}
	if err := os.WriteFile(filepath.Join(templatesDir, "layout.gohtml"), []byte("{{define \"layout\"}}{{end}}"), 0o600); err != nil {
		t.Fatalf("write layout: %v", err)
	}
	if err := os.WriteFile(filepath.Join(templatesDir, "pages", "settings.gohtml"), []byte("{{define \"settings_page\"}}ok{{end}}"), 0o600); err != nil {
		t.Fatalf("write settings: %v", err)
	}

	origWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(base); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(origWD) })

	tpl := loadSettingsTemplate()
	if tpl.Lookup("settings_page") == nil {
		t.Fatal("expected settings_page template to be loaded from resolved template root")
	}
}
