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
