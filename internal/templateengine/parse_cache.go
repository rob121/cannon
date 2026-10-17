package templateengine

import (
	"context"
	"html/template"
	"sync"

	"github.com/rob121/cannon/internal/settings"
)

func init() {
	settings.SetOnGeneralSaved(func(_ context.Context, siteID string, _ map[string]any) {
		InvalidateSite(siteID)
	})
}

// Caching strategy:
//   - Frontend: cache source bundles (main + partial HTML) to skip disk/embed reads.
//     Each Engine still Parses with its own FuncMap.
//   - Admin: cache a parsed template set (embed is already in memory; Parse is the cost),
//     Clone + Funcs per Engine. Admin FuncMaps are always complete in production.

type partialSource struct {
	name    string
	content string
}

type sourceBundle struct {
	main     string
	mainSrc  string
	partials []partialSource
}

var (
	parseCacheMu    sync.RWMutex
	frontendSources = map[string]*sourceBundle{}
	adminParsedSets = map[string]*template.Template{}
	parseCacheReads int
)

func frontendCacheKey(siteID, theme, name string) string {
	return siteID + "\x00" + theme + "\x00" + name
}

func adminCacheKey(siteID, theme string) string {
	return siteID + "\x00" + theme + "\x00admin"
}

// InvalidateSite drops cached template sources/sets for one site.
func InvalidateSite(siteID string) {
	if siteID == "" {
		return
	}
	prefix := siteID + "\x00"
	parseCacheMu.Lock()
	defer parseCacheMu.Unlock()
	for k := range frontendSources {
		if len(k) >= len(prefix) && k[:len(prefix)] == prefix {
			delete(frontendSources, k)
		}
	}
	for k := range adminParsedSets {
		if len(k) >= len(prefix) && k[:len(prefix)] == prefix {
			delete(adminParsedSets, k)
		}
	}
}

// InvalidateAll drops every cached template source/set.
func InvalidateAll() {
	parseCacheMu.Lock()
	frontendSources = map[string]*sourceBundle{}
	adminParsedSets = map[string]*template.Template{}
	parseCacheMu.Unlock()
}

func getFrontendSources(key string) *sourceBundle {
	parseCacheMu.RLock()
	defer parseCacheMu.RUnlock()
	return frontendSources[key]
}

func setFrontendSources(key string, bundle *sourceBundle) {
	parseCacheMu.Lock()
	frontendSources[key] = bundle
	parseCacheMu.Unlock()
}

func getAdminParsedSet(key string) *template.Template {
	parseCacheMu.RLock()
	defer parseCacheMu.RUnlock()
	return adminParsedSets[key]
}

func setAdminParsedSet(key string, set *template.Template) {
	parseCacheMu.Lock()
	adminParsedSets[key] = set
	parseCacheMu.Unlock()
}
