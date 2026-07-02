package templatemeta

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSaveMeta(t *testing.T) {
	root := t.TempDir()
	meta := PackMeta{
		Name:   "Test Theme",
		Author: "Cannon",
		Type:   TypeFull,
		Status: StatusActive,
	}
	if err := Save(root, meta); err != nil {
		t.Fatal(err)
	}
	got, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != meta.Name || got.Author != meta.Author {
		t.Fatalf("got %#v", got)
	}
}

func TestOverridesEnabled(t *testing.T) {
	root := t.TempDir()
	if !OverridesEnabled(root) {
		t.Fatal("expected overrides enabled by default")
	}
	if err := Save(root, PackMeta{Status: StatusInactive}); err != nil {
		t.Fatal(err)
	}
	if OverridesEnabled(root) {
		t.Fatal("expected overrides disabled for inactive pack")
	}
}

func TestGroupDefaults(t *testing.T) {
	meta := DefaultPackMeta()
	if meta.GroupType("admin") != TypeBackend {
		t.Fatalf("admin type = %q", meta.GroupType("admin"))
	}
	if meta.GroupType("default") != TypeFrontend {
		t.Fatalf("default type = %q", meta.GroupType("default"))
	}
}

func TestLoadMissingMeta(t *testing.T) {
	meta, err := Load(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if meta.Status != StatusActive {
		t.Fatalf("got %#v", meta)
	}
}

func TestSaveCreatesFile(t *testing.T) {
	root := t.TempDir()
	if err := Save(root, DefaultPackMeta()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, FileName)); err != nil {
		t.Fatal(err)
	}
}
