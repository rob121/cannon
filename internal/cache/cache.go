package cache

import (
	"context"

	"github.com/rob121/cannon/internal/sites"
)

// InvalidateSite drops all in-memory domain caches for one site.
// This clears settings, routes, blocks, and group lookups — not permission caches
// (see security.InvalidateSite).
func InvalidateSite(siteID string) {
	if siteID == "" {
		return
	}
	invalidateSettings(siteID)
	invalidateRoutes(siteID)
	invalidateGroups(siteID)
	invalidateBlocks(siteID)
	invalidateToggle(siteID)
}

// InvalidateAll drops every site's in-memory domain caches.
func InvalidateAll() {
	settingsMu.Lock()
	settingsBySite = map[string]*siteSettingsCache{}
	settingsMu.Unlock()

	routesMu.Lock()
	routesBySite = map[string][]cachedRoute{}
	routesMu.Unlock()

	groupsMu.Lock()
	groupsBySite = map[string]*siteGroupsCache{}
	groupsMu.Unlock()

	blocksMu.Lock()
	blocksBySite = map[string]*siteBlocksCache{}
	blocksMu.Unlock()

	toggleMu.Lock()
	toggles = map[string]bool{}
	toggleKnown = map[string]bool{}
	toggleMu.Unlock()
}

// SiteIDFromContext returns the current site id or "" when unavailable.
func SiteIDFromContext(ctx context.Context) string {
	site, err := sites.FromContext(ctx)
	if err != nil || site == nil {
		return ""
	}
	return site.ID
}
