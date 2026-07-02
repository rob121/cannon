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
	"github.com/rob121/cannon/internal/templatemgr"
)

func TestAppendQueryParam(t *testing.T) {
	got := appendQueryParam("/admin/templates/mysite", "error", "template not found")
	if !strings.Contains(got, "error=template+not+found") {
		t.Fatalf("got %q", got)
	}
	got = appendQueryParam("/admin/templates/mysite?sort=name&dir=asc", "error", "failed")
	if !strings.HasPrefix(got, "/admin/templates/mysite?sort=name&dir=asc&error=") {
		t.Fatalf("got %q", got)
	}
}

func TestTemplateEditUsesFormPath(t *testing.T) {
	root := t.TempDir()
	site := &config.SiteConfig{TemplateDir: root}
	h := &Handler{}

	path := "mysite/dashboard.html"
	if err := os.MkdirAll(filepath.Join(root, "mysite"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "mysite", "dashboard.html"), []byte("<html>old</html>"), 0644); err != nil {
		t.Fatal(err)
	}

	body := url.Values{
		"path":    {path},
		"content": {"<html>updated</html>"},
	}
	req := httptest.NewRequest("POST", "/admin/templates/edit", strings.NewReader(body.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	h.templateEdit(rec, req, site)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303; body=%q", rec.Code, rec.Body.String())
	}
	raw, err := os.ReadFile(filepath.Join(root, "mysite", "dashboard.html"))
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != "<html>updated</html>" {
		t.Fatalf("content = %q", raw)
	}
}

func TestTemplateEditBuiltinCreatesOverride(t *testing.T) {
	root := t.TempDir()
	site := &config.SiteConfig{TemplateDir: root}
	h := &Handler{}

	if err := os.MkdirAll(filepath.Join(root, "mysite"), 0755); err != nil {
		t.Fatal(err)
	}

	path := "mysite/layout.html"
	body := url.Values{
		"path":    {path},
		"content": {"<html>custom layout</html>"},
	}
	req := httptest.NewRequest("POST", "/admin/templates/edit", strings.NewReader(body.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	h.templateEdit(rec, req, site)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303; body=%q", rec.Code, rec.Body.String())
	}
	if _, err := os.Stat(filepath.Join(root, "mysite", "layout.html")); err != nil {
		t.Fatalf("expected override on disk: %v", err)
	}
}

func TestTemplateOverrideUsesFormPath(t *testing.T) {
	root := t.TempDir()
	site := &config.SiteConfig{TemplateDir: root}
	h := &Handler{}

	if err := os.MkdirAll(filepath.Join(root, "mysite"), 0755); err != nil {
		t.Fatal(err)
	}

	body := url.Values{
		"path":  {"mysite/layout.html"},
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
	if !strings.HasPrefix(loc, "/admin/templates/mysite") {
		t.Fatalf("location = %q", loc)
	}
	if _, err := os.Stat(filepath.Join(root, "mysite", "layout.html")); err != nil {
		t.Fatalf("expected override on disk: %v", err)
	}
}

func TestTemplateRevertUsesFormPath(t *testing.T) {
	root := t.TempDir()
	site := &config.SiteConfig{TemplateDir: root}
	h := &Handler{}

	if err := os.MkdirAll(filepath.Join(root, "fr"), 0755); err != nil {
		t.Fatal(err)
	}
	path := "fr/layout.html"
	if err := templatemgr.Override(root, path); err != nil {
		t.Fatal(err)
	}

	body := url.Values{"path": {path}}
	req := httptest.NewRequest("POST", "/admin/templates/revert?sort=name&dir=asc", strings.NewReader(body.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	h.templateRevert(rec, req, site)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303; body=%q", rec.Code, rec.Body.String())
	}
	if _, err := os.Stat(filepath.Join(root, "fr", "layout.html")); !os.IsNotExist(err) {
		t.Fatalf("expected override removed: %v", err)
	}
	loc := rec.Header().Get("Location")
	if !strings.HasPrefix(loc, "/admin/templates/fr") {
		t.Fatalf("location = %q", loc)
	}
}
