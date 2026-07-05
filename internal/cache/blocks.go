package cache

import (
	"context"
	"strings"
	"sync"

	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/sites"
	"gorm.io/gorm"
)

type siteBlocksCache struct {
	mu     sync.RWMutex
	spaces map[string][]models.Block
}

var (
	blocksMu     sync.RWMutex
	blocksBySite = map[string]*siteBlocksCache{}
)

func blocksCacheFor(siteID string) *siteBlocksCache {
	blocksMu.RLock()
	c, ok := blocksBySite[siteID]
	blocksMu.RUnlock()
	if ok {
		return c
	}
	blocksMu.Lock()
	defer blocksMu.Unlock()
	if c, ok = blocksBySite[siteID]; ok {
		return c
	}
	c = &siteBlocksCache{spaces: map[string][]models.Block{}}
	blocksBySite[siteID] = c
	return c
}

// BlocksForSpace returns active blocks for a template space before viewer filtering.
func BlocksForSpace(ctx context.Context, db *gorm.DB, space string) ([]models.Block, error) {
	space = strings.TrimSpace(space)
	if space == "" {
		return nil, nil
	}

	siteID := SiteIDFromContext(ctx)
	if siteID == "" || !Enabled(ctx) {
		return loadBlocksForSpace(db, space)
	}

	c := blocksCacheFor(siteID)
	c.mu.RLock()
	rows, ok := c.spaces[space]
	c.mu.RUnlock()
	if ok {
		return cloneBlocks(rows), nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if rows, ok = c.spaces[space]; ok {
		return cloneBlocks(rows), nil
	}

	rows, err := loadBlocksForSpace(db, space)
	if err != nil {
		return nil, err
	}
	c.spaces[space] = cloneBlocks(rows)
	return rows, nil
}

func loadBlocksForSpace(db *gorm.DB, space string) ([]models.Block, error) {
	var rows []models.Block
	err := db.Preload("Groups").Where("space = ? AND status = ?", space, models.StatusActive).
		Order("sort asc, block_id asc").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	return cloneBlocks(rows), nil
}

// InvalidateBlocks drops cached blocks for a site.
func InvalidateBlocks(siteID string) {
	if siteID == "" {
		return
	}
	c := blocksCacheFor(siteID)
	c.mu.Lock()
	c.spaces = map[string][]models.Block{}
	c.mu.Unlock()
}

func invalidateBlocks(siteID string) {
	InvalidateBlocks(siteID)
}

func cloneBlocks(rows []models.Block) []models.Block {
	if len(rows) == 0 {
		return nil
	}
	out := make([]models.Block, len(rows))
	for i, row := range rows {
		out[i] = row
		if len(row.Groups) > 0 {
			out[i].Groups = make([]models.Group, len(row.Groups))
			copy(out[i].Groups, row.Groups)
		}
	}
	return out
}

// WarmBlocksForSpace preloads one space into the cache.
func WarmBlocksForSpace(ctx context.Context, space string) error {
	db, err := sites.DB(ctx)
	if err != nil {
		return err
	}
	_, err = BlocksForSpace(ctx, db, space)
	return err
}
