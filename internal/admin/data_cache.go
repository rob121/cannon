package admin

import (
	"context"

	"github.com/rob121/cannon/internal/cache"
	"github.com/rob121/cannon/internal/security"
)

func invalidateRoutesDataCache(ctx context.Context) {
	siteID := cache.SiteIDFromContext(ctx)
	cache.InvalidateRoutes(siteID)
	cache.InvalidateMenus(siteID) // menu hrefs depend on route path/status
	cache.InvalidateSitePublic(siteID)
}

func invalidateBlocksDataCache(ctx context.Context) {
	siteID := cache.SiteIDFromContext(ctx)
	cache.InvalidateBlocks(siteID)
	cache.InvalidateSitePublic(siteID)
}

func invalidateGroupsDataCache(ctx context.Context) {
	siteID := cache.SiteIDFromContext(ctx)
	cache.InvalidateGroups(siteID)
	cache.InvalidateSitePublic(siteID)
}

func invalidateMenusDataCache(ctx context.Context) {
	siteID := cache.SiteIDFromContext(ctx)
	cache.InvalidateMenus(siteID)
	cache.InvalidateSitePublic(siteID)
}

func invalidateSecuritySiteContext(ctx context.Context) {
	security.InvalidateSiteContext(ctx)
}

func invalidateSecurityUser(ctx context.Context, userID uint) {
	security.InvalidateSiteUser(ctx, userID)
}

func invalidateSecurityUserFromRequest(ctx context.Context, idStr string) {
	id, ok := parseID(idStr)
	if !ok {
		return
	}
	invalidateSecurityUser(ctx, id)
}
