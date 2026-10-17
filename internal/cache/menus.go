package cache

import (
	"context"
	"strings"
	"sync"

	"github.com/rob121/cannon/internal/lrucache"
	"github.com/rob121/cannon/internal/models"
	"gorm.io/gorm"
)

type siteMenusCache struct {
	mu    sync.Mutex
	menus *lrucache.LRU[string, []models.MenuItem]
}

var (
	menusMu     sync.RWMutex
	menusBySite = map[string]*siteMenusCache{}
)

func menusCacheFor(siteID string) *siteMenusCache {
	menusMu.RLock()
	c, ok := menusBySite[siteID]
	menusMu.RUnlock()
	if ok {
		return c
	}
	menusMu.Lock()
	defer menusMu.Unlock()
	if c, ok = menusBySite[siteID]; ok {
		return c
	}
	c = &siteMenusCache{menus: lrucache.New[string, []models.MenuItem](MaxMenusPerSite)}
	menusBySite[siteID] = c
	return c
}

// ActiveMenuItems returns all items for an active menu (pre-viewer filter), with Groups and Route preloaded.
func ActiveMenuItems(ctx context.Context, db *gorm.DB, menuName string) ([]models.MenuItem, error) {
	menuName = strings.TrimSpace(menuName)
	if menuName == "" {
		return nil, nil
	}
	key := strings.ToLower(menuName)

	siteID := SiteIDFromContext(ctx)
	if siteID == "" || !Enabled(ctx) {
		return loadActiveMenuItems(db, menuName)
	}

	c := menusCacheFor(siteID)
	c.mu.Lock()
	if rows, ok := c.menus.Get(key); ok {
		c.mu.Unlock()
		return cloneMenuItems(rows), nil
	}
	c.mu.Unlock()

	rows, err := loadActiveMenuItems(db, menuName)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	c.menus.Set(key, cloneMenuItems(rows))
	c.mu.Unlock()
	return rows, nil
}

func loadActiveMenuItems(db *gorm.DB, menuName string) ([]models.MenuItem, error) {
	var menu models.Menu
	err := db.Preload("Items", func(db *gorm.DB) *gorm.DB {
		return db.Order("sort asc, menu_item_id asc")
	}).Preload("Items.Groups").Preload("Items.Route").
		Where("LOWER(menu_name) = LOWER(?) AND status = ?", menuName, models.StatusActive).
		First(&menu).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return []models.MenuItem{}, nil
		}
		return nil, err
	}
	return cloneMenuItems(menu.Items), nil
}

// InvalidateMenus drops cached menus for a site.
func InvalidateMenus(siteID string) {
	if siteID == "" {
		return
	}
	c := menusCacheFor(siteID)
	c.mu.Lock()
	c.menus.Clear()
	c.mu.Unlock()
}

func invalidateMenus(siteID string) {
	InvalidateMenus(siteID)
}

func cloneMenuItems(rows []models.MenuItem) []models.MenuItem {
	if len(rows) == 0 {
		return nil
	}
	out := make([]models.MenuItem, len(rows))
	for i, row := range rows {
		out[i] = row
		if len(row.Groups) > 0 {
			out[i].Groups = make([]models.Group, len(row.Groups))
			copy(out[i].Groups, row.Groups)
		}
		if row.Route != nil {
			routeCopy := *row.Route
			out[i].Route = &routeCopy
		}
	}
	return out
}
