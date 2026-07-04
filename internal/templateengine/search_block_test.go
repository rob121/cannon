package templateengine

import (
	"strings"
	"testing"

	"github.com/rob121/cannon/internal/themes"
)

func TestSearchHorizontalBlockTemplate(t *testing.T) {
	e := New(nil, themes.Selection{}, nil, nil, testFrontendFuncs())
	tmpl, err := e.parse("default/partials/blocks/search-horizontal.html")
	if err != nil {
		t.Fatal(err)
	}
	var buf strings.Builder
	if err := tmpl.Execute(&buf, map[string]any{
		"Action":      "/content/search",
		"Placeholder": "Search items…",
		"Button":      "Search",
		"Label":       "Site search",
		"Class":       "header-search",
		"BlockID":     uint(3),
	}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, part := range []string{
		"site-search-block site-search-horizontal header-search",
		`method="get"`,
		`action="/content/search"`,
		`name="q"`,
		`id="search-block-q-3"`,
		"Site search",
		"Search items…",
		`type="submit"`,
		"input-group",
	} {
		if !strings.Contains(out, part) {
			t.Fatalf("missing %q in output: %s", part, out)
		}
	}
	if strings.Contains(out, "captcha") {
		t.Fatalf("search block should not include captcha: %s", out)
	}
}

func TestSearchVerticalBlockTemplate(t *testing.T) {
	e := New(nil, themes.Selection{}, nil, nil, testFrontendFuncs())
	tmpl, err := e.parse("default/partials/blocks/search-vertical.html")
	if err != nil {
		t.Fatal(err)
	}
	var buf strings.Builder
	if err := tmpl.Execute(&buf, map[string]any{
		"Action":      "/content/search",
		"Placeholder": "Search…",
		"Button":      "Find",
		"Class":       "sidebar-search",
		"BlockID":     uint(5),
	}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, part := range []string{
		"site-search-block site-search-vertical sidebar-search",
		`aria-label="Search…"`,
		`id="search-block-q-5"`,
		"btn btn-primary w-100",
		"Find",
	} {
		if !strings.Contains(out, part) {
			t.Fatalf("missing %q in output: %s", part, out)
		}
	}
}
