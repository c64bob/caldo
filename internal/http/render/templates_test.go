package render

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveTemplateRoot_UsesEnvironmentOverride(t *testing.T) {
	tmp := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmp, "layout.gohtml"), []byte("{{define \"layout\"}}{{end}}"), 0o600); err != nil {
		t.Fatalf("write template: %v", err)
	}
	t.Setenv("CALDO_TEMPLATE_DIR", tmp)

	root, err := resolveTemplateRoot()
	if err != nil {
		t.Fatalf("resolve template root: %v", err)
	}
	if root != tmp {
		t.Fatalf("expected %q, got %q", tmp, root)
	}
}
