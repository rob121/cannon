package templateengine

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/rob121/cannon/internal/config"
	"github.com/rob121/cannon/internal/hooks"
	"github.com/rob121/cannon/internal/themes"
)

func TestRenderOnAfterRenderCanReplaceBodyAndHeaders(t *testing.T) {
	dir := t.TempDir()
	themeDir := filepath.Join(dir, "mysite")
	if err := os.MkdirAll(themeDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(themeDir, "layout.html"), []byte(`<html>{{.Main}}</html>`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(themeDir, "page.html"), []byte(`<main>{{.Message}}</main>`), 0644); err != nil {
		t.Fatal(err)
	}

	e := New(&config.SiteConfig{TemplateDir: dir}, themes.Selection{Frontend: "mysite"}, nil, nil, nil)
	ctx := hooks.WithFire(context.Background(), func(_ context.Context, event string, args map[string]any) (map[string]any, error) {
		if event != hooks.OnAfterRender {
			return args, nil
		}
		if args["body"] != `<html><main>Hello</main></html>` {
			t.Fatalf("unexpected render body: %#v", args["body"])
		}
		return map[string]any{
			"body_encoding": "base64",
			"body_base64":   base64.StdEncoding.EncodeToString([]byte("compressed bytes")),
			"headers": map[string][]string{
				"Content-Encoding": {"gzip"},
				"Vary":             {"Accept-Encoding"},
			},
		}, nil
	})
	e.SetHookContext(ctx)
	rec := httptest.NewRecorder()

	if err := e.Render(rec, "default/layout.html", "default/page.html", map[string]any{"Message": "Hello"}); err != nil {
		t.Fatal(err)
	}

	if rec.Body.String() != "compressed bytes" {
		t.Fatalf("expected replaced body, got %q", rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
		t.Fatalf("expected html content type, got %q", got)
	}
	if got := rec.Header().Get("Content-Encoding"); got != "gzip" {
		t.Fatalf("expected gzip content encoding, got %q", got)
	}
	if got := rec.Header().Get("Vary"); got != "Accept-Encoding" {
		t.Fatalf("expected vary Accept-Encoding, got %q", got)
	}
}

func TestRenderErrorOnAfterRenderAppliesHeaders(t *testing.T) {
	dir := t.TempDir()
	themeDir := filepath.Join(dir, "mysite")
	if err := os.MkdirAll(themeDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(themeDir, "layout.html"), []byte(`<html>{{.Main}}</html>`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(themeDir, "error.html"), []byte(`<main>missing</main>`), 0644); err != nil {
		t.Fatal(err)
	}

	e := New(&config.SiteConfig{TemplateDir: dir}, themes.Selection{Frontend: "mysite"}, nil, nil, nil)
	ctx := hooks.WithFire(context.Background(), func(_ context.Context, event string, args map[string]any) (map[string]any, error) {
		if event != hooks.OnAfterRender {
			return args, nil
		}
		return map[string]any{
			"body_encoding": "base64",
			"body_base64":   base64.StdEncoding.EncodeToString([]byte("compressed error")),
			"headers": map[string][]string{
				"Content-Encoding": {"gzip"},
			},
		}, nil
	})
	e.SetHookContext(ctx)
	rec := httptest.NewRecorder()

	if err := e.RenderError(rec, http.StatusNotFound, map[string]any{
		"Title":        "Not Found",
		"ErrorCode":    404,
		"ErrorMessage": "missing page",
	}); err != nil {
		t.Fatal(err)
	}

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
	if rec.Body.String() != "compressed error" {
		t.Fatalf("expected replaced body, got %q", rec.Body.String())
	}
	if got := rec.Header().Get("Content-Encoding"); got != "gzip" {
		t.Fatalf("expected gzip content encoding, got %q", got)
	}
}

func TestRenderOnAfterRenderCanReplaceTextBody(t *testing.T) {
	dir := t.TempDir()
	themeDir := filepath.Join(dir, "mysite")
	if err := os.MkdirAll(themeDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(themeDir, "layout.html"), []byte(`{{.Main}}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(themeDir, "page.html"), []byte(`before`), 0644); err != nil {
		t.Fatal(err)
	}

	e := New(&config.SiteConfig{TemplateDir: dir}, themes.Selection{Frontend: "mysite"}, nil, nil, nil)
	ctx := hooks.WithFire(context.Background(), func(_ context.Context, event string, args map[string]any) (map[string]any, error) {
		if event == hooks.OnAfterRender {
			args["body"] = "after"
		}
		return args, nil
	})
	e.SetHookContext(ctx)
	rec := httptest.NewRecorder()

	if err := e.Render(rec, "default/layout.html", "default/page.html", nil); err != nil {
		t.Fatal(err)
	}

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if rec.Body.String() != "after" {
		t.Fatalf("expected replaced body, got %q", rec.Body.String())
	}
}
