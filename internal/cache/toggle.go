package cache

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/sites"
	"gorm.io/gorm"
)

const (
	globalSettingsScope   = "global"
	generalSettingsSection = "general"
)

const toggleKey = "memory_cache_enabled"

var (
	toggleMu    sync.RWMutex
	toggles     = map[string]bool{}
	toggleKnown = map[string]bool{}
)

// Enabled reports whether domain-level in-memory caches are active for the site.
func Enabled(ctx context.Context) bool {
	siteID := SiteIDFromContext(ctx)
	if siteID == "" {
		return false
	}
	toggleMu.RLock()
	_, ok := toggleKnown[siteID]
	enabled := toggles[siteID]
	toggleMu.RUnlock()
	if ok {
		return enabled
	}
	enabled = loadToggleFromDB(ctx)
	SetToggle(siteID, enabled)
	return enabled
}

// SetToggle records the enable flag without a database read.
func SetToggle(siteID string, enabled bool) {
	if siteID == "" {
		return
	}
	toggleMu.Lock()
	toggles[siteID] = enabled
	toggleKnown[siteID] = true
	toggleMu.Unlock()
}

func invalidateToggle(siteID string) {
	toggleMu.Lock()
	delete(toggles, siteID)
	delete(toggleKnown, siteID)
	toggleMu.Unlock()
}

func loadToggleFromDB(ctx context.Context) bool {
	db, err := sites.DB(ctx)
	if err != nil {
		return true
	}
	var row models.Setting
	err = db.Where("scope = ? AND section = ?", globalSettingsScope, generalSettingsSection).First(&row).Error
	if err == gorm.ErrRecordNotFound || row.Data == "" {
		return true
	}
	if err != nil {
		return true
	}
	var data map[string]any
	if err := json.Unmarshal([]byte(row.Data), &data); err != nil {
		return true
	}
	v, ok := data[toggleKey]
	if !ok || v == nil {
		return true
	}
	return parseBoolValue(v, true)
}

func parseBoolValue(v any, fallback bool) bool {
	switch b := v.(type) {
	case bool:
		return b
	case float64:
		return b != 0
	case string:
		return b == "true" || b == "1"
	default:
		return fallback
	}
}
