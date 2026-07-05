package settings

import (
	"context"

	"github.com/rob121/cannon/internal/cache"
)

type cachedStore struct {
	inner Store
}

// ToggleKey is the general settings field that enables domain-level memory caches.
func ToggleKey() string {
	return "memory_cache_enabled"
}

// NewStore returns the default database-backed settings store with optional in-memory caching.
func NewStore() Store {
	return cachedStore{inner: dbStore{}}
}

func (c cachedStore) Load(ctx context.Context, scope, section string) (map[string]any, error) {
	if data, ok := cache.GetSettings(ctx, scope, section); ok {
		return data, nil
	}
	data, err := c.inner.Load(ctx, scope, section)
	if err != nil {
		return nil, err
	}
	cache.SetSettings(ctx, scope, section, data)
	return cache.CloneMap(data), nil
}

func (c cachedStore) Save(ctx context.Context, scope, section string, data map[string]any) error {
	if err := c.inner.Save(ctx, scope, section, data); err != nil {
		return err
	}
	siteID := cache.SiteIDFromContext(ctx)
	cache.InvalidateSettings(siteID, scope, section)
	if scope == ScopeGlobal && section == SectionGeneral {
		enabled := Bool(data, ToggleKey())
		if !enabled {
			cache.InvalidateSite(siteID)
		}
		cache.SetToggle(siteID, enabled)
	}
	return nil
}
