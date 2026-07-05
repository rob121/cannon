package cache

import (
	"context"
	"fmt"
	"sync"
)

type siteSettingsCache struct {
	mu       sync.RWMutex
	sections map[string]map[string]map[string]any // scope -> section -> data
}

var (
	settingsMu     sync.RWMutex
	settingsBySite = map[string]*siteSettingsCache{}
)

func settingsCacheFor(siteID string) *siteSettingsCache {
	settingsMu.RLock()
	c, ok := settingsBySite[siteID]
	settingsMu.RUnlock()
	if ok {
		return c
	}
	settingsMu.Lock()
	defer settingsMu.Unlock()
	if c, ok = settingsBySite[siteID]; ok {
		return c
	}
	c = &siteSettingsCache{sections: map[string]map[string]map[string]any{}}
	settingsBySite[siteID] = c
	return c
}

// SettingsKey identifies one settings section cache entry.
type SettingsKey struct {
	SiteID  string
	Scope   string
	Section string
}

// GetSettings returns a cached settings section when caching is enabled.
func GetSettings(ctx context.Context, scope, section string) (map[string]any, bool) {
	if !Enabled(ctx) {
		return nil, false
	}
	siteID := SiteIDFromContext(ctx)
	if siteID == "" {
		return nil, false
	}
	c := settingsCacheFor(siteID)
	c.mu.RLock()
	defer c.mu.RUnlock()
	scopeMap, ok := c.sections[scope]
	if !ok {
		return nil, false
	}
	data, ok := scopeMap[section]
	if !ok {
		return nil, false
	}
	return cloneMap(data), true
}

// SetSettings stores a settings section in the site cache.
func SetSettings(ctx context.Context, scope, section string, data map[string]any) {
	if !Enabled(ctx) {
		return
	}
	siteID := SiteIDFromContext(ctx)
	if siteID == "" {
		return
	}
	c := settingsCacheFor(siteID)
	c.mu.Lock()
	defer c.mu.Unlock()
	scopeMap, ok := c.sections[scope]
	if !ok {
		scopeMap = map[string]map[string]any{}
		c.sections[scope] = scopeMap
	}
	scopeMap[section] = cloneMap(data)
}

// InvalidateSettings drops one or all settings sections for a site.
func InvalidateSettings(siteID, scope, section string) {
	if siteID == "" {
		return
	}
	c := settingsCacheFor(siteID)
	c.mu.Lock()
	defer c.mu.Unlock()
	if scope == "" {
		c.sections = map[string]map[string]map[string]any{}
		return
	}
	scopeMap, ok := c.sections[scope]
	if !ok {
		return
	}
	if section == "" {
		delete(c.sections, scope)
		return
	}
	delete(scopeMap, section)
}

func invalidateSettings(siteID string) {
	InvalidateSettings(siteID, "", "")
}

// CloneMap returns a shallow copy of a settings map.
func CloneMap(data map[string]any) map[string]any {
	return cloneMap(data)
}

func cloneMap(data map[string]any) map[string]any {
	if data == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(data))
	for k, v := range data {
		out[k] = v
	}
	return out
}

// SettingsCacheKey builds a stable cache key string for diagnostics.
func SettingsCacheKey(scope, section string) string {
	return fmt.Sprintf("%s:%s", scope, section)
}
