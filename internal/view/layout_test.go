package view

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"caldo/internal/assets"
)

func TestBaseLayoutDeclaresFaviconsAndBrowserMetadata(t *testing.T) {
	t.Parallel()

	ctx := WithAssetManifest(context.Background(), assets.Manifest{
		"app.css":              "app.hash.css",
		"htmx.min.js":          "htmx.hash.js",
		"htmx-sse.js":          "htmx-sse.hash.js",
		"alpine.min.js":        "alpine.hash.js",
		"app.js":               "app.hash.js",
		"favicon.svg":          "favicon.svg-hash.svg",
		"favicon.png":          "favicon.png-hash.png",
		"apple-touch-icon.png": "apple-touch-icon.hash.png",
	})

	var rendered bytes.Buffer
	if err := BaseLayout("Heute", EmptyContent()).Render(ctx, &rendered); err != nil {
		t.Fatalf("render layout: %v", err)
	}

	output := rendered.String()
	for _, want := range []string{
		`<meta name="application-name" content="Caldo">`,
		`<meta name="apple-mobile-web-app-title" content="Caldo">`,
		`<meta name="color-scheme" content="light dark">`,
		`<meta name="theme-color" content="#ffffff" media="(prefers-color-scheme: light)">`,
		`<meta name="theme-color" content="#151513" media="(prefers-color-scheme: dark)">`,
		`<link rel="icon" type="image/svg+xml" href="/static/favicon.svg-hash.svg">`,
		`<link rel="icon" type="image/png" sizes="32x32" href="/static/favicon.png-hash.png">`,
		`<link rel="apple-touch-icon" sizes="180x180" href="/static/apple-touch-icon.hash.png">`,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected browser metadata to include %q in %s", want, output)
		}
	}
}

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
		`aria-expanded="false"`,
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

func TestBaseLayoutUsesSemanticTopbarHeading(t *testing.T) {
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
	if !strings.Contains(output, `<h1 class="caldo-topbar-heading">Heute</h1>`) {
		t.Fatalf("expected topbar title to be the semantic page heading: %s", output)
	}
	if strings.Contains(output, `<p class="caldo-topbar-heading">`) {
		t.Fatalf("topbar heading must not render as a paragraph: %s", output)
	}
}

func TestBaseLayoutKeepsQuickAddOutOfTopbar(t *testing.T) {
	t.Parallel()

	ctx := WithCSRFToken(context.Background(), "token-123")
	ctx = WithAssetManifest(ctx, assets.Manifest{
		"app.css":       "app.hash.css",
		"htmx.min.js":   "htmx.hash.js",
		"htmx-sse.js":   "htmx-sse.hash.js",
		"alpine.min.js": "alpine.hash.js",
		"app.js":        "app.hash.js",
	})

	var rendered bytes.Buffer
	if err := BaseLayoutWithTopbarAction("Heute", TaskListDisplayControls(TaskListDisplayView{}), EmptyContent()).Render(ctx, &rendered); err != nil {
		t.Fatalf("render layout: %v", err)
	}

	output := rendered.String()
	if count := strings.Count(output, `<a href="/quick-add"`); count != 2 {
		t.Fatalf("expected sidebar and mobile Quick Add links only, got %d in %s", count, output)
	}
	topbarStart := strings.Index(output, `<header class="caldo-topbar">`)
	topbarEnd := strings.Index(output, `</header>`)
	if topbarStart < 0 || topbarEnd <= topbarStart {
		t.Fatalf("expected topbar markup in %s", output)
	}
	topbar := output[topbarStart:topbarEnd]
	for _, unwanted := range []string{`href="/quick-add"`, `data-quick-add-open`} {
		if strings.Contains(topbar, unwanted) {
			t.Fatalf("topbar contains Quick Add control %q in %s", unwanted, topbar)
		}
	}
	for _, want := range []string{`data-task-display`, `href="/search"`, `id="sync-status"`, `data-theme-toggle`} {
		if !strings.Contains(topbar, want) {
			t.Fatalf("topbar lost action %q in %s", want, topbar)
		}
	}
}

func TestBaseLayoutRendersOptionalTaskDisplayControlOnlyInTopbar(t *testing.T) {
	t.Parallel()

	ctx := WithCSRFToken(context.Background(), "token-123")
	ctx = WithAssetManifest(ctx, assets.Manifest{
		"app.css":       "app.hash.css",
		"htmx.min.js":   "htmx.hash.js",
		"htmx-sse.js":   "htmx-sse.hash.js",
		"alpine.min.js": "alpine.hash.js",
		"app.js":        "app.hash.js",
	})

	var rendered bytes.Buffer
	component := BaseLayoutWithTopbarAction(
		"Heute",
		TaskListDisplayControls(TaskListDisplayView{}),
		ConfigurableTaskListPage("Heute", "Keine Aufgaben", nil),
	)
	if err := component.Render(ctx, &rendered); err != nil {
		t.Fatalf("render layout with topbar action: %v", err)
	}

	output := rendered.String()
	if count := strings.Count(output, " data-task-display>"); count != 1 {
		t.Fatalf("expected exactly one display control, got %d in %s", count, output)
	}
	actionsStart := strings.Index(output, `<div class="caldo-topbar-actions">`)
	displayStart := strings.Index(output, " data-task-display>")
	mainStart := strings.Index(output, `<main class="caldo-main">`)
	if actionsStart < 0 || displayStart < actionsStart || mainStart < displayStart {
		t.Fatalf("expected display control between topbar actions and main content: %s", output)
	}
	if strings.Contains(output[mainStart:], " data-task-display>") {
		t.Fatalf("main content must not contain a display control: %s", output[mainStart:])
	}

	rendered.Reset()
	if err := BaseLayout("Einstellungen", EmptyContent()).Render(ctx, &rendered); err != nil {
		t.Fatalf("render layout without topbar action: %v", err)
	}
	if strings.Contains(rendered.String(), " data-task-display>") {
		t.Fatalf("non-configurable page must not render a display control: %s", rendered.String())
	}
}

func TestBaseLayoutUsesSyncStatusFromContext(t *testing.T) {
	t.Parallel()

	ctx := WithCSRFToken(context.Background(), "token-123")
	ctx = WithAssetManifest(ctx, assets.Manifest{
		"app.css":       "app.hash.css",
		"htmx.min.js":   "htmx.hash.js",
		"htmx-sse.js":   "htmx-sse.hash.js",
		"alpine.min.js": "alpine.hash.js",
		"app.js":        "app.hash.js",
	})
	ctx = WithSyncStatus(ctx, SyncStatusView{State: "idle", LastSuccess: LocalDateTimeView{Text: "02.01.2026 03:04", ISO: "2026-01-02T03:04:00Z"}})

	component := BaseLayout("Heute", EmptyContent())

	var rendered bytes.Buffer
	if err := component.Render(ctx, &rendered); err != nil {
		t.Fatalf("render layout: %v", err)
	}

	output := rendered.String()
	for _, want := range []string{
		`data-sync-state="idle"`,
		`aria-label="Jetzt synchronisieren. Status: bereit. Letzter erfolgreicher Sync: 02.01.2026 03:04"`,
		`title="Jetzt synchronisieren. Status: bereit. Letzter erfolgreicher Sync: 02.01.2026 03:04"`,
		`02.01.2026 03:04`,
		`datetime="2026-01-02T03:04:00Z"`,
		`data-local-date-time`,
		`id="sync-status"`,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected layout sync status to include %q in %s", want, output)
		}
	}
}

func TestBaseLayoutDoesNotOpenNormalEventsStreamByDefault(t *testing.T) {
	t.Parallel()

	ctx := WithCSRFToken(context.Background(), "token-123")
	ctx = WithAssetManifest(ctx, assets.Manifest{
		"app.css":       "app.hash.css",
		"htmx.min.js":   "htmx.hash.js",
		"htmx-sse.js":   "htmx-sse.hash.js",
		"alpine.min.js": "alpine.hash.js",
		"app.js":        "app.hash.js",
	})

	component := BaseLayout("Caldo Setup", EmptyContent())

	var rendered bytes.Buffer
	if err := component.Render(ctx, &rendered); err != nil {
		t.Fatalf("render layout: %v", err)
	}

	output := rendered.String()
	if strings.Contains(output, `sse-connect="/events"`) {
		t.Fatalf("expected default layout not to open normal events stream: %s", output)
	}
	if !strings.Contains(output, `Letzter erfolgreicher Sync: nie`) {
		t.Fatalf("expected default layout to keep sync status fallback: %s", output)
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
		`title="Appearance: Dark"`,
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
		`caldo-theme-icon-system`,
		`aria-label="Darstellung: System"`,
		`title="Darstellung: System"`,
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
		`data-nav-settings`,
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
	if count := strings.Count(output, `data-nav-settings`); count != 2 {
		t.Fatalf("expected desktop and mobile settings nav blocks, got %d in %s", count, output)
	}
	assertSettingsOutsideSystemFilters(t, output)
	if strings.Contains(output, `href="/settings" aria-current="page"`) {
		t.Fatalf("expected inactive settings link not to be current: %s", output)
	}
}

func TestBaseLayoutPinsSettingsNavigationToBottom(t *testing.T) {
	t.Parallel()

	ctx := WithCSRFToken(context.Background(), "token-123")
	ctx = WithAssetManifest(ctx, assets.Manifest{
		"app.css":       "app.hash.css",
		"htmx.min.js":   "htmx.hash.js",
		"htmx-sse.js":   "htmx-sse.hash.js",
		"alpine.min.js": "alpine.hash.js",
		"app.js":        "app.hash.js",
	})

	component := BaseLayout("Einstellungen", PlaceholderPage("Einstellungen"))

	var rendered bytes.Buffer
	if err := component.Render(ctx, &rendered); err != nil {
		t.Fatalf("render layout: %v", err)
	}

	output := rendered.String()
	assertSettingsOutsideSystemFilters(t, output)
	if count := strings.Count(output, `data-nav-settings`); count != 2 {
		t.Fatalf("expected desktop and mobile settings nav blocks, got %d in %s", count, output)
	}
	if count := strings.Count(output, `href="/settings" aria-current="page"`); count != 2 {
		t.Fatalf("expected settings links to be active in desktop and mobile bottom navs, got %d in %s", count, output)
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
		`aria-labelledby="shortcut-help-title"`,
		`id="shortcut-help-title"`,
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
		`Jetzt synchronisieren`,
		`Hilfe öffnen`,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected shortcut help to include %q", want)
		}
	}
	if strings.Contains(output, `Mehrfachbearbeitung ist nicht verfügbar`) {
		t.Fatalf("expected shortcut help to omit unavailable multi-edit row")
	}
}

func assertSettingsOutsideSystemFilters(t *testing.T, output string) {
	t.Helper()

	sections := strings.Split(output, `data-nav-system-filters`)
	if len(sections) != 3 {
		t.Fatalf("expected desktop and mobile system filter lists, got %d in %s", len(sections)-1, output)
	}
	for _, section := range sections[1:] {
		end := strings.Index(section, `data-nav-user-filters`)
		if end < 0 {
			t.Fatalf("expected user filters after system filters in %s", section)
		}
		systemList := section[:end]
		if strings.Contains(systemList, `href="/settings"`) || strings.Contains(systemList, `>Einstellungen<`) {
			t.Fatalf("settings must not be rendered inside system filters: %s", systemList)
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
		[]NavigationOverviewItem{{ID: "project-1", Name: "Inbox mit sehr langem Namen", Href: "/projects/project-1", Count: 7, HasCount: true, Active: true}},
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
		`caldo-sidebar-project-list`,
		`tabindex="0"`,
		`Büro`,
		`Heute Fokus`,
		`href="/projects/project-1"`,
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
