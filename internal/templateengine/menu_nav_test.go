package templateengine

import (
	"strings"
	"testing"

	"github.com/rob121/cannon/internal/themes"
)

func TestMenuNavTemplateNested(t *testing.T) {
	e := New(nil, themes.Selection{}, nil, nil, testFrontendFuncs())
	navTmpl, err := e.parse("default/partials/menu-nav.html")
	if err != nil {
		t.Fatal(err)
	}
	node := navTmpl.Lookup("menu-nav-node")
	if node == nil {
		t.Fatal("menu-nav-node define missing")
	}
	var buf strings.Builder
	err = node.Execute(&buf, map[string]any{
		"Item": map[string]any{
			"Name": "Resources",
			"Href": "/resources",
			"Children": []map[string]any{
				{"Name": "Downloads", "Href": "/resources/downloads"},
			},
		},
		"Depth": 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, part := range []string{"dropdown-toggle", "dropdown-menu", "Downloads", `data-bs-toggle="dropdown"`} {
		if !strings.Contains(out, part) {
			t.Fatalf("missing %q in menu nav output: %s", part, out)
		}
	}
}