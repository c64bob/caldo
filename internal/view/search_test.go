package view

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestSearchPageRendersSaveFilterFormForEligibleQuery(t *testing.T) {
	t.Parallel()

	ctx := WithCSRFToken(context.Background(), "token-123")
	component := SearchPage("#Work", nil, InlineTaskCreateView{}, SearchSaveFilterView{
		Enabled:    true,
		Query:      "#Work",
		IsFavorite: true,
	})

	var rendered bytes.Buffer
	if err := component.Render(ctx, &rendered); err != nil {
		t.Fatalf("render search page: %v", err)
	}

	output := rendered.String()
	for _, want := range []string{
		`data-search-save-filter-form`,
		`id="global-search"`,
		`hx-get="/search/results"`,
		`hx-trigger="input changed delay:350ms, search"`,
		`hx-target="#search-live-results"`,
		`hx-swap="outerHTML"`,
		`hx-push-url="false"`,
		`data-live-search-input`,
		`id="search-live-results"`,
		`data-search-live-results`,
		`aria-live="polite"`,
		`method="post"`,
		`action="/filters"`,
		`hx-post="/filters"`,
		`hx-target="body"`,
		`X-CSRF-Token&#34;:&#34;token-123`,
		`name="query" value="#Work"`,
		`data-search-save-filter-name`,
		`data-search-save-filter-query`,
		`#Work`,
		`name="favorite" value="1" checked`,
		`Filter anlegen`,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected search save filter form to include %q in %s", want, output)
		}
	}
}

func TestSearchPageOmitsSaveFilterFormWhenQueryIsNotEligible(t *testing.T) {
	t.Parallel()

	component := SearchPage("plain text", nil, InlineTaskCreateView{}, SearchSaveFilterView{})

	var rendered bytes.Buffer
	if err := component.Render(context.Background(), &rendered); err != nil {
		t.Fatalf("render search page: %v", err)
	}

	output := rendered.String()
	if strings.Contains(output, `data-search-save-filter-form`) {
		t.Fatalf("search page must not render save filter form for ineligible query: %s", output)
	}
}
