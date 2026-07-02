package server

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/rob121/cannon/internal/config"
	"github.com/rob121/cannon/internal/content"
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
	items, _, err := content.ListItems(ctx, nil, content.ListOptions{Page: 1, Limit: 5000})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	categories, _ := content.CategoryTree(ctx)
	tags, _ := content.ListTags(ctx)

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	fmt.Fprint(w, `<?xml version="1.0" encoding="UTF-8"?>`)
	fmt.Fprint(w, `<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`)
	writeSitemapURL(w, base+"/", time.Now())
	for _, item := range items {
		if item.Status != models.ItemStatusPublished {
			continue
		}
		lastMod := item.UpdatedAt
		if lastMod.IsZero() {
			lastMod = item.CreatedAt
		}
		writeSitemapURL(w, base+content.ItemURL(item.Slug), lastMod)
	}
	for _, cat := range categories {
		writeSitemapURL(w, base+content.CategoryURL(cat.Slug), cat.UpdatedAt)
	}
	for _, tag := range tags {
		writeSitemapURL(w, base+content.TagURL(tag.Slug), tag.UpdatedAt)
	}
	fmt.Fprint(w, `</urlset>`)
}

func writeSitemapURL(w http.ResponseWriter, loc string, lastMod time.Time) {
	fmt.Fprint(w, `<url><loc>`)
	fmt.Fprint(w, xmlEscape(loc))
	fmt.Fprint(w, `</loc>`)
	if !lastMod.IsZero() {
		fmt.Fprint(w, `<lastmod>`)
		fmt.Fprint(w, lastMod.UTC().Format("2006-01-02"))
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
