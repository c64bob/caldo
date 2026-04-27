package view

import (
	"context"

	"caldo/internal/assets"
)

type contextKey string

const (
	assetManifestKey contextKey = "asset_manifest"
	csrfTokenKey     contextKey = "csrf_token"
)

// WithAssetManifest stores the static asset manifest in request context.
func WithAssetManifest(ctx context.Context, manifest assets.Manifest) context.Context {
	return context.WithValue(ctx, assetManifestKey, manifest)
}

// AssetPath returns the cache-busted /static path for a logical asset key.
func AssetPath(ctx context.Context, logicalName string) string {
	manifest, ok := ctx.Value(assetManifestKey).(assets.Manifest)
	if !ok {
		return ""
	}

	resolved, err := manifest.Resolve(logicalName)
	if err != nil {
		return ""
	}

	return "/static/" + resolved
}

// WithCSRFToken stores the current CSRF token in request context.
func WithCSRFToken(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, csrfTokenKey, token)
}

// CSRFToken returns the CSRF token from request context.
func CSRFToken(ctx context.Context) string {
	token, _ := ctx.Value(csrfTokenKey).(string)
	return token
}
