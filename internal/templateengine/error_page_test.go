package templateengine

import (
	"fmt"
	"html/template"
	"strings"
	"testing"

	"github.com/rob121/cannon/internal/themes"
)

func TestResolveErrorTemplate(t *testing.T) {
	e := New(nil, themes.Selection{}, nil, nil, nil)
	if got := e.ResolveErrorTemplate(404); got != "default/error/404.html" {
		t.Fatalf("404 template = %q", got)
	}
	if got := e.ResolveErrorTemplate(418); got != "default/error.html" {
		t.Fatalf("fallback template = %q", got)
	}
}

func TestResolveAdminErrorTemplate(t *testing.T) {
	e := New(nil, themes.Selection{}, nil, nil, TestAdminFuncs())
	if got := e.ResolveAdminErrorTemplate(404); got != "admin/error/404.html" {
		t.Fatalf("admin 404 template = %q", got)
	}
}

func TestErrorTemplate404(t *testing.T) {
	e := New(nil, themes.Selection{}, nil, nil, testFrontendFuncs())
	tmpl, err := e.parse("default/error/404.html")
	if err != nil {
		t.Fatal(err)
	}
	var buf strings.Builder
	err = tmpl.Execute(&buf, map[string]any{
		"Title":        "Not Found",
		"ErrorCode":    404,
		"ErrorMessage": "item not found",
		"HomeURL":      "/",
	})
	if err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "item not found") || !strings.Contains(out, "Return Home") {
		t.Fatalf("unexpected 404 page: %s", out)
	}
}

func TestAdminErrorTemplate404(t *testing.T) {
	e := New(nil, themes.Selection{}, nil, nil, TestAdminFuncs())
	tmpl, err := e.parseAdmin("admin/error/404.html")
	if err != nil {
		t.Fatal(err)
	}
	var buf strings.Builder
	err = tmpl.Execute(&buf, map[string]any{
		"Title":        "Not Found",
		"ErrorStatus":  404,
		"ErrorMessage": "This item could not be found.",
		"ErrorHint":    "Check the URL or return to the dashboard.",
		"ErrorIcon":    "bi-search",
		"BackURL":      "/admin",
	})
	if err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, part := range []string{
		"admin-error-page",
		"This item could not be found.",
		"Back to Dashboard",
	} {
		if !strings.Contains(out, part) {
			t.Fatalf("missing %q in admin 404 page: %s", part, out)
		}
	}
}

func TestDefaultErrorTemplateFallback(t *testing.T) {
	e := New(nil, themes.Selection{}, nil, nil, testFrontendFuncs())
	tmpl, err := e.parse("default/error.html")
	if err != nil {
		t.Fatal(err)
	}
	var buf strings.Builder
	err = tmpl.Execute(&buf, map[string]any{
		"Title":        "Not Found",
		"ErrorCode":    404,
		"ErrorMessage": "item not found",
		"HomeURL":      "/",
	})
	if err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "item not found") || !strings.Contains(out, "Return Home") {
		t.Fatalf("unexpected error page: %s", out)
	}
}

func TestDefaultErrorMessage(t *testing.T) {
	if DefaultErrorMessage(404) == "" || DefaultErrorMessage(500) == "" {
		t.Fatal("expected default messages")
	}
}

func TestPublicErrorMessageHidesTemplateInternals(t *testing.T) {
	raw := `template: layout.html:103:20: executing "layout.html" at <space "sidebar">: error calling space: extension cannon-ext-contact has no block capability`
	if got := PublicErrorMessage(500, raw); got != DefaultErrorMessage(500) {
		t.Fatalf("got %q", got)
	}
	if got := PublicErrorMessage(404, "item not found"); got != "item not found" {
		t.Fatalf("got %q", got)
	}
}

func TestRenderErrorFallsBackWhenLayoutFails(t *testing.T) {
	e := New(nil, themes.Selection{}, func(string) (template.HTML, error) {
		return "", fmt.Errorf("extension cannon-ext-contact has no block capability")
	}, nil, testFrontendFuncs())
	var buf strings.Builder
	// default/layout.html calls {{space "sidebar"}}, so layout render fails and
	// RenderError must fall back to the standalone document.
	err := e.RenderError(&buf, 500, map[string]any{
		"Title":        "Server Error",
		"ErrorCode":    500,
		"ErrorMessage": DefaultErrorMessage(500),
		"HomeURL":      "/",
	})
	if err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{"Server Error", "Return Home", DefaultErrorMessage(500), "<!doctype html>"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in standalone error: %s", want, out)
		}
	}
	if strings.Contains(out, "cannon-ext-contact") {
		t.Fatalf("standalone page leaked technical detail: %s", out)
	}
}
