package admin

import (
	"net/http/httptest"
	"testing"
)

func TestSafeExtensionAdminRedirectLocation(t *testing.T) {
	r := httptest.NewRequest("GET", "https://cms.example/admin/extension-apps/demo/settings", nil)
	r.URL.Host = "cms.example"
	r.URL.Scheme = "https"

	if loc, ok := SafeExtensionAdminRedirectLocation("demo", "/admin/extension-apps/demo/page", r); !ok || loc != "/admin/extension-apps/demo/page" {
		t.Fatalf("relative ok path: loc=%q ok=%v", loc, ok)
	}
	if _, ok := SafeExtensionAdminRedirectLocation("demo", "https://evil.example/admin/extension-apps/demo/x", r); ok {
		t.Fatal("expected off-site redirect to be blocked")
	}
	if _, ok := SafeExtensionAdminRedirectLocation("demo", "https://cms.example/admin/users", r); ok {
		t.Fatal("expected redirect outside extension path to be blocked")
	}
	if loc, ok := SafeExtensionAdminRedirectLocation("demo", "https://cms.example/admin/extension-apps/demo/sub", r); !ok || loc == "" {
		t.Fatalf("same-origin extension path: loc=%q ok=%v", loc, ok)
	}
}
