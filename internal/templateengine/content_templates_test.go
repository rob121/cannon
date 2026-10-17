package templateengine

import (
	"html/template"
	"strings"
	"testing"

	"github.com/rob121/cannon/internal/content"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/themes"
)

func TestItemTemplateContentSlugSpaces(t *testing.T) {
	InvalidateAll()
	var spaces []string
	funcs := testFrontendFuncs()
	funcs["safeHTML"] = func(value string) template.HTML { return template.HTML(value) }
	funcs["csrfField"] = func() template.HTML { return "" }
	funcs["space"] = func(name string) (template.HTML, error) {
		spaces = append(spaces, name)
		return template.HTML("<!--" + name + "-->"), nil
	}
	e := New(nil, themes.Selection{}, nil, nil, funcs)
	tmpl, err := e.parse("default/controllers/content/item.html")
	if err != nil {
		t.Fatal(err)
	}
	var out strings.Builder
	if err := tmpl.Execute(&out, map[string]any{
		"Item":            models.Item{Title: "Important Dates", Slug: "important-dates"},
		"IntroHTML":       template.HTML("<p>Intro</p>"),
		"BodyHTML":        template.HTML("<p>Body</p>"),
		"ContentSettings": content.Settings{},
	}); err != nil {
		t.Fatal(err)
	}
	want := []string{
		"content-important-dates-start",
		"content-important-dates-middle",
		"content-important-dates-end",
	}
	if len(spaces) != len(want) {
		t.Fatalf("spaces = %#v, want %#v", spaces, want)
	}
	for i, name := range want {
		if spaces[i] != name {
			t.Fatalf("spaces[%d] = %q, want %q", i, spaces[i], name)
		}
	}
	html := out.String()
	start := strings.Index(html, "<!--content-important-dates-start-->")
	intro := strings.Index(html, "<p>Intro</p>")
	middle := strings.Index(html, "<!--content-important-dates-middle-->")
	body := strings.Index(html, "<p>Body</p>")
	end := strings.Index(html, "<!--content-important-dates-end-->")
	if !(start >= 0 && start < intro && intro < middle && middle < body && body < end) {
		t.Fatalf("unexpected order: start=%d intro=%d middle=%d body=%d end=%d\n%s", start, intro, middle, body, end, html)
	}
}

func TestItemTemplateHidesEmptyBody(t *testing.T) {
	InvalidateAll()
	funcs := testFrontendFuncs()
	funcs["safeHTML"] = func(value string) template.HTML { return template.HTML(value) }
	funcs["csrfField"] = func() template.HTML { return "" }
	funcs["space"] = func(name string) (template.HTML, error) { return "", nil }
	e := New(nil, themes.Selection{}, nil, nil, funcs)
	tmpl, err := e.parse("default/controllers/content/item.html")
	if err != nil {
		t.Fatal(err)
	}
	var out strings.Builder
	if err := tmpl.Execute(&out, map[string]any{
		"Item":            models.Item{Title: "Camp Workday", Slug: "camp-workday"},
		"IntroHTML":       template.HTML("<p>Intro only</p>"),
		"BodyHTML":        template.HTML(""),
		"ContentSettings": content.Settings{},
	}); err != nil {
		t.Fatal(err)
	}
	html := out.String()
	if !strings.Contains(html, "Intro only") {
		t.Fatal("expected intro rendered")
	}
	if strings.Contains(html, "site-item-detail__body") {
		t.Fatalf("empty body should not render body wrapper:\n%s", html)
	}
}

func TestItemTemplateRelatedVisibility(t *testing.T) {
	InvalidateAll()
	funcs := testFrontendFuncs()
	funcs["safeHTML"] = func(value string) template.HTML { return template.HTML(value) }
	funcs["csrfField"] = func() template.HTML { return "" }
	e := New(nil, themes.Selection{}, nil, nil, funcs)
	tmpl, err := e.parse("default/controllers/content/item.html")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name    string
		enabled bool
		related []models.Item
		want    bool
	}{
		{"shown", true, []models.Item{{Title: "Related Story", Slug: "related-story"}}, true},
		{"hidden", false, []models.Item{{Title: "Related Story", Slug: "related-story"}}, false},
		{"empty", true, nil, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out strings.Builder
			if err := tmpl.Execute(&out, map[string]any{
				"Item":            models.Item{Title: "Story"},
				"ContentSettings": content.Settings{ShowRelatedItems: tc.enabled},
				"Related":         tc.related,
			}); err != nil {
				t.Fatal(err)
			}
			if got := strings.Contains(out.String(), ">Related</h2>"); got != tc.want {
				t.Fatalf("Related section visible = %v, want %v: %s", got, tc.want, out.String())
			}
			if got := strings.Contains(out.String(), "Related Story"); got != tc.want {
				t.Fatalf("related link visible = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestFeaturedTemplateIncludesPaginationPartial(t *testing.T) {
	e := New(nil, themes.Selection{}, nil, nil, testFrontendFuncs())
	tmpl, err := e.parse("default/controllers/content/featured.html")
	if err != nil {
		t.Fatal(err)
	}
	var buf strings.Builder
	if err := tmpl.Execute(&buf, map[string]any{
		"FeaturedCategory": map[string]any{
			"Name":        "Featured",
			"Description": "",
			"Image":       "",
		},
		"ShowFeaturedTitle":       true,
		"ShowFeaturedDescription": true,
		"LeadingItems":            nil,
		"RemainingItems":          nil,
		"ItemListClass":           "site-item-list",
		"Pagination":              false,
		"TotalPages":              0,
		"Page":                    1,
		"PaginationBaseURL":       "/content/featured",
	}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "Featured") {
		t.Fatalf("featured template did not render title: %s", buf.String())
	}
}
