package templatemgr

import (
	"os"
	"path/filepath"
	"testing"
)

func TestListControllerTemplateOptions(t *testing.T) {
	root := t.TempDir()
	themeDir := filepath.Join(root, "fr", "controllers", "content")
	if err := os.MkdirAll(themeDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(themeDir, "author.html"), []byte("<html></html>"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(themeDir, "foobar.html"), []byte("<html></html>"), 0644); err != nil {
		t.Fatal(err)
	}

	opts, err := ListControllerTemplateOptions(root, "default", "content", map[string]string{
		"index":    "Home",
		"category": "Category",
		"author":   "Author",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(opts) == 0 {
		t.Fatal("expected template options")
	}

	found := map[string]bool{}
	for _, opt := range opts {
		if opt.Controller != "content" {
			t.Fatalf("unexpected controller: %q", opt.Controller)
		}
		found[opt.Path] = true
	}
	if !found["default/controllers/content/index.html"] {
		t.Fatal("missing index template option")
	}
	if !found["default/controllers/content/author.html"] {
		t.Fatal("missing author template option")
	}
	if !found["default/controllers/content/foobar.html"] {
		t.Fatal("missing custom foobar template option")
	}
}
