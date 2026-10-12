package templateengine

import (
	"strings"
	"testing"

	"github.com/rob121/cannon/internal/themes"
)

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
