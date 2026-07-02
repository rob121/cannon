package templateengine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rob121/cannon/internal/config"
	"github.com/rob121/cannon/internal/themes"
)

func TestAdminThemeOverrideVisible(t *testing.T) {
	dir := t.TempDir()
	themeDir := filepath.Join(dir, "myadmin")
	if err := os.MkdirAll(themeDir, 0755); err != nil {
		t.Fatal(err)
	}
	marker := "SITE-OVERRIDE-MARKER"
	if err := os.WriteFile(filepath.Join(themeDir, "dashboard.html"), []byte("<h1>"+marker+"</h1>"), 0644); err != nil {
		t.Fatal(err)
	}
	e := New(&config.SiteConfig{TemplateDir: dir}, themes.Selection{Admin: "myadmin"}, nil, nil, testAdminFuncs())
	tmpl, err := e.ParseNested("admin/dashboard.html")
	if err != nil {
		t.Fatal(err)
	}
	var buf strings.Builder
	if err := tmpl.Execute(&buf, map[string]any{}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), marker) {
		t.Fatalf("expected theme override in output, got: %q", buf.String())
	}
}

func TestFrontendThemeOverrideVisible(t *testing.T) {
	dir := t.TempDir()
	themeDir := filepath.Join(dir, "mysite")
	if err := os.MkdirAll(themeDir, 0755); err != nil {
		t.Fatal(err)
	}
	marker := "DEFAULT-OVERRIDE-MARKER"
	if err := os.WriteFile(filepath.Join(themeDir, "page.html"), []byte("<p>"+marker+"</p>"), 0644); err != nil {
		t.Fatal(err)
	}
	e := New(&config.SiteConfig{TemplateDir: dir}, themes.Selection{Frontend: "mysite"}, nil, nil, nil)
	tmpl, err := e.ParseNested("default/page.html")
	if err != nil {
		t.Fatal(err)
	}
	var buf strings.Builder
	if err := tmpl.Execute(&buf, nil); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), marker) {
		t.Fatalf("expected theme override in output, got: %q", buf.String())
	}
}
