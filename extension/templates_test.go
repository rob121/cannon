package extension_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/rob121/cannon/extension"
)

func TestTablePrefixFromName(t *testing.T) {
	if got := extension.TablePrefixFromName("cannon-ext-contact"); got != "contact" {
		t.Fatalf("prefix: got %q", got)
	}
	if got := extension.TableName("contact", "forms"); got != "contact_forms" {
		t.Fatalf("table name: got %q", got)
	}
	info := extension.Info{Name: "cannon-ext-contact"}
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

func TestTemplatesListEmitsOverridePaths(t *testing.T) {
	embed := fstest.MapFS{
		"templates/contact/form.html":  {Data: []byte("<p>Default</p>")},
		"templates/contact/admin.html": {Data: []byte("<p>Admin</p>")},
		"templates/contact/readme.txt": {Data: []byte("ignored")},
	}
	tpl := extension.NewTemplates(embed, "templates")

	got, err := tpl.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("expected two html templates, got %#v", got)
	}
	want := map[string]string{
		"contact/admin.html": "extension/contact/admin.html",
		"contact/form.html":  "extension/contact/form.html",
	}
	for _, tmpl := range got {
		if tmpl.OverridePath != want[tmpl.Path] {
			t.Fatalf("template %q override path: got %q want %q", tmpl.Path, tmpl.OverridePath, want[tmpl.Path])
		}
	}
}

func TestServerTemplatesCapabilityListsOverridableFiles(t *testing.T) {
	s := extension.New(extension.Info{Name: "template-test", Version: "1"})
	s.EmbedTemplates(fstest.MapFS{
		"templates/contact/form.html": {Data: []byte("<p>Default</p>")},
	}, "templates")
	h := s.Handler()

	req := httptest.NewRequest(http.MethodGet, "/capabilities", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	var caps struct {
		Capabilities map[string]string `json:"capabilities"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&caps); err != nil {
		t.Fatalf("decode capabilities: %v", err)
	}
	if caps.Capabilities["templates"] != "/templates" {
		t.Fatalf("templates capability: got %#v", caps.Capabilities)
	}

	req = httptest.NewRequest(http.MethodGet, "/templates", nil)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	var list extension.TemplateListResponse
	if err := json.NewDecoder(rec.Body).Decode(&list); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(list.Templates) != 1 || list.Templates[0].Path != "contact/form.html" || list.Templates[0].OverridePath != "extension/contact/form.html" {
		t.Fatalf("unexpected templates list: %#v", list.Templates)
	}

	req = httptest.NewRequest(http.MethodGet, "/templates/contact/form.html", nil)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	var source extension.TemplateSourceResponse
	if err := json.NewDecoder(rec.Body).Decode(&source); err != nil {
		t.Fatalf("decode source: %v", err)
	}
	if source.Path != "contact/form.html" || source.OverridePath != "extension/contact/form.html" || source.Content != "<p>Default</p>" {
		t.Fatalf("unexpected source: %+v", source)
	}
}
