package appupdate

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNormalizeLiveBinaryPathStripsPrevious(t *testing.T) {
	got := normalizeLiveBinaryPath("/usr/local/bin/cannon.previous")
	if got != "/usr/local/bin/cannon" {
		t.Fatalf("got %q", got)
	}
	got = normalizeLiveBinaryPath("/usr/local/bin/cannon.previous.latest")
	if got != "/usr/local/bin/cannon" {
		t.Fatalf("got %q", got)
	}
}

func TestLatestPathNotNestedUnderPrevious(t *testing.T) {
	got := latestPath("/usr/local/bin/cannon.previous")
	if got != "/usr/local/bin/cannon.latest" {
		t.Fatalf("got %q", got)
	}
}

func TestFindSiblingBesideExecutable(t *testing.T) {
	root := t.TempDir()
	bin := filepath.Join(root, "cannon")
	prev := filepath.Join(root, "cannon.previous")
	if err := os.WriteFile(bin, []byte("live"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(prev, []byte("old"), 0755); err != nil {
		t.Fatal(err)
	}
	got, ok := findSibling(bin, ".previous")
	if !ok || got != prev {
		t.Fatalf("got %q ok=%v", got, ok)
	}
}

func TestFindSiblingViaSymlink(t *testing.T) {
	root := t.TempDir()
	realDir := filepath.Join(root, "real")
	linkDir := filepath.Join(root, "link")
	if err := os.MkdirAll(realDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(linkDir, 0755); err != nil {
		t.Fatal(err)
	}
	realBin := filepath.Join(realDir, "cannon")
	prev := filepath.Join(realDir, "cannon.previous")
	linkBin := filepath.Join(linkDir, "cannon")
	if err := os.WriteFile(realBin, []byte("live"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(prev, []byte("old"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(realBin, linkBin); err != nil {
		t.Fatal(err)
	}
	got, ok := findSibling(linkBin, ".previous")
	want, _ := filepath.EvalSymlinks(prev)
	gotResolved, _ := filepath.EvalSymlinks(got)
	if !ok || gotResolved != want {
		t.Fatalf("got %q ok=%v want %q", got, ok, prev)
	}
}
