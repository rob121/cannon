package lang

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rob121/cannon/internal/config"
)

func TestEmbeddedSiteDefaults(t *testing.T) {
	mgr, err := NewEmbeddedManager("en-US")
	if err != nil {
		t.Fatal(err)
	}
	if got := mgr.Fmt("nav.sign_in"); got != "Sign in" {
		t.Fatalf("nav.sign_in: %q", got)
	}
	if got := mgr.Fmt("auth.signed_in_as", "Username", "jane"); got != "Signed in as jane" {
		t.Fatalf("auth.signed_in_as: %q", got)
	}
}

func TestDiskOverridesEmbedded(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "en-US-site.ini")
	if err := os.WriteFile(path, []byte("[nav]\nsign_in = \"Log in\"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	mgr, err := NewManager(&config.SiteConfig{LanguageDir: dir}, "en-US")
	if err != nil {
		t.Fatal(err)
	}
	if got := mgr.Fmt("nav.sign_in"); got != "Log in" {
		t.Fatalf("override nav.sign_in: %q", got)
	}
	if got := mgr.Fmt("nav.profile"); got != "Profile" {
		t.Fatalf("embedded fallback nav.profile: %q", got)
	}
}

func TestLocaleTag(t *testing.T) {
	mgr, err := NewEmbeddedManager("en-US")
	if err != nil {
		t.Fatal(err)
	}
	if got := mgr.LocaleTag(); got != "en" {
		t.Fatalf("LocaleTag: %q", got)
	}
}
