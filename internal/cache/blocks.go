package cache

import (
	"context"
	"strings"
	"sync"

	"github.com/rob121/cannon/internal/lrucache"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/sites"
	"gorm.io/gorm"
)

type siteBlocksCache struct {
	mu     sync.Mutex
	spaces *lrucache.LRU[string, []models.Block]
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
	c = &siteBlocksCache{spaces: lrucache.New[string, []models.Block](MaxBlockSpacesPerSite)}
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
	c.mu.Lock()
	if rows, ok := c.spaces.Get(space); ok {
		c.mu.Unlock()
		return cloneBlocks(rows), nil
	}
	c.mu.Unlock()

	rows, err := loadBlocksForSpace(db, space)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	c.spaces.Set(space, cloneBlocks(rows))
	c.mu.Unlock()
	return rows, nil
}

func loadBlocksForSpace(db *gorm.DB, space string) ([]models.Block, error) {
	var rows []models.Block
	err := db.Preload("Groups").
		Joins("JOIN block_spaces ON block_spaces.block_id = blocks.block_id").
		Where("block_spaces.space = ? AND blocks.status = ?", space, models.StatusActive).
		Select("blocks.*, block_spaces.sort AS sort").
		Order("block_spaces.sort ASC, blocks.block_id ASC").
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
	c.spaces.Clear()
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
		if len(row.Spaces) > 0 {
			out[i].Spaces = make([]models.BlockSpace, len(row.Spaces))
			copy(out[i].Spaces, row.Spaces)
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
