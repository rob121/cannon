package templatemgr

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOverrideCopiesBuiltin(t *testing.T) {
	root := t.TempDir()
	rel := "admin/dashboard.html"

	if err := Override(root, rel); err != nil {
		t.Fatal(err)
	}

	target := filepath.Join(root, filepath.FromSlash(rel))
	raw, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) == 0 {
		t.Fatal("expected copied template content")
	}

	if err := Override(root, rel); err == nil {
		t.Fatal("expected error when overriding existing file")
	}
}

func TestRevertOverrideRestoresBuiltin(t *testing.T) {
	root := t.TempDir()
	rel := "fr/layout.html"

	if err := Override(root, rel); err != nil {
		t.Fatal(err)
	}
	if !CanRevertOverride(root, rel) {
		t.Fatal("expected revert to be available")
	}
	if err := RevertOverride(root, rel); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel))); !os.IsNotExist(err) {
		t.Fatalf("expected override removed, stat err = %v", err)
	}
	if CanRevertOverride(root, rel) {
		t.Fatal("expected revert to be unavailable after removal")
	}
	if err := RevertOverride(root, rel); err == nil {
		t.Fatal("expected error when reverting missing override")
	}
}
