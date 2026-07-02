package extension_test

import (
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/rob121/cannon/extension"
)

func TestTablePrefixFromName(t *testing.T) {
	if got := extension.TablePrefixFromName("cannon-extension-contact"); got != "contact" {
		t.Fatalf("prefix: got %q", got)
	}
	if got := extension.TableName("contact", "forms"); got != "contact_forms" {
		t.Fatalf("table name: got %q", got)
	}
	info := extension.Info{Name: "cannon-extension-contact"}
	if got := info.TablePrefix(); got != "contact" {
		t.Fatalf("info prefix: got %q", got)
	}
}

func TestTemplateOverridePath(t *testing.T) {
	got, err := extension.TemplateOverridePath("contact/form.html")
	if err != nil {
		t.Fatal(err)
	}
	if got != "extension/contact/form.html" {
		t.Fatalf("override path: got %q", got)
	}
}

func TestTemplatesReadAndExecute(t *testing.T) {
	embed := fstest.MapFS{
		"templates/contact/form.html": {Data: []byte("<h1>{{.Title}}</h1>")},
	}
	tpl := extension.NewTemplates(embed, "templates")

	body, err := tpl.Execute("contact/form.html", map[string]string{"Title": "Contact"})
	if err != nil {
		t.Fatal(err)
	}
	if body != "<h1>Contact</h1>" {
		t.Fatalf("body: got %q", body)
	}
}

func TestTemplatesSiteOverride(t *testing.T) {
	dir := t.TempDir()
	overrideDir := filepath.Join(dir, "extension", "contact")
	if err := os.MkdirAll(overrideDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(overrideDir, "form.html"), []byte("<p>Override</p>"), 0644); err != nil {
		t.Fatal(err)
	}

	embed := fstest.MapFS{
		"templates/contact/form.html": {Data: []byte("<p>Default</p>")},
	}
	tpl := extension.NewTemplates(embed, "templates").WithTemplateDir(dir)

	body, err := tpl.Execute("contact/form.html", nil)
	if err != nil {
		t.Fatal(err)
	}
	if body != "<p>Override</p>" {
		t.Fatalf("body: got %q", body)
	}
}
