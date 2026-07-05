package admin

import (
	"context"

	"github.com/rob121/cannon/internal/cache"
	"github.com/rob121/cannon/internal/security"
)

func invalidateRoutesDataCache(ctx context.Context) {
	cache.InvalidateRoutes(cache.SiteIDFromContext(ctx))
}

func invalidateBlocksDataCache(ctx context.Context) {
	cache.InvalidateBlocks(cache.SiteIDFromContext(ctx))
}

func invalidateGroupsDataCache(ctx context.Context) {
	cache.InvalidateGroups(cache.SiteIDFromContext(ctx))
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
