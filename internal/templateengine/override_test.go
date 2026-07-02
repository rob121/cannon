package templateengine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rob121/cannon/internal/config"
)

func TestAdminSiteOverrideVisible(t *testing.T) {
	dir := t.TempDir()
	adminDir := filepath.Join(dir, "admin")
	if err := os.MkdirAll(adminDir, 0755); err != nil {
		t.Fatal(err)
	}
	marker := "SITE-OVERRIDE-MARKER"
	if err := os.WriteFile(filepath.Join(adminDir, "dashboard.html"), []byte("<h1>"+marker+"</h1>"), 0644); err != nil {
		t.Fatal(err)
	}
	e := New(&config.SiteConfig{TemplateDir: dir}, nil, nil, testAdminFuncs())
	tmpl, err := e.ParseNested("admin/dashboard.html")
	if err != nil {
		t.Fatal(err)
	}
	var buf strings.Builder
	if err := tmpl.Execute(&buf, map[string]any{}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), marker) {
		t.Fatalf("expected site override in output, got: %q", buf.String())
	}
}

func TestDefaultSiteOverrideVisible(t *testing.T) {
	dir := t.TempDir()
	defaultDir := filepath.Join(dir, "default")
	if err := os.MkdirAll(defaultDir, 0755); err != nil {
		t.Fatal(err)
	}
	marker := "DEFAULT-OVERRIDE-MARKER"
	if err := os.WriteFile(filepath.Join(defaultDir, "page.html"), []byte("<p>"+marker+"</p>"), 0644); err != nil {
		t.Fatal(err)
	}
	e := New(&config.SiteConfig{TemplateDir: dir}, nil, nil, nil)
	tmpl, err := e.ParseNested("default/page.html")
	if err != nil {
		t.Fatal(err)
	}
	var buf strings.Builder
	if err := tmpl.Execute(&buf, nil); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), marker) {
		t.Fatalf("expected site override in output, got: %q", buf.String())
	}
}
