package templatemgr

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rob121/cannon/internal/templatemeta"
	"github.com/rob121/cannon/internal/themes"
)

func TestDetectAndMigrateLegacy(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "default", "controllers"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "default", "layout.html"), []byte("<html>{{.Main}}</html>"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := templatemeta.Save(root, templatemeta.PackMeta{
		Name: "Old Pack", Type: templatemeta.TypeFull, Status: templatemeta.StatusActive,
	}); err != nil {
		t.Fatal(err)
	}

	report, ok := DetectLegacy(root)
	if !ok || !report.HasRootMeta || !report.HasDefaultDir {
		t.Fatalf("expected legacy detect, got %#v ok=%v", report, ok)
	}

	front, admin, err := MigrateLegacy(root, "mysite", "")
	if err != nil {
		t.Fatal(err)
	}
	if front != "mysite" || admin != "" {
		t.Fatalf("front=%q admin=%q", front, admin)
	}
	if _, err := os.Stat(filepath.Join(root, "mysite", "layout.html")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "default")); !os.IsNotExist(err) {
		t.Fatal("legacy default/ should be removed")
	}
	if _, err := os.Stat(filepath.Join(root, templatemeta.FileName)); !os.IsNotExist(err) {
		t.Fatal("root template.json should be removed")
	}
	list, err := themes.List(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].Name != "mysite" {
		t.Fatalf("themes=%#v", list)
	}
	meta, err := templatemeta.Load(themes.Root(root, "mysite"))
	if err != nil {
		t.Fatal(err)
	}
	if meta.Type != templatemeta.TypeFull {
		t.Fatalf("type=%q", meta.Type)
	}
}

func TestMigrateLegacyBothTrees(t *testing.T) {
	root := t.TempDir()
	for _, tree := range []string{"default", "admin"} {
		if err := os.MkdirAll(filepath.Join(root, tree), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, tree, "layout.html"), []byte("<html></html>"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	front, admin, err := MigrateLegacy(root, "site", "site-admin")
	if err != nil {
		t.Fatal(err)
	}
	if front != "site" || admin != "site-admin" {
		t.Fatalf("front=%q admin=%q", front, admin)
	}
	fm, _ := templatemeta.Load(themes.Root(root, "site"))
	am, _ := templatemeta.Load(themes.Root(root, "site-admin"))
	if fm.Type != templatemeta.TypeFrontend || am.Type != templatemeta.TypeBackend {
		t.Fatalf("types front=%q admin=%q", fm.Type, am.Type)
	}
}
