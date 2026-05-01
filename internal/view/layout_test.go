package view

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"caldo/internal/assets"
)

func TestBaseLayoutIncludesWriteStatusRegion(t *testing.T) {
	t.Parallel()

	ctx := WithCSRFToken(context.Background(), "token-123")
	ctx = WithAssetManifest(ctx, assets.Manifest{
		"app.css":      "app.hash.css",
		"htmx.min.js":  "htmx.hash.js",
		"htmx-sse.js":  "htmx-sse.hash.js",
		"alpine.min.js": "alpine.hash.js",
		"app.js":       "app.hash.js",
	})

	component := BaseLayout("Heute", EmptyContent())

	var rendered bytes.Buffer
	if err := component.Render(ctx, &rendered); err != nil {
		t.Fatalf("render layout: %v", err)
	}

	output := rendered.String()
	if !strings.Contains(output, `id="write-status"`) {
		t.Fatal("expected write status region in base layout")
	}
}
