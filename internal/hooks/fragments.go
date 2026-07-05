package hooks

import (
	"fmt"
	"strings"
)

var appendArgKeys = map[string]struct{}{
	"head_html":     {},
	"body_html":     {},
	"robots_append": {},
}

// MergeArgs merges hook argument maps. Fragment keys are appended; sitemap_urls are concatenated.
func MergeArgs(base, patch map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range base {
		out[k] = v
	}
	for k, v := range patch {
		if _, ok := appendArgKeys[k]; ok {
			appendStringArg(out, k, v)
			continue
		}
		if k == "sitemap_urls" {
			out[k] = mergeSitemapURLs(out[k], v)
			continue
		}
		out[k] = v
	}
	return out
}

func appendStringArg(args map[string]any, key string, value any) {
	fragment, ok := value.(string)
	if !ok || strings.TrimSpace(fragment) == "" {
		return
	}
	if existing, ok := args[key].(string); ok && strings.TrimSpace(existing) != "" {
		args[key] = existing + "\n" + fragment
		return
	}
	args[key] = fragment
}

func mergeSitemapURLs(existing, patch any) []map[string]any {
	out := append([]map[string]any{}, sitemapURLSlice(existing)...)
	out = append(out, sitemapURLSlice(patch)...)
	return out
}

func sitemapURLSlice(v any) []map[string]any {
	switch rows := v.(type) {
	case []map[string]any:
		return append([]map[string]any(nil), rows...)
	case []any:
		out := make([]map[string]any, 0, len(rows))
		for _, row := range rows {
			if m, ok := row.(map[string]any); ok {
				out = append(out, m)
			}
		}
		return out
	default:
		return nil
	}
}

// HTMLFragment returns a trimmed HTML fragment argument.
func HTMLFragment(args map[string]any, key string) string {
	return strings.TrimSpace(StringArg(args, key))
}

// SitemapURL describes one sitemap entry from hook arguments.
type SitemapURL struct {
	Loc     string
	LastMod string
}

// SitemapURLs parses sitemap_urls hook arguments.
func SitemapURLs(args map[string]any) []SitemapURL {
	raw := sitemapURLSlice(args["sitemap_urls"])
	out := make([]SitemapURL, 0, len(raw))
	for _, row := range raw {
		loc := strings.TrimSpace(fmt.Sprint(row["loc"]))
		if loc == "" {
			continue
		}
		out = append(out, SitemapURL{
			Loc:     loc,
			LastMod: strings.TrimSpace(fmt.Sprint(row["lastmod"])),
		})
	}
	return out
}
