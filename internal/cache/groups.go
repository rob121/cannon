package cache

import (
	"context"
	"sync"

	"github.com/rob121/cannon/internal/models"
	"gorm.io/gorm"
)

type siteGroupsCache struct {
	mu            sync.RWMutex
	publicGroupID uint
	publicKnown   bool
	viewerGroups  map[uint][]uint
}

var (
	groupsMu     sync.RWMutex
	groupsBySite = map[string]*siteGroupsCache{}
)

func groupsCacheFor(siteID string) *siteGroupsCache {
	groupsMu.RLock()
	c, ok := groupsBySite[siteID]
	groupsMu.RUnlock()
	if ok {
		return c
	}
	groupsMu.Lock()
	defer groupsMu.Unlock()
	if c, ok = groupsBySite[siteID]; ok {
		return c
	}
	c = &siteGroupsCache{viewerGroups: map[uint][]uint{}}
	groupsBySite[siteID] = c
	return c
}

// PublicGroupID returns the public group id, using the site cache when enabled.
func PublicGroupID(ctx context.Context, db *gorm.DB) (uint, error) {
	siteID := SiteIDFromContext(ctx)
	if siteID != "" && Enabled(ctx) {
		c := groupsCacheFor(siteID)
		c.mu.RLock()
		if c.publicKnown {
			id := c.publicGroupID
			c.mu.RUnlock()
			return id, nil
		}
		c.mu.RUnlock()
	}

	id, err := loadPublicGroupID(db)
	if err != nil {
		return 0, err
	}
	if siteID != "" && Enabled(ctx) {
		c := groupsCacheFor(siteID)
		c.mu.Lock()
		c.publicGroupID = id
		c.publicKnown = true
		c.mu.Unlock()
	}
	return id, nil
}

// ViewerGroupIDs returns group ids for a viewer, using the site cache when enabled.
func ViewerGroupIDs(ctx context.Context, db *gorm.DB, userID uint, load func(*gorm.DB, uint) ([]uint, error)) ([]uint, error) {
	publicID, err := PublicGroupID(ctx, db)
	if err != nil {
		return nil, err
	}
	if userID == 0 {
		return []uint{publicID}, nil
	}

	siteID := SiteIDFromContext(ctx)
	if siteID != "" && Enabled(ctx) {
		c := groupsCacheFor(siteID)
		c.mu.RLock()
		if ids, ok := c.viewerGroups[userID]; ok {
			c.mu.RUnlock()
			return cloneUints(ids), nil
		}
		c.mu.RUnlock()
	}

	ids, err := load(db, userID)
	if err != nil {
		return nil, err
	}
	ids = ensureContains(ids, publicID)

	if siteID != "" && Enabled(ctx) {
		c := groupsCacheFor(siteID)
		c.mu.Lock()
		c.viewerGroups[userID] = cloneUints(ids)
		c.mu.Unlock()
	}
	return cloneUints(ids), nil
}

// InvalidateGroups drops cached group lookups for a site.
func InvalidateGroups(siteID string) {
	if siteID == "" {
		return
	}
	c := groupsCacheFor(siteID)
	c.mu.Lock()
	c.publicKnown = false
	c.publicGroupID = 0
	c.viewerGroups = map[uint][]uint{}
	c.mu.Unlock()
}

// InvalidateViewerGroups drops cached viewer group ids for one user.
func InvalidateViewerGroups(siteID string, userID uint) {
	if siteID == "" {
		return
	}
	c := groupsCacheFor(siteID)
	c.mu.Lock()
	delete(c.viewerGroups, userID)
	c.mu.Unlock()
}

func invalidateGroups(siteID string) {
	InvalidateGroups(siteID)
}

func loadPublicGroupID(db *gorm.DB) (uint, error) {
	var group models.Group
	if err := db.Where("name = ?", "public").First(&group).Error; err != nil {
		return 0, err
	}
	return group.GroupID, nil
}

func ensureContains(ids []uint, id uint) []uint {
	for _, v := range ids {
		if v == id {
			return ids
		}
	}
	out := make([]uint, len(ids)+1)
	copy(out, ids)
	out[len(ids)] = id
	return out
}

func cloneUints(ids []uint) []uint {
	if len(ids) == 0 {
		return nil
	}
	out := make([]uint, len(ids))
	copy(out, ids)
	return out
}
