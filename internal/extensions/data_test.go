package extensions

import (
	"testing"
)

func TestParseExtensionDataPath(t *testing.T) {
	hash, dataPath, ok := ParseExtensionDataPath("/ext/abc123/contact/submit")
	if !ok || hash != "abc123" || dataPath != "contact/submit" {
		t.Fatalf("parse: hash=%q dataPath=%q ok=%v", hash, dataPath, ok)
	}
	if _, _, ok := ParseExtensionDataPath("/ext/abc123"); ok {
		t.Fatal("expected false without data path")
	}
	if _, _, ok := ParseExtensionDataPath("/contact/submit"); ok {
		t.Fatal("expected false without /ext prefix")
	}
}

func TestRouteHashMatchesSocketBasename(t *testing.T) {
	name := "cannon-ext-contact"
	siteID := "example"
	hash := RouteHash(name, siteID)
	socket := "/tmp/sockets/" + hash + ".sock"
	if RouteHashFromSocket(socket) != hash {
		t.Fatalf("socket basename: got %q want %q", RouteHashFromSocket(socket), hash)
	}
}

func TestPublicDataURL(t *testing.T) {
	got := PublicDataURL("demo", "site-1", "contact/submit")
	want := "/ext/" + RouteHash("demo", "site-1") + "/contact/submit"
	if got != want {
		t.Fatalf("PublicDataURL: got %q want %q", got, want)
	}
}
