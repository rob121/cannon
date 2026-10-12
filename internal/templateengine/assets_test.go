package templateengine

import (
	"strings"
	"testing"
)

func TestEmbeddedDefaultSiteAssets(t *testing.T) {
	for _, name := range []string{"site.css", "site-mfa.js", "site-analytics.js"} {
		raw, err := SiteAsset(name)
		if err != nil {
			t.Fatalf("SiteAsset(%q): %v", name, err)
		}
		if len(raw) == 0 {
			t.Fatalf("SiteAsset(%q) returned empty payload", name)
		}
	}
	raw, err := SiteAsset("site-analytics.js")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "cannon-realtime-config") {
		t.Fatalf("site-analytics.js missing realtime config hook")
	}
}

func TestEmbeddedAdminCKEditorAssets(t *testing.T) {
	for _, name := range []string{"ckeditor5.umd.js", "ckeditor5.css"} {
		raw, err := AdminAsset(name)
		if err != nil {
			t.Fatalf("AdminAsset(%q): %v", name, err)
		}
		if len(raw) == 0 {
			t.Fatalf("AdminAsset(%q) returned empty payload", name)
		}
	}
	raw, err := AdminAsset("ckeditor5.umd.js")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "GeneralHtmlSupport") {
		t.Fatalf("ckeditor5.umd.js missing GeneralHtmlSupport")
	}
}
