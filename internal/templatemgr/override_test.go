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
