package templatemgr

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rob121/cannon/internal/templatemeta"
)

func TestListThemes(t *testing.T) {
	root := t.TempDir()
	themeDir := filepath.Join(root, "mysite")
	if err := os.MkdirAll(filepath.Join(themeDir, "assets"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(themeDir, "layout.html"), []byte("<html></html>"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := templatemeta.Save(themeDir, templatemeta.DefaultPackMeta()); err != nil {
		t.Fatal(err)
	}

	themes, err := ListThemes(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(themes) != 1 || themes[0].Name != "mysite" {
		t.Fatalf("unexpected themes: %#v", themes)
	}
}

func TestThemeTemplates(t *testing.T) {
	root := t.TempDir()
	themeDir := filepath.Join(root, "mysite")
	if err := os.MkdirAll(themeDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := templatemeta.Save(themeDir, templatemeta.PackMeta{Type: templatemeta.TypeFrontend, Status: templatemeta.StatusActive}); err != nil {
		t.Fatal(err)
	}

	entries, err := ThemeTemplates(root, "mysite")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) == 0 {
		t.Fatal("expected theme templates")
	}
	found := false
	for _, entry := range entries {
		if entry.Relative == "layout.html" {
			found = true
		}
	}
	if !found {
		t.Fatal("layout template not listed")
	}
}

func TestValidateTheme(t *testing.T) {
	if err := ValidateTheme("mysite"); err != nil {
		t.Fatal(err)
	}
	if err := ValidateTheme("../mysite"); err == nil {
		t.Fatal("expected invalid theme error")
	}
}

func TestGroupEntriesByFolder(t *testing.T) {
	entries := []TemplateEntry{
		{Path: "fr/controllers/content/author.html", Name: "author", Relative: "controllers/content/author.html", Builtin: true},
		{Path: "fr/layout.html", Name: "layout", Relative: "layout.html", Builtin: true, Overridden: true},
		{Path: "fr/page.html", Name: "page", Relative: "page.html", Builtin: true},
		{Path: "fr/controllers/content/index.html", Name: "index", Relative: "controllers/content/index.html", Builtin: true},
		{Path: "fr/partials/header.html", Name: "header", Relative: "partials/header.html", Builtin: false, Overridden: true},
	}

	root, folders := GroupEntriesByFolder(entries, "name", "asc")
	if len(root) != 2 {
		t.Fatalf("root entries = %d, want 2", len(root))
	}
	if len(folders) != 2 {
		t.Fatalf("folders = %d, want 2", len(folders))
	}
	if folders[0].Key != "controllers/content" || folders[0].Total != 2 {
		t.Fatalf("unexpected first folder: %#v", folders[0])
	}
	if folders[1].Key != "partials" || !folders[1].HasOverride {
		t.Fatalf("unexpected second folder: %#v", folders[1])
	}
}
