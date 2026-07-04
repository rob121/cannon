package lang

import (
	"html/template"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTranslationPreviewActive(t *testing.T) {
	r := httptest.NewRequest("GET", "/?tp=1", nil)
	if !TranslationPreviewActive(r) {
		t.Fatal("expected tp=1 to enable translation preview")
	}
	r = httptest.NewRequest("GET", "/", nil)
	if TranslationPreviewActive(r) {
		t.Fatal("expected translation preview off without tp=1")
	}
}

func TestFormatLangPreview(t *testing.T) {
	mgr, err := NewEmbeddedManager("en-US")
	if err != nil {
		t.Fatal(err)
	}

	if got := formatLang(mgr, "nav.sign_in", true); got != "Sign in (nav.sign_in)" {
		t.Fatalf("preview: %q", got)
	}
	if got := formatLang(mgr, "nav.sign_in", false); got != "Sign in" {
		t.Fatalf("normal: %q", got)
	}
	if got := formatLang(mgr, "missing.key", true); got != "missing.key" {
		t.Fatalf("missing key should not duplicate: %q", got)
	}
}

func TestFormatLangPreviewPerCall(t *testing.T) {
	mgr, err := NewEmbeddedManager("en-US")
	if err != nil {
		t.Fatal(err)
	}

	got := formatLang(mgr, "auth.signed_in_as", false, "tp", "1", "Username", "jane")
	if got != "Signed in as jane (auth.signed_in_as)" {
		t.Fatalf("per-call tp: %q", got)
	}
}

func TestLangTemplateFunc(t *testing.T) {
	funcs := FuncMap(mustEmbeddedManager(t), true)
	langFn, ok := funcs["lang"].(func(string, ...string) string)
	if !ok {
		t.Fatal("lang func missing")
	}
	if got := langFn("nav.profile"); got != "Profile (nav.profile)" {
		t.Fatalf("template lang: %q", got)
	}
}

func TestLangTemplateFuncRenders(t *testing.T) {
	tpl, err := template.New("test").Funcs(FuncMap(mustEmbeddedManager(t), true)).Parse(`{{lang "nav.sign_in"}}`)
	if err != nil {
		t.Fatal(err)
	}
	var buf strings.Builder
	if err := tpl.Execute(&buf, nil); err != nil {
		t.Fatal(err)
	}
	if got := buf.String(); got != "Sign in (nav.sign_in)" {
		t.Fatalf("rendered: %q", got)
	}
}

func mustEmbeddedManager(t *testing.T) *Manager {
	t.Helper()
	mgr, err := NewEmbeddedManager("en-US")
	if err != nil {
		t.Fatal(err)
	}
	return mgr
}
