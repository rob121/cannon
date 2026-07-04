package templateengine

import (
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
	e := New(nil, themes.Selection{}, nil, nil, testAdminFuncs())
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
	e := New(nil, themes.Selection{}, nil, nil, testAdminFuncs())
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
