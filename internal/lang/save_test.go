package lang

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rob121/cannon/internal/config"
)

func TestSaveNewKeyExistingSection(t *testing.T) {
	dir := t.TempDir()
	adminPath := filepath.Join(dir, "en-US-admin.ini")
	if err := os.WriteFile(adminPath, []byte("[dashboard]\ntitle = Dashboard\n"), 0644); err != nil {
		t.Fatal(err)
	}

	site := &config.SiteConfig{LanguageDir: dir}
	mgr, err := NewManager(site, "en-US")
	if err != nil {
		t.Fatal(err)
	}

	if err := mgr.Save("admin", "dashboard", "welcome", "Welcome"); err != nil {
		t.Fatalf("Save: %v", err)
	}

	entries := mgr.Entries("admin")
	found := false
	for _, e := range entries {
		if e[0] == "dashboard.welcome" && e[1] == "Welcome" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected dashboard.welcome in entries, got %#v", entries)
	}
}

func TestSaveEmptySection(t *testing.T) {
	dir := t.TempDir()
	site := &config.SiteConfig{LanguageDir: dir}
	mgr, err := NewManager(site, "en-US")
	if err != nil {
		t.Fatal(err)
	}

	if err := mgr.Save("admin", "", "standalone", "Value"); err != nil {
		t.Fatalf("Save empty section: %v", err)
	}

	entries := mgr.Entries("admin")
	found := false
	for _, e := range entries {
		if e[0] == "standalone" && e[1] == "Value" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected standalone in entries, got %#v", entries)
	}
}

func TestReplaceScope(t *testing.T) {
	dir := t.TempDir()
	adminPath := filepath.Join(dir, "en-US-admin.ini")
	if err := os.WriteFile(adminPath, []byte("[dashboard]\ntitle = Old\n"), 0644); err != nil {
		t.Fatal(err)
	}

	site := &config.SiteConfig{LanguageDir: dir}
	mgr, err := NewManager(site, "en-US")
	if err != nil {
		t.Fatal(err)
	}

	sections := map[string]map[string]string{
		"dashboard": {"title": "Dashboard", "welcome": "Welcome"},
		"":          {"standalone": "Value"},
	}
	if err := mgr.ReplaceScope("admin", sections); err != nil {
		t.Fatalf("ReplaceScope: %v", err)
	}

	got := mgr.Sections("admin")
	if len(got) != 2 {
		t.Fatalf("expected 2 sections, got %d", len(got))
	}

	entries := mgr.Entries("admin")
	want := map[string]string{
		"dashboard.title":   "Dashboard",
		"dashboard.welcome": "Welcome",
		"standalone":        "Value",
	}
	for k, v := range want {
		if mgr.bundles["admin"].Values[k] != v {
			t.Fatalf("expected %q=%q, got %q", k, v, mgr.bundles["admin"].Values[k])
		}
	}
	if len(entries) != len(want) {
		t.Fatalf("expected %d entries, got %#v", len(want), entries)
	}
}

func TestFilesOrdering(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"en-US-site.ini", "en-US-admin.ini", "en-US.ini"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("[x]\ny = z\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	site := &config.SiteConfig{LanguageDir: dir}
	mgr, err := NewManager(site, "en-US")
	if err != nil {
		t.Fatal(err)
	}

	files := mgr.Files()
	if len(files) != 3 {
		t.Fatalf("expected 3 files, got %d", len(files))
	}
	names := []string{files[0].Filename, files[1].Filename, files[2].Filename}
	want := []string{"en-US-admin.ini", "en-US-site.ini", "en-US.ini"}
	for i, n := range want {
		if names[i] != n {
			t.Fatalf("file[%d]: want %q, got %q (all: %#v)", i, n, names[i], names)
		}
	}
}

func TestScopeFromFilename(t *testing.T) {
	cases := map[string]string{
		"en-US.ini":       "default",
		"en-US-admin.ini": "admin",
		"en-US-site.ini":  "site",
	}
	for name, want := range cases {
		if got := scopeFromFilename("en-US", name); got != want {
			t.Fatalf("%s: want scope %q, got %q", name, want, got)
		}
	}
}
