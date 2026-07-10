package assets

import (
	"crypto/sha256"
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadManifest(t *testing.T) {
	t.Parallel()

	manifestPath := filepath.Join("..", "..", "web", "static", "manifest.json")
	manifest, err := LoadManifest(manifestPath)
	if err != nil {
		t.Fatalf("LoadManifest returned error: %v", err)
	}

	got, err := manifest.Resolve("app.css")
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if !strings.HasPrefix(got, "app.") || !strings.HasSuffix(got, ".css") {
		t.Fatalf("unexpected app css mapping: got %q", got)
	}

	for _, resolved := range manifest {
		assertAssetNameContainsHashPrefix(t, filepath.Dir(manifestPath), resolved)
	}
}

func TestManifestIncludesFaviconSet(t *testing.T) {
	t.Parallel()

	manifestPath := filepath.Join("..", "..", "web", "static", "manifest.json")
	manifest, err := LoadManifest(manifestPath)
	if err != nil {
		t.Fatalf("LoadManifest returned error: %v", err)
	}

	for logicalName, wantSize := range map[string]int{
		"favicon.png":          32,
		"apple-touch-icon.png": 180,
	} {
		resolved, err := manifest.Resolve(logicalName)
		if err != nil {
			t.Fatalf("Resolve(%q) returned error: %v", logicalName, err)
		}
		file, err := os.Open(filepath.Join(filepath.Dir(manifestPath), resolved))
		if err != nil {
			t.Fatalf("open %q: %v", logicalName, err)
		}
		config, err := png.DecodeConfig(file)
		closeErr := file.Close()
		if err != nil {
			t.Fatalf("decode %q: %v", logicalName, err)
		}
		if closeErr != nil {
			t.Fatalf("close %q: %v", logicalName, closeErr)
		}
		if config.Width != wantSize || config.Height != wantSize {
			t.Fatalf("unexpected %q dimensions: got %dx%d want %dx%d", logicalName, config.Width, config.Height, wantSize, wantSize)
		}
	}

	if _, err := manifest.Resolve("favicon.svg"); err != nil {
		t.Fatalf("Resolve favicon.svg returned error: %v", err)
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

	manifest := Manifest{"app.css": "app.hash.css"}
	if _, err := manifest.Resolve("unknown.css"); err == nil {
		t.Fatal("expected error for unknown asset")
	}
}

func TestLoadManifestFailsWhenMappedAssetDoesNotExist(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "manifest.json")
	if err := os.WriteFile(manifestPath, []byte(`{"app.css":"app.hash.css"}`), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	if _, err := LoadManifest(manifestPath); err == nil {
		t.Fatal("expected error for missing asset file")
	}
}

func assertAssetNameContainsHashPrefix(t *testing.T, dir string, resolved string) {
	t.Helper()

	data, err := os.ReadFile(filepath.Join(dir, resolved))
	if err != nil {
		t.Fatalf("read asset %q: %v", resolved, err)
	}

	sum := sha256.Sum256(data)
	hashPrefix := fmt.Sprintf("%x", sum[:])[:7]
	if !strings.Contains(resolved, "."+hashPrefix) {
		t.Fatalf("asset %q does not contain content hash prefix %q", resolved, hashPrefix)
	}
}
