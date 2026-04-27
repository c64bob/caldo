package view

import (
	"context"
	"testing"

	"caldo/internal/assets"
)

func TestAssetPathResolvedFromManifestInContext(t *testing.T) {
	t.Parallel()

	ctx := WithAssetManifest(context.Background(), assets.Manifest{"app.css": "app.hash.css"})

	if got := AssetPath(ctx, "app.css"); got != "/static/app.hash.css" {
		t.Fatalf("unexpected resolved path: got %q", got)
	}
}

func TestAssetPathReturnsEmptyWhenMissing(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	if got := AssetPath(ctx, "app.css"); got != "" {
		t.Fatalf("expected empty path, got %q", got)
	}
}

func TestCSRFTokenRoundTrip(t *testing.T) {
	t.Parallel()

	ctx := WithCSRFToken(context.Background(), "token-123")
	if got := CSRFToken(ctx); got != "token-123" {
		t.Fatalf("unexpected csrf token: got %q", got)
	}
}
