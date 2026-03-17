package render

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveTemplateRoot_IgnoresEnvironmentOverride(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "web", "templates")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("mkdir templates: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "layout.gohtml"), []byte("{{define \"layout\"}}{{end}}"), 0o600); err != nil {
		t.Fatalf("write layout template: %v", err)
	}

	envOverride := t.TempDir()
	if err := os.WriteFile(filepath.Join(envOverride, "layout.gohtml"), []byte("{{define \"layout\"}}{{end}}"), 0o600); err != nil {
		t.Fatalf("write override template: %v", err)
	}
	// Wird bewusst ignoriert: Template-Pfad bleibt einheitlich über Standard-/Autodetect-Logik.
	t.Setenv("CALDO_TEMPLATE_DIR", envOverride)
	t.Chdir(base)

	resolved, err := resolveTemplateRoot()
	if err != nil {
		t.Fatalf("resolve template root: %v", err)
	}
	if resolved != root {
		t.Fatalf("expected %q, got %q", root, resolved)
	}
}
