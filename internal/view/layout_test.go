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
	for _, want := range []string{
		`id="write-status"`,
		`data-write-status`,
		`role="status"`,
		`aria-live="polite"`,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected write status region to include %q", want)
		}
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

func TestBaseLayoutRendersRequiredMainNavigation(t *testing.T) {
	t.Parallel()

	ctx := WithCSRFToken(context.Background(), "token-123")
	ctx = WithAssetManifest(ctx, assets.Manifest{
		"app.css":       "app.hash.css",
		"htmx.min.js":   "htmx.hash.js",
		"htmx-sse.js":   "htmx-sse.hash.js",
		"alpine.min.js": "alpine.hash.js",
		"app.js":        "app.hash.js",
	})

	component := BaseLayout("Suche", PlaceholderPage("Suche"))

	var rendered bytes.Buffer
	if err := component.Render(ctx, &rendered); err != nil {
		t.Fatalf("render layout: %v", err)
	}

	output := rendered.String()
	for _, want := range []string{
		`aria-label="Hauptnavigation"`,
		`aria-label="Mobile Hauptnavigation"`,
		`href="/today"`,
		`>Heute<`,
		`href="/upcoming"`,
		`>Demnächst<`,
		`href="/projects"`,
		`>Alle Projekte<`,
		`href="/labels"`,
		`>Alle Labels<`,
		`href="/filters"`,
		`>Alle Filter<`,
		`href="/favorites"`,
		`>Favoriten<`,
		`href="/search"`,
		`>Suche<`,
		`href="/conflicts"`,
		`>Konflikte<`,
		`href="/settings"`,
		`>Einstellungen<`,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected main navigation to include %q", want)
		}
	}
	if count := strings.Count(output, `href="/search" aria-current="page"`); count != 2 {
		t.Fatalf("expected desktop and mobile search links to be current, got %d in %s", count, output)
	}
	if strings.Contains(output, `href="/settings" aria-current="page"`) {
		t.Fatalf("expected inactive settings link not to be current: %s", output)
	}
}

func TestBaseLayoutRendersShortcutHelp(t *testing.T) {
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
		`data-shortcut-help-dialog`,
		`data-shortcut-help-close`,
		`Tastaturkürzel`,
		`Neue Aufgabe`,
		`Suche`,
		`Heute`,
		`Demnächst`,
		`Favoriten`,
		`Projekte`,
		`Labels`,
		`Filter`,
		`Konflikte`,
		`Einstellungen`,
		`Hilfe öffnen`,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected shortcut help to include %q", want)
		}
	}
}

func TestBaseLayoutRendersNavigationCountsAndDynamicGroups(t *testing.T) {
	t.Parallel()

	ctx := WithCSRFToken(context.Background(), "token-123")
	ctx = WithAssetManifest(ctx, assets.Manifest{
		"app.css":       "app.hash.css",
		"htmx.min.js":   "htmx.hash.js",
		"htmx-sse.js":   "htmx-sse.hash.js",
		"alpine.min.js": "alpine.hash.js",
		"app.js":        "app.hash.js",
	})
	ctx = WithNavigation(ctx, BuildNavigationSnapshot(
		3, 4, 1, 2, 5, 0, 1,
		[]NavigationOverviewItem{{Name: "Inbox mit sehr langem Namen", Href: "/search?q=%23Inbox", Count: 7, HasCount: true}},
		[]NavigationOverviewItem{{Name: "Büro", Href: "/search?q=%40B%C3%BCro", Count: 2, HasCount: true}},
		[]NavigationOverviewItem{{Name: "Heute Fokus", Href: "/filters/filter-1"}},
	))

	component := BaseLayout("Erledigt", PlaceholderPage("Erledigt"))

	var rendered bytes.Buffer
	if err := component.Render(ctx, &rendered); err != nil {
		t.Fatalf("render layout: %v", err)
	}

	output := rendered.String()
	for _, want := range []string{
		`caldo-nav-count`,
		`Inbox mit sehr langem Namen`,
		`Büro`,
		`Heute Fokus`,
		`href="/search?q=%23Inbox"`,
		`href="/search?q=%40B%C3%BCro"`,
		`href="/filters/filter-1"`,
		`Abgeschlossen`,
		`caldo-nav-link-active`,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected layout to include %q", want)
		}
	}
}
