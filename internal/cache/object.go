package cache

import (
	"fmt"
	"sync"
	"time"

	"github.com/rob121/cannon/internal/lrucache"
)

// ObjectEntry is a typed blob in the process-local object cache.
type ObjectEntry struct {
	Value   any
	Expires time.Time
}

type siteObjectCache struct {
	objects *lrucache.LRU[string, ObjectEntry]
}

var (
	objectsMu     sync.Mutex
	objectsBySite = map[string]*siteObjectCache{}
)

func objectCacheFor(siteID string) *siteObjectCache {
	objectsMu.Lock()
	defer objectsMu.Unlock()
	c := objectsBySite[siteID]
	if c == nil {
		c = &siteObjectCache{objects: lrucache.New[string, ObjectEntry](MaxObjectsPerSite)}
		objectsBySite[siteID] = c
	}
	return c
}

// ObjectKey builds a stable object-cache key.
func ObjectKey(kind string, id string) string {
	return "obj:" + kind + ":" + id
}

// GetObject returns a non-expired object entry.
func GetObject(siteID, key string) (any, bool) {
	if siteID == "" || key == "" {
		return nil, false
	}
	c := objectCacheFor(siteID)
	objectsMu.Lock()
	defer objectsMu.Unlock()
	ent, ok := c.objects.Get(key)
	if !ok {
		return nil, false
	}
	if !ent.Expires.IsZero() && time.Now().After(ent.Expires) {
		c.objects.Delete(key)
		return nil, false
	}
	return ent.Value, true
}

// SetObject stores a value until expires. Zero expires means no TTL (until site flush).
func SetObject(siteID, key string, value any, expires time.Time) {
	if siteID == "" || key == "" || value == nil {
		return
	}
	c := objectCacheFor(siteID)
	objectsMu.Lock()
	defer objectsMu.Unlock()
	c.objects.Set(key, ObjectEntry{Value: value, Expires: expires})
}

// InvalidateSiteObjects drops all cached objects for a site.
func InvalidateSiteObjects(siteID string) {
	if siteID == "" {
		return
	}
	objectsMu.Lock()
	defer objectsMu.Unlock()
	if c := objectsBySite[siteID]; c != nil {
		c.objects.Clear()
	}
}

// InvalidateAllObjects drops every site's object cache.
func InvalidateAllObjects() {
	objectsMu.Lock()
	defer objectsMu.Unlock()
	objectsBySite = map[string]*siteObjectCache{}
}

// FormatObjectID formats a numeric id for ObjectKey.
func FormatObjectID(id uint) string {
	return fmt.Sprintf("%d", id)
}
