package blocks

import (
	"html/template"
	"os"
	"path/filepath"
	"testing"

	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/themes"
)

func TestListWrapperTemplatesIncludesDefaultAndPlain(t *testing.T) {
	opts, err := ListWrapperTemplates("", themes.BuiltinFrontend)
	if err != nil {
		t.Fatal(err)
	}
	if !wrapperOptionContains(opts, DefaultWrapperTemplate) {
		t.Fatalf("missing default wrapper: %#v", opts)
	}
	if !wrapperOptionContains(opts, PlainWrapperTemplate) {
		t.Fatalf("missing plain wrapper: %#v", opts)
	}
	if !wrapperOptionContains(opts, CardWrapperTemplate) {
		t.Fatalf("missing card wrapper: %#v", opts)
	}
	for _, skip := range []string{LoginBlockTemplate, MenuVerticalBlockTemplate, MenuHorizontalBlockTemplate} {
		if wrapperOptionContains(opts, skip) {
			t.Fatalf("body template %q should not be listed", skip)
		}
	}
}

func TestListWrapperTemplatesIncludesConfiguredTheme(t *testing.T) {
	root := t.TempDir()
	themeDir := filepath.Join(root, "fr", "partials", "blocks")
	if err := os.MkdirAll(themeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	custom := filepath.Join(themeDir, "sidebar.html")
	if err := os.WriteFile(custom, []byte("{{.Body}}"), 0o644); err != nil {
		t.Fatal(err)
	}

	opts, err := ListWrapperTemplates(root, "fr")
	if err != nil {
		t.Fatal(err)
	}
	if !wrapperOptionContains(opts, "fr/partials/blocks/sidebar.html") {
		t.Fatalf("missing custom theme wrapper: %#v", opts)
	}
}

func wrapperOptionContains(opts []WrapperOption, path string) bool {
	for _, opt := range opts {
		if opt.Path == path {
			return true
		}
	}
	return false
}

func TestFinishBlockHTMLPlainWrapper(t *testing.T) {
	row := models.Block{Name: "Promo", Space: "sidebar"}
	meta := Metadata{TemplateWrapper: PlainWrapperTemplate, ShowName: true}
	render := func(name string, data map[string]any) (string, error) {
		if name != PlainWrapperTemplate {
			t.Fatalf("template = %q", name)
		}
		body, _ := data["Body"].(template.HTML)
		return string(body), nil
	}
	got, err := finishBlockHTML(row, meta, "<em>Hi</em>", render)
	if err != nil {
		t.Fatal(err)
	}
	if got != "<em>Hi</em>" {
		t.Fatalf("got %q", got)
	}
}
