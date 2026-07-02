package admin

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rob121/cannon/internal/config"
)

func TestAppendQueryParam(t *testing.T) {
	got := appendQueryParam("/admin/templates/default", "error", "template not found: admin/layout.html")
	if !strings.Contains(got, "error=template+not+found") {
		t.Fatalf("got %q", got)
	}
	got = appendQueryParam("/admin/templates/default?sort=name&dir=asc", "error", "failed")
	if !strings.HasPrefix(got, "/admin/templates/default?sort=name&dir=asc&error=") {
		t.Fatalf("got %q", got)
	}
}

func TestTemplateOverrideUsesFormPath(t *testing.T) {
	root := t.TempDir()
	site := &config.SiteConfig{TemplateDir: root}
	h := &Handler{}

	body := url.Values{
		"path": {"admin/dashboard.html"},
		"_csrf": {"ignored-by-unit-test"},
	}
	req := httptest.NewRequest("POST", "/admin/templates/override?sort=name&dir=asc", strings.NewReader(body.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	h.templateOverride(rec, req, site)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303; body=%q", rec.Code, rec.Body.String())
	}
	loc := rec.Header().Get("Location")
	if !strings.HasPrefix(loc, "/admin/templates/admin") {
		t.Fatalf("location = %q", loc)
	}
	if _, err := os.Stat(filepath.Join(root, "admin", "dashboard.html")); err != nil {
		t.Fatalf("expected override on disk: %v", err)
	}
}
