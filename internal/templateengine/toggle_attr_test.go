package templateengine

import (
	"bytes"
	"strings"
	"testing"

	"github.com/rob121/cannon/internal/themes"
)

func TestFormToggleLabels(t *testing.T) {
	eng := New(nil, themes.Selection{}, nil, nil, testAdminFuncs())
	set, err := eng.parseAdmin("admin/layout.html")
	if err != nil {
		t.Fatal(err)
	}
	toggle := set.Lookup("form_toggle")
	if toggle == nil {
		t.Fatal("form_toggle not found")
	}
	var buf bytes.Buffer
	if err := toggle.Execute(&buf, map[string]any{
		"Name":     "featured",
		"Checked":  true,
		"OnLabel":  "Featured",
		"OffLabel": "Not featured",
	}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if strings.Contains(out, "&#34;") || strings.Contains(out, "data-on=") {
		t.Fatalf("form_toggle should not use quoted data attributes: %s", out)
	}
	for _, want := range []string{
		`<span class="admin-form-toggle-label-on">Featured</span>`,
		`<span class="admin-form-toggle-label-off">Not featured</span>`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in output: %s", want, out)
		}
	}
}
