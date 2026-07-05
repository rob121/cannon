package extension

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHookAbort(t *testing.T) {
	resp := HookAbort("denied")
	if !resp.Stop {
		t.Fatal("expected stop")
	}
	args := HookArguments(HookWireRequest{Arguments: resp.Arguments})
	if args["allowed"] != false || args["error"] != "denied" {
		t.Fatalf("args=%v", args)
	}
}

func TestHookFragmentHelpers(t *testing.T) {
	args := map[string]any{
		"head_html":      `<script src="a.js"></script>`,
		"body_html":      `<script src="b.js"></script>`,
		"robots_append":  "Disallow: /private/",
		"sitemap_urls":   []map[string]any{{"loc": "/x"}},
	}
	if HookHeadHTML(args) == "" || HookBodyHTML(args) == "" || HookRobotsAppend(args) == "" {
		t.Fatal("expected fragment helpers to read values")
	}
	if len(HookSitemapURLs(args)) != 1 {
		t.Fatalf("sitemap urls: %v", HookSitemapURLs(args))
	}
}

func TestDecodeHookWireRequest(t *testing.T) {
	body := `{"event":"onBeforeRoute","arguments":{"path":"/"},"site_id":"demo","method":"GET","url":"/"}`
	req := httptest.NewRequest("POST", "/hooks", strings.NewReader(body))
	req.ContentLength = int64(len(body))
	got, err := DecodeHookWireRequest(req, "fallback")
	if err != nil {
		t.Fatal(err)
	}
	if got.Event != "onBeforeRoute" || got.SiteID != "demo" {
		t.Fatalf("req=%+v", got)
	}
}
