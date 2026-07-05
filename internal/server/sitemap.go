package server

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/rob121/cannon/internal/config"
	"github.com/rob121/cannon/internal/content"
	"github.com/rob121/cannon/internal/hooks"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/sites"
)

func (s *Server) serveSitemapXML(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	site, err := sites.FromContext(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	base := strings.TrimRight(siteBaseURL(r, site), "/")
	items, _, err := content.ListItems(ctx, nil, content.ListOptions{Page: 1, Limit: 5000, AllLocales: true})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	categories, _ := content.CategoryTreeAll(ctx)
	tags, _ := content.ListTags(ctx)

	entries := []hooks.SitemapURL{{Loc: base + "/", LastMod: time.Now().UTC().Format("2006-01-02")}}
	for _, item := range items {
		if item.Status != models.ItemStatusPublished {
			continue
		}
		lastMod := item.UpdatedAt
		if lastMod.IsZero() {
			lastMod = item.CreatedAt
		}
		entries = append(entries, hooks.SitemapURL{
			Loc:     base + content.ItemURLForContext(content.WithLocale(ctx, item.Locale), item.Slug),
			LastMod: lastMod.UTC().Format("2006-01-02"),
		})
	}
	for _, cat := range categories {
		entries = append(entries, hooks.SitemapURL{
			Loc:     base + content.CategoryURLForContext(content.WithLocale(ctx, cat.Locale), cat.Slug),
			LastMod: cat.UpdatedAt.UTC().Format("2006-01-02"),
		})
	}
	for _, tag := range tags {
		entries = append(entries, hooks.SitemapURL{
			Loc:     base + content.TagURL(tag.Slug),
			LastMod: tag.UpdatedAt.UTC().Format("2006-01-02"),
		})
	}

	hookArgs := map[string]any{
		"base_url": base,
		"sitemap_urls": sitemapEntriesToMaps(entries),
	}
	if out, err := hooks.Fire(ctx, hooks.OnSitemapGenerate, hookArgs); err == nil {
		entries = append(entries, hooks.SitemapURLs(out)...)
	}

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	fmt.Fprint(w, `<?xml version="1.0" encoding="UTF-8"?>`)
	fmt.Fprint(w, `<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`)
	for _, entry := range entries {
		writeSitemapURL(w, entry.Loc, entry.LastMod)
	}
	fmt.Fprint(w, `</urlset>`)
}

func sitemapEntriesToMaps(entries []hooks.SitemapURL) []map[string]any {
	out := make([]map[string]any, 0, len(entries))
	for _, entry := range entries {
		row := map[string]any{"loc": entry.Loc}
		if entry.LastMod != "" {
			row["lastmod"] = entry.LastMod
		}
		out = append(out, row)
	}
	return out
}

func writeSitemapURL(w http.ResponseWriter, loc, lastMod string) {
	fmt.Fprint(w, `<url><loc>`)
	fmt.Fprint(w, xmlEscape(loc))
	fmt.Fprint(w, `</loc>`)
	if strings.TrimSpace(lastMod) != "" {
		fmt.Fprint(w, `<lastmod>`)
		fmt.Fprint(w, xmlEscape(lastMod))
		fmt.Fprint(w, `</lastmod>`)
	}
	fmt.Fprint(w, `</url>`)
}

func xmlEscape(s string) string {
	replacer := strings.NewReplacer(
		`&`, "&amp;",
		`<`, "&lt;",
		`>`, "&gt;",
		`"`, "&quot;",
		`'`, "&apos;",
	)
	return replacer.Replace(s)
}

func siteBaseURL(r *http.Request, site *config.SiteConfig) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if proto := strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")); proto != "" {
		scheme = strings.ToLower(strings.Split(proto, ",")[0])
	}
	host := r.Host
	if host == "" && site != nil && site.Host != "" {
		host = site.Host
	}
	if host == "" {
		host = "localhost"
	}
	return scheme + "://" + strings.TrimRight(host, "/")
}
