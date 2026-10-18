package blocks

import (
	"context"

	"github.com/rob121/cannon/internal/cache"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/sites"
	"gorm.io/gorm"
)

type dashboardSeed struct {
	Name  string
	Type  models.BlockType
	Space string
	Sort  int
}

var adminDashboardSeeds = []dashboardSeed{
	{Name: "Live Analytics", Type: models.BlockTypeAdminLiveAnalytics, Space: "admin-dashboard-top", Sort: 10},
	{Name: "Review Queue", Type: models.BlockTypeAdminReviewQueue, Space: "admin-dashboard-widgets", Sort: 10},
	{Name: "Update Check", Type: models.BlockTypeAdminUpdateCheck, Space: "admin-dashboard-widgets", Sort: 20},
	{Name: "Latest Signups", Type: models.BlockTypeAdminLatestSignups, Space: "admin-dashboard-widgets", Sort: 30},
	{Name: "Quick Start", Type: models.BlockTypeAdminQuickStart, Space: "admin-dashboard-main", Sort: 10},
	{Name: "System Reload", Type: models.BlockTypeAdminSystemReload, Space: "admin-dashboard-main", Sort: 20},
}

// EnsureAdminDashboardBlocks creates missing built-in admin dashboard widgets.
// Existing rows of the same type are left alone (status, sort, and spaces stay user-managed).
func EnsureAdminDashboardBlocks(ctx context.Context) error {
	db, err := sites.DB(ctx)
	if err != nil {
		return err
	}
	created := false
	for _, seed := range adminDashboardSeeds {
		var count int64
		if err := db.Model(&models.Block{}).Where("type = ?", seed.Type).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			continue
		}
		if err := createDashboardSeed(db, seed); err != nil {
			return err
		}
		created = true
	}
	if created {
		if site, err := sites.FromContext(ctx); err == nil && site != nil {
			cache.InvalidateBlocks(site.ID)
		}
	}
	return nil
}

func createDashboardSeed(db *gorm.DB, seed dashboardSeed) error {
	row := models.Block{
		Name:     seed.Name,
		Type:     seed.Type,
		Space:    seed.Space,
		Status:   models.StatusActive,
		Sort:     seed.Sort,
		Metadata: AdminDashboardPlainMeta(),
	}
	if err := db.Create(&row).Error; err != nil {
		return err
	}
	if err := db.Create(&models.BlockSpace{
		BlockID: row.BlockID,
		Space:   seed.Space,
		Sort:    seed.Sort,
	}).Error; err != nil {
		return err
	}
	return attachDashboardSeedGroups(db, &row)
}

func attachDashboardSeedGroups(db *gorm.DB, row *models.Block) error {
	var access []models.Group
	if err := db.Where("name IN ? AND status = ?", []string{"Administrators", "public"}, models.StatusActive).
		Find(&access).Error; err != nil {
		return err
	}
	if len(access) == 0 {
		return nil
	}
	return db.Model(row).Association("Groups").Replace(access)
}
