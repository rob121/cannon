package templatemgr

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/rob121/cannon/internal/templateengine"
)

func TestValidateRelPath(t *testing.T) {
	cases := map[string]bool{
		"admin/dashboard.html":              true,
		"default/page.html":                   true,
		"admin/dashboard.html?sort=name":      true,
		"admin/dashboard.html&sort=name":      true,
		"":                                    false,
		"../secret.html":                      false,
		"/abs.html":                           false,
		"admin/dashboard.txt":                 false,
		"global?section=general":              false,
	}
	for path, ok := range cases {
		err := ValidateRelPath(path)
		if ok && err != nil {
			t.Fatalf("%q: unexpected error: %v", path, err)
		}
		if !ok && err == nil {
			t.Fatalf("%q: expected error", path)
		}
	}
}

func TestCleanRelPath(t *testing.T) {
	if got := CleanRelPath("admin/configuration.html?section=general"); got != "admin/configuration.html" {
		t.Fatalf("CleanRelPath: got %q", got)
	}
	if got := CleanRelPath("global?section=general"); got != "global" {
		t.Fatalf("CleanRelPath global: got %q", got)
	}
	if got := CleanRelPath("fr%252Fcontrollers%252Fcontent%252Fauthor.html"); got != "fr/controllers/content/author.html" {
		t.Fatalf("CleanRelPath double-encoded: got %q", got)
	}
	if got := CleanRelPath("fr%2Fcontrollers%2Fcontent%2Fauthor.html"); got != "fr/controllers/content/author.html" {
		t.Fatalf("CleanRelPath encoded: got %q", got)
	}
}

func TestReadDoubleEncodedPath(t *testing.T) {
	root := t.TempDir()
	rel := "fr/controllers/content/author.html"
	if err := os.MkdirAll(filepath.Join(root, "fr", "controllers", "content"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(rel)), []byte("<html>author</html>"), 0644); err != nil {
		t.Fatal(err)
	}
	content, _, fromBuiltin, err := Read(root, CleanRelPath("fr%252Fcontrollers%252Fcontent%252Fauthor.html"))
	if err != nil {
		t.Fatal(err)
	}
	if fromBuiltin {
		t.Fatal("expected on-disk override")
	}
	if content != "<html>author</html>" {
		t.Fatalf("content = %q", content)
	}
}

func TestSaveVersions(t *testing.T) {
	root := t.TempDir()
	rel := "admin/dashboard.html"

	if err := Save(root, rel, []byte("v1")); err != nil {
		t.Fatal(err)
	}
	for i := 2; i <= 5; i++ {
		if err := Save(root, rel, []byte(fmt.Sprintf("v%d", i))); err != nil {
			t.Fatal(err)
		}
	}

	versionDir := filepath.Join(root, VersionsDir, "admin")
	entries, err := os.ReadDir(versionDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3 {
		t.Fatalf("expected 3 versions kept, got %d", len(entries))
	}

	content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "v5" {
		t.Fatalf("expected latest content, got %q", string(content))
	}
}

func TestOverrideAllBuiltins(t *testing.T) {
	builtins, err := templateengine.BuiltinTemplates()
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	for _, path := range builtins {
		if err := Override(root, path); err != nil {
			t.Fatalf("override %q: %v", path, err)
		}
	}
}

func TestListSkipsVersions(t *testing.T) {
	root := t.TempDir()
	if err := Save(root, "default/page.html", []byte("hello")); err != nil {
		t.Fatal(err)
	}
	files, err := List(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0].Path != "default/page.html" {
		t.Fatalf("unexpected list: %#v", files)
	}
}
