package templatemgr

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNormalizeGitURL(t *testing.T) {
	got, err := NormalizeGitURL("https://github.com/example/cannon-theme.git")
	if err != nil {
		t.Fatal(err)
	}
	if got != "https://github.com/example/cannon-theme.git" {
		t.Fatalf("got %q", got)
	}
	got, err = NormalizeGitURL("git@github.com:example/cannon-theme.git")
	if err != nil {
		t.Fatal(err)
	}
	if got != "git@github.com:example/cannon-theme.git" {
		t.Fatalf("got %q", got)
	}
	if _, err := NormalizeGitURL("file:///tmp/repo"); err == nil {
		t.Fatal("expected file URL rejection")
	}
}

func TestThemeNameFromRepoURL(t *testing.T) {
	if got := ThemeNameFromRepoURL("https://github.com/acme/My Cool Theme.git"); got != "my-cool-theme" {
		t.Fatalf("got %q", got)
	}
	if got := ThemeNameFromRepoURL("git@github.com:acme/site-theme.git"); got != "site-theme" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveThemeSourceRootSingleDirectory(t *testing.T) {
	root := t.TempDir()
	cloneDir := filepath.Join(root, "clone")
	inner := filepath.Join(cloneDir, "theme-pack")
	if err := os.MkdirAll(filepath.Join(inner, "assets"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(inner, "layout.html"), []byte("<html></html>"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := resolveThemeSourceRoot(cloneDir)
	if err != nil {
		t.Fatal(err)
	}
	if got != inner {
		t.Fatalf("got %q want %q", got, inner)
	}
}

func TestCopyFlatCloneSkipsGitMetadata(t *testing.T) {
	src := t.TempDir()
	dest := t.TempDir()
	if err := os.MkdirAll(filepath.Join(src, ".git", "objects"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, ".git", "HEAD"), []byte("ref: refs/heads/main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "layout.html"), []byte("<html></html>"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := copyFlatClone(src, dest); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dest, "layout.html")); err != nil {
		t.Fatalf("layout missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dest, ".git")); !os.IsNotExist(err) {
		t.Fatalf(".git should not be copied: %v", err)
	}
}
