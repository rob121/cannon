package security

import (
	"sync"

	"github.com/rob121/cannon/internal/cache"
	"github.com/rob121/cannon/internal/lrucache"
)

// MaxCachedUsersPerSite caps permission sets retained per site (small-VPS bound).
var MaxCachedUsersPerSite = 512

type siteCache struct {
	mu    sync.Mutex
	users *lrucache.LRU[uint, EffectivePermissions]
}

var cacheMu sync.RWMutex
var siteCaches = map[string]*siteCache{}

func siteCacheFor(siteID string) *siteCache {
	cacheMu.RLock()
	c, ok := siteCaches[siteID]
	cacheMu.RUnlock()
	if ok {
		return c
	}
	cacheMu.Lock()
	defer cacheMu.Unlock()
	if c, ok = siteCaches[siteID]; ok {
		return c
	}
	c = &siteCache{users: lrucache.New[uint, EffectivePermissions](MaxCachedUsersPerSite)}
	siteCaches[siteID] = c
	return c
}

func cacheGet(siteID string, userID uint) (EffectivePermissions, bool) {
	c := siteCacheFor(siteID)
	c.mu.Lock()
	defer c.mu.Unlock()
	perms, ok := c.users.Get(userID)
	if !ok {
		return EffectivePermissions{}, false
	}
	return cloneEffective(perms), true
}

func cacheSet(siteID string, userID uint, perms EffectivePermissions) {
	c := siteCacheFor(siteID)
	c.mu.Lock()
	defer c.mu.Unlock()
	c.users.Set(userID, cloneEffective(perms))
}

func cloneEffective(perms EffectivePermissions) EffectivePermissions {
	out := NewEffectivePermissions()
	for k := range perms.Allow {
		out.Allow[k] = struct{}{}
	}
	for k := range perms.Deny {
		out.Deny[k] = struct{}{}
	}
	return out
}

// InvalidateUser drops cached permissions for one user on a site.
func InvalidateUser(siteID string, userID uint) {
	c := siteCacheFor(siteID)
	c.mu.Lock()
	defer c.mu.Unlock()
	c.users.Delete(userID)
	cache.InvalidateViewerGroups(siteID, userID)
}

// InvalidateSite drops all cached permissions for a site.
func InvalidateSite(siteID string) {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	delete(siteCaches, siteID)
	cache.InvalidateGroups(siteID)
}

// InvalidateAll drops every cached permission set and domain group lookups.
func InvalidateAll() {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	siteCaches = map[string]*siteCache{}
	cache.InvalidateAll()
}
