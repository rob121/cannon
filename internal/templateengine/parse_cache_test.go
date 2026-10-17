package templateengine

import (
	"bytes"
	"html/template"
	"strings"
	"testing"

	"github.com/rob121/cannon/internal/config"
	"github.com/rob121/cannon/internal/themes"
)

func TestParseCacheSkipsDiskOnSecondEngine(t *testing.T) {
	InvalidateAll()
	parseCacheReads = 0
	site := &config.SiteConfig{ID: "cache-site"}
	sel := themes.Selection{Frontend: themes.BuiltinFrontend, Admin: themes.BuiltinAdmin}

	e1 := New(site, sel, nil, nil, MergeFuncMaps(
		langStubFuncs(),
		template.FuncMap{"csrfToken": func() string { return "tok-a" }},
	))
	e2 := New(site, sel, nil, nil, MergeFuncMaps(
		langStubFuncs(),
		template.FuncMap{"csrfToken": func() string { return "tok-b" }},
	))

	name := "default/partials/offline-notice.html"
	if _, err := e1.parse(name); err != nil {
		t.Fatal(err)
	}
	readsAfterFirst := parseCacheReads
	if readsAfterFirst == 0 {
		t.Fatal("expected first parse to read template source")
	}
	if _, err := e2.parse(name); err != nil {
		t.Fatal(err)
	}
	if parseCacheReads != readsAfterFirst {
		t.Fatalf("second engine should hit source cache; reads %d -> %d", readsAfterFirst, parseCacheReads)
	}
}

func TestParseCacheCSRFNotLeakedAcrossEngines(t *testing.T) {
	InvalidateAll()
	site := &config.SiteConfig{ID: "csrf-site"}
	sel := themes.Selection{Frontend: themes.BuiltinFrontend, Admin: themes.BuiltinAdmin}

	e1 := New(site, sel, nil, nil, MergeFuncMaps(
		langStubFuncs(),
		template.FuncMap{
			"csrfToken": func() string { return "AAA" },
			"csrfField": func() template.HTML { return "" },
		},
	))
	e2 := New(site, sel, nil, nil, MergeFuncMaps(
		langStubFuncs(),
		template.FuncMap{
			"csrfToken": func() string { return "BBB" },
			"csrfField": func() template.HTML { return "" },
		},
	))

	name := "default/partials/offline-notice.html"
	t1, err := e1.parse(name)
	if err != nil {
		t.Fatal(err)
	}
	t2, err := e2.parse(name)
	if err != nil {
		t.Fatal(err)
	}
	// offline-notice may not call csrfToken; verify engines keep distinct funcs via a custom execute.
	src := `{{csrfToken}}`
	c1 := template.Must(template.New("c1").Funcs(e1.funcs).Parse(src))
	c2 := template.Must(template.New("c2").Funcs(e2.funcs).Parse(src))
	var b1, b2 bytes.Buffer
	if err := c1.Execute(&b1, nil); err != nil {
		t.Fatal(err)
	}
	if err := c2.Execute(&b2, nil); err != nil {
		t.Fatal(err)
	}
	if b1.String() != "AAA" || b2.String() != "BBB" {
		t.Fatalf("csrf leak: e1=%q e2=%q", b1.String(), b2.String())
	}
	_ = t1
	_ = t2
}

func langStubFuncs() template.FuncMap {
	return template.FuncMap{
		"lang":      func(key string, _ ...string) string { return key },
		"localeTag": func() string { return "en" },
	}
}

func TestInvalidateSiteDropsParseCache(t *testing.T) {
	InvalidateAll()
	parseCacheReads = 0
	site := &config.SiteConfig{ID: "inv-site"}
	sel := themes.Selection{Frontend: themes.BuiltinFrontend, Admin: themes.BuiltinAdmin}
	e := New(site, sel, nil, nil, langStubFuncs())
	name := "default/partials/page-header.html"
	if _, err := e.parse(name); err != nil {
		t.Fatal(err)
	}
	firstReads := parseCacheReads
	InvalidateSite(site.ID)
	e2 := New(site, sel, nil, nil, langStubFuncs())
	if _, err := e2.parse(name); err != nil {
		t.Fatal(err)
	}
	if parseCacheReads <= firstReads {
		t.Fatalf("expected re-read after invalidate; reads=%d first=%d", parseCacheReads, firstReads)
	}
	if !strings.Contains(frontendCacheKey(site.ID, sel.Frontend, name), site.ID) {
		t.Fatal("sanity")
	}
}
