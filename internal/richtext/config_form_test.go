package richtext

import (
	"strings"
	"testing"
)

func TestInjectConfigurationToolbarBuilder(t *testing.T) {
	form := `<form method="post"><input name="x"><button class="btn btn-admin-primary" type="submit">Save</button></form>`
	cfg := Settings{
		IntroToolbar: []string{"bold", "italic", "|", "link"},
		BodyToolbar:  []string{"heading", "bold"},
	}
	out := InjectConfigurationToolbarBuilder(form, cfg)
	for _, part := range []string{
		"admin-richtext-toolbar-builder",
		"name=\"#/properties/toolbar_intro\"",
		"name=\"#/properties/toolbar_body\"",
		"data-item-id=\"bold\"",
		"admin-richtext-toolbar-pool",
	} {
		if !strings.Contains(out, part) {
			t.Fatalf("missing %q in:\n%s", part, out)
		}
	}
}
