package templateengine

import (
	"context"
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rob121/cannon/internal/config"
	"github.com/rob121/cannon/internal/lang"
	"github.com/rob121/cannon/internal/themes"
)

func TestLayoutOfflineNotice(t *testing.T) {
	e := New(&config.SiteConfig{Name: "Demo"}, themes.Selection{}, nil, nil, testLayoutFuncs(true))
	e.SetHookContext(context.Background())
	defer e.SetHookContext(nil)

	var buf strings.Builder
	if err := e.Render(&buf, "default/layout.html", "default/page.html", map[string]any{
		"Title": "Home",
	}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if strings.Count(out, "site-offline-notice") != 1 {
		t.Fatalf("expected one offline notice, got %d in: %s", strings.Count(out, "site-offline-notice"), out)
	}
	if !strings.Contains(out, "Preview mode.") {
		t.Fatal("expected offline notice message")
	}
}

func TestLayoutOfflineNoticeInjectedForCustomTheme(t *testing.T) {
	dir := t.TempDir()
	themeDir := filepath.Join(dir, "fr")
	if err := os.MkdirAll(themeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	customLayout := `<!doctype html><html><body class="fr-theme"><nav>Nav</nav>{{.Main}}</body></html>`
	if err := os.WriteFile(filepath.Join(themeDir, "layout.html"), []byte(customLayout), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(themeDir, "template.json"), []byte(`{"name":"FR","type":"frontend","status":"active"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	e := New(&config.SiteConfig{TemplateDir: dir, Name: "Demo"}, themes.Selection{Frontend: "fr"}, nil, nil, testLayoutFuncs(true))
	e.SetHookContext(context.Background())
	defer e.SetHookContext(nil)

	var buf strings.Builder
	if err := e.Render(&buf, "default/layout.html", "default/page.html", map[string]any{
		"Title":     "Home",
		"IsOffline": true,
	}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "site-offline-notice") {
		t.Fatalf("expected injected offline notice for custom theme layout: %s", out)
	}
}

func TestInjectAfterBodyOpen(t *testing.T) {
	got := string(injectAfterBodyOpen([]byte(`<html><body class="x"><main></main></body></html>`), `<div class="site-offline-notice"></div>`))
	if !strings.Contains(got, `<body class="x"><div class="site-offline-notice"></div><main>`) {
		t.Fatalf("unexpected inject output: %s", got)
	}
}

func testLayoutFuncs(offline bool) template.FuncMap {
	funcs := template.FuncMap{
		"isOffline":           func() bool { return offline },
		"siteName":            func() string { return "Demo" },
		"siteMetaDescription": func() string { return "" },
		"siteMetaKeywords":    func() string { return "" },
		"siteOGTitle":         func() string { return "" },
		"siteOGImage":         func() string { return "" },
		"siteTwitterCard":     func() string { return "summary_large_image" },
		"siteTwitterSite":     func() string { return "" },
		"siteTwitterCreator":  func() string { return "" },
		"siteHeadExtra":       func() template.HTML { return "" },
		"year":                func() int { return 2026 },
		"menu":                func(string) ([]map[string]any, error) { return nil, nil },
		"lenspace":            func(string) (int, error) { return 0, nil },
		"space":               func(string) (template.HTML, error) { return "", nil },
		"controllerURL":       func(string, string) string { return "#" },
	}
	for k, v := range lang.TestFuncMap() {
		funcs[k] = v
	}
	return funcs
}
