package templateengine

import (
	"html/template"
	"strings"
	"testing"

	"github.com/rob121/cannon/internal/themes"
)

func TestCardBlockWrapperTemplate(t *testing.T) {
	e := New(nil, themes.Selection{}, nil, nil, testFrontendFuncs())
	tmpl, err := e.parse("default/partials/blocks/card.html")
	if err != nil {
		t.Fatal(err)
	}
	var buf strings.Builder
	if err := tmpl.Execute(&buf, map[string]any{
		"Name":     "Footer Links",
		"Space":    "footer",
		"ShowName": true,
		"Body":     template.HTML("<p>Links</p>"),
	}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, part := range []string{"site-block site-block--footer", "Footer Links", "card-header", "Links"} {
		if !strings.Contains(out, part) {
			t.Fatalf("missing %q in output: %s", part, out)
		}
	}

	buf.Reset()
	if err := tmpl.Execute(&buf, map[string]any{
		"ShowName": false,
		"Body":     template.HTML("<p>Only body</p>"),
	}); err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(buf.String()) != "<p>Only body</p>" {
		t.Fatalf("got %q", buf.String())
	}
}

func TestMenuVerticalBlockTemplate(t *testing.T) {
	e := New(nil, themes.Selection{}, nil, nil, testFrontendFuncs())
	tmpl, err := e.parse("default/partials/blocks/menu-vertical.html")
	if err != nil {
		t.Fatal(err)
	}
	items := []map[string]any{
		{
			"Name": "About",
			"Href": "/about",
			"Children": []map[string]any{
				{"Name": "Team", "Href": "/about/team"},
			},
		},
	}
	var buf strings.Builder
	if err := tmpl.Execute(&buf, map[string]any{"Items": items, "Class": "sidebar-nav"}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, part := range []string{"site-menu-vertical", "sidebar-nav", "nav flex-column", "About", "Team", "ms-3 border-start"} {
		if !strings.Contains(out, part) {
			t.Fatalf("missing %q in output: %s", part, out)
		}
	}
}

func TestMenuHorizontalBlockTemplate(t *testing.T) {
	e := New(nil, themes.Selection{}, nil, nil, testFrontendFuncs())
	tmpl, err := e.parse("default/partials/blocks/menu-horizontal.html")
	if err != nil {
		t.Fatal(err)
	}
	items := []map[string]any{
		{
			"Name": "Services",
			"Href": "/services",
			"Children": []map[string]any{
				{"Name": "Consulting", "Href": "/services/consulting"},
			},
		},
	}
	var buf strings.Builder
	if err := tmpl.Execute(&buf, map[string]any{"Items": items, "Class": "block-nav"}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, part := range []string{"site-menu-horizontal", "block-nav", "nav flex-wrap", "dropdown-toggle", "Services", "Consulting"} {
		if !strings.Contains(out, part) {
			t.Fatalf("missing %q in output: %s", part, out)
		}
	}
}
