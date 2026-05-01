package assets

import (
	"path/filepath"
	"testing"
)

func TestLoadManifest(t *testing.T) {
	t.Parallel()

	manifest, err := LoadManifest(filepath.Join("..", "..", "web", "static", "manifest.json"))
	if err != nil {
		t.Fatalf("LoadManifest returned error: %v", err)
	}

	got, err := manifest.Resolve("app.css")
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if got != "app.8f3a1c2.css" {
		t.Fatalf("unexpected manifest mapping: got %q", got)
	}
}

func TestLoadManifestFailsForMissingFile(t *testing.T) {
	t.Parallel()

	if _, err := LoadManifest(filepath.Join(t.TempDir(), "manifest.json")); err == nil {
		t.Fatal("expected error for missing manifest")
	}
}

func TestResolveFailsForUnknownAsset(t *testing.T) {
	t.Parallel()

	manifest := Manifest{"app.css": "app.8f3a1c2.css"}
	if _, err := manifest.Resolve("unknown.css"); err == nil {
		t.Fatal("expected error for unknown asset")
	}
}
