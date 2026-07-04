package security

import (
	"sync"
)

type siteCache struct {
	mu    sync.RWMutex
	users map[uint]EffectivePermissions
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
	c = &siteCache{users: map[uint]EffectivePermissions{}}
	siteCaches[siteID] = c
	return c
}

func cacheGet(siteID string, userID uint) (EffectivePermissions, bool) {
	c := siteCacheFor(siteID)
	c.mu.RLock()
	defer c.mu.RUnlock()
	perms, ok := c.users[userID]
	if !ok {
		return EffectivePermissions{}, false
	}
	return cloneEffective(perms), true
}

func cacheSet(siteID string, userID uint, perms EffectivePermissions) {
	c := siteCacheFor(siteID)
	c.mu.Lock()
	defer c.mu.Unlock()
	c.users[userID] = cloneEffective(perms)
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
	delete(c.users, userID)
}

// InvalidateSite drops all cached permissions for a site.
func InvalidateSite(siteID string) {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	delete(siteCaches, siteID)
}

// InvalidateAll drops every cached permission set.
func InvalidateAll() {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	siteCaches = map[string]*siteCache{}
}
