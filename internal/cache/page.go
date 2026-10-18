package cache

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/rob121/cannon/internal/lrucache"
)

const (
	PageCacheHeader     = "X-Cannon-Page-Cache"
	PageCacheHeaderHIT  = "HIT"
	PageCacheHeaderMISS = "MISS"
)

// PageEntry is a cached anonymous HTML (or other) response body.
type PageEntry struct {
	Status      int
	ContentType string
	Body        []byte
	Expires     time.Time
}

type sitePageCache struct {
	pages *lrucache.LRU[string, PageEntry]
}

var (
	pagesMu     sync.Mutex
	pagesBySite = map[string]*sitePageCache{}
)

func pageCacheFor(siteID string) *sitePageCache {
	pagesMu.Lock()
	defer pagesMu.Unlock()
	c := pagesBySite[siteID]
	if c == nil {
		c = &sitePageCache{pages: lrucache.New[string, PageEntry](MaxPagesPerSite)}
		pagesBySite[siteID] = c
	}
	return c
}

// BuildPageKey builds a cache key from content locale, path, and query string.
func BuildPageKey(locale, path, rawQuery string) string {
	locale = strings.TrimSpace(locale)
	if locale == "" {
		locale = "_"
	}
	path = strings.TrimSpace(path)
	if path == "" {
		path = "/"
	}
	if rawQuery == "" {
		return locale + "|" + path
	}
	return locale + "|" + path + "?" + rawQuery
}

// ShouldCacheRequest reports whether the request is eligible for page caching.
func ShouldCacheRequest(r *http.Request, authenticated bool) bool {
	if r == nil || authenticated {
		return false
	}
	if r.Method != http.MethodGet {
		return false
	}
	path := r.URL.Path
	for _, prefix := range []string{"/admin", "/api", "/assets", "/theme", "/install"} {
		if path == prefix || strings.HasPrefix(path, prefix+"/") {
			return false
		}
	}
	if r.URL.Query().Get("tp") == "1" {
		return false
	}
	return true
}

// ShouldCacheResponse reports whether a completed response may be stored.
func ShouldCacheResponse(status int, header http.Header) bool {
	if status != http.StatusOK {
		return false
	}
	if header != nil && header.Get("Set-Cookie") != "" {
		return false
	}
	return true
}

// GetPage returns a non-expired page entry.
func GetPage(siteID, key string) (PageEntry, bool) {
	if siteID == "" || key == "" {
		return PageEntry{}, false
	}
	c := pageCacheFor(siteID)
	pagesMu.Lock()
	defer pagesMu.Unlock()
	ent, ok := c.pages.Get(key)
	if !ok {
		return PageEntry{}, false
	}
	if !ent.Expires.IsZero() && time.Now().After(ent.Expires) {
		c.pages.Delete(key)
		return PageEntry{}, false
	}
	// Copy body so callers cannot mutate the LRU entry.
	out := ent
	if ent.Body != nil {
		out.Body = append([]byte(nil), ent.Body...)
	}
	return out, true
}

// SetPage stores a page entry until expires.
func SetPage(siteID, key string, status int, contentType string, body []byte, expires time.Time) {
	if siteID == "" || key == "" || status != http.StatusOK {
		return
	}
	c := pageCacheFor(siteID)
	pagesMu.Lock()
	defer pagesMu.Unlock()
	stored := PageEntry{
		Status:      status,
		ContentType: contentType,
		Expires:     expires,
	}
	if body != nil {
		stored.Body = append([]byte(nil), body...)
	}
	c.pages.Set(key, stored)
}

// InvalidateSitePages drops all cached pages for a site.
func InvalidateSitePages(siteID string) {
	if siteID == "" {
		return
	}
	pagesMu.Lock()
	defer pagesMu.Unlock()
	if c := pagesBySite[siteID]; c != nil {
		c.pages.Clear()
	}
}

// InvalidateAllPages drops every site's page cache.
func InvalidateAllPages() {
	pagesMu.Lock()
	defer pagesMu.Unlock()
	pagesBySite = map[string]*sitePageCache{}
}

// InvalidateSitePublic flushes page and object caches for a site (content/layout changes).
func InvalidateSitePublic(siteID string) {
	InvalidateSitePages(siteID)
	InvalidateSiteObjects(siteID)
}
