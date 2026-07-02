package blocks

import (
	"testing"

	"github.com/rob121/cannon/internal/models"
)

func TestResolveBlockTemplate(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{DefaultWrapperTemplate, DefaultWrapperTemplate},
		{LoginBlockTemplate, LoginBlockTemplate},
		{"default/partials/block.html", DefaultWrapperTemplate},
		{"default/partials/login-block.html", LoginBlockTemplate},
		{"default/partials/blocks/custom.html", "default/partials/blocks/custom.html"},
		{"", ""},
	}
	for _, tc := range tests {
		if got := ResolveBlockTemplate(tc.in); got != tc.want {
			t.Fatalf("ResolveBlockTemplate(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestFinishBlockHTMLLegacyWrapper(t *testing.T) {
	row := models.Block{Name: "Promo", Space: "sidebar"}
	meta := Metadata{TemplateWrapper: "default/partials/block.html", ShowName: true}
	render := func(name string, data map[string]any) (string, error) {
		if name != DefaultWrapperTemplate {
			t.Fatalf("template = %q, want %q", name, DefaultWrapperTemplate)
		}
		return "<wrapped>", nil
	}
	got, err := finishBlockHTML(row, meta, "<em>Hi</em>", render)
	if err != nil {
		t.Fatal(err)
	}
	if got != "<wrapped>" {
		t.Fatalf("got %q", got)
	}
}
