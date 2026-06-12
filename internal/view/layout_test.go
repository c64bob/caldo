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
		`data-quick-add-open`,
		`id="quick-add-overlay"`,
		`data-quick-add-overlay`,
		`aria-controls="quick-add-overlay"`,
		`caldo-nav-link-active`,
		`caldo-button caldo-button-secondary`,
		`caldo-dialog`,
		`caldo-kbd`,
		`caldo-page-title`,
		`<button type="button" data-theme-toggle`,
		`href="/quick-add"`,
		`href="/search"`,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected layout to include component class %q", want)
		}
	}
}

func TestBaseLayoutUsesUIPreferences(t *testing.T) {
	t.Parallel()

	ctx := WithCSRFToken(context.Background(), "token-123")
	ctx = WithAssetManifest(ctx, assets.Manifest{
		"app.css":       "app.hash.css",
		"htmx.min.js":   "htmx.hash.js",
		"htmx-sse.js":   "htmx-sse.hash.js",
		"alpine.min.js": "alpine.hash.js",
		"app.js":        "app.hash.js",
	})
	ctx = WithUIPreferences(ctx, "en", "dark")

	component := BaseLayout("Heute", PlaceholderPage("Heute"))

	var rendered bytes.Buffer
	if err := component.Render(ctx, &rendered); err != nil {
		t.Fatalf("render layout: %v", err)
	}

	output := rendered.String()
	for _, want := range []string{
		`<html lang="en" data-theme-root data-theme-mode="dark" class="dark">`,
		`<title>Today</title>`,
		`Current view`,
		`>Today<`,
		`New task`,
		`System filters`,
		`Appearance: Dark`,
		`data-theme-mode="dark"`,
		`data-theme-label-prefix="Appearance"`,
		`data-theme-dark-label="Dark"`,
		`aria-label="Appearance: Dark"`,
		`Keyboard shortcuts`,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected localized themed layout to include %q in %s", want, output)
		}
	}
}

func TestBaseLayoutRendersSystemThemeToggleButton(t *testing.T) {
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
		`<html lang="de" data-theme-root data-theme-mode="system"`,
		`<button type="button" data-theme-toggle`,
		`data-theme-mode="system"`,
		`data-theme-label-prefix="Darstellung"`,
		`data-theme-system-label="System"`,
		`data-theme-light-label="Hell"`,
		`data-theme-dark-label="Dunkel"`,
		`aria-label="Darstellung: System"`,
		`Darstellung: System`,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected system theme layout to include %q in %s", want, output)
		}
	}
	if strings.Contains(output, `href="/settings#ui-settings" data-theme-toggle`) {
		t.Fatalf("expected theme toggle to be a button, got settings link: %s", output)
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
		[]NavigationOverviewItem{{ID: "project-1", Name: "Inbox mit sehr langem Namen", Href: "/search?q=%23Inbox", Count: 7, HasCount: true}},
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
		`data-project-drop-target`,
		`data-project-id="project-1"`,
		`data-project-name="Inbox mit sehr langem Namen"`,
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
