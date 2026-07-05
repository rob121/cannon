package hooks

import "testing"

func TestMergeArgsAppendsHTMLFragments(t *testing.T) {
	got := MergeArgs(map[string]any{"head_html": "<a>"}, map[string]any{"head_html": "<b>"})
	if got["head_html"] != "<a>\n<b>" {
		t.Fatalf("head_html: %q", got["head_html"])
	}
}

func TestMergeArgsMergesSitemapURLs(t *testing.T) {
	got := MergeArgs(
		map[string]any{"sitemap_urls": []map[string]any{{"loc": "/a"}}},
		map[string]any{"sitemap_urls": []any{map[string]any{"loc": "/b", "lastmod": "2026-01-01"}}},
	)
	rows, ok := got["sitemap_urls"].([]map[string]any)
	if !ok || len(rows) != 2 || rows[1]["loc"] != "/b" {
		t.Fatalf("sitemap_urls: %#v", got["sitemap_urls"])
	}
}
