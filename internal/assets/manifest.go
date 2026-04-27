package assets

import (
	"encoding/json"
	"fmt"
	"os"
)

// Manifest maps logical asset names to cache-busted file names.
type Manifest map[string]string

// LoadManifest reads and validates a manifest.json file.
func LoadManifest(path string) (Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read manifest: %w", err)
	}

	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("decode manifest: %w", err)
	}

	if len(manifest) == 0 {
		return nil, fmt.Errorf("manifest is empty")
	}

	return manifest, nil
}

// Resolve returns the cache-busted file name for a logical asset path.
func (m Manifest) Resolve(name string) (string, error) {
	resolved, ok := m[name]
	if !ok {
		return "", fmt.Errorf("asset %q missing from manifest", name)
	}
	return resolved, nil
}
