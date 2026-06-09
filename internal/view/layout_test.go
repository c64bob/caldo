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
		"app.css":       "app.hash.css",
		"htmx.min.js":   "htmx.hash.js",
		"htmx-sse.js":   "htmx-sse.hash.js",
		"alpine.min.js": "alpine.hash.js",
		"app.js":        "app.hash.js",
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

func TestBaseLayoutUsesComponentFoundationClasses(t *testing.T) {
	t.Parallel()

	ctx := WithCSRFToken(context.Background(), "token-123")
	ctx = WithAssetManifest(ctx, assets.Manifest{
		"app.css":       "app.hash.css",
		"htmx.min.js":   "htmx.hash.js",
		"htmx-sse.js":   "htmx-sse.hash.js",
		"alpine.min.js": "alpine.hash.js",
		"app.js":        "app.hash.js",
	})

	component := BaseLayout("Heute", PlaceholderPage("Heute"))

	var rendered bytes.Buffer
	if err := component.Render(ctx, &rendered); err != nil {
		t.Fatalf("render layout: %v", err)
	}

	output := rendered.String()
	for _, want := range []string{
		`caldo-app-shell`,
		`caldo-sidebar`,
		`caldo-topbar`,
		`caldo-main`,
		`caldo-content`,
		`data-mobile-nav-open`,
		`data-mobile-nav-dialog`,
		`aria-label="Mobile Hauptnavigation"`,
		`caldo-nav-link-active`,
		`caldo-button caldo-button-secondary`,
		`caldo-dialog`,
		`caldo-kbd`,
		`caldo-page-title`,
		`href="/quick-add"`,
		`href="/search"`,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected layout to include component class %q", want)
		}
	}
}
