package database

import (
	"strings"

	"github.com/rob121/cannon/internal/models"
	"gorm.io/gorm"
)

// RepairBlockSpacesSchema backfills block_spaces from the legacy blocks.space column.
func RepairBlockSpacesSchema(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	if err := db.AutoMigrate(&models.BlockSpace{}); err != nil {
		return err
	}
	var rows []models.Block
	if err := db.Select("block_id", "space", "sort").Find(&rows).Error; err != nil {
		return err
	}
	for _, row := range rows {
		space := strings.TrimSpace(row.Space)
		if space == "" {
			continue
		}
		var count int64
		if err := db.Model(&models.BlockSpace{}).
			Where("block_id = ? AND space = ?", row.BlockID, space).
			Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			continue
		}
		if err := db.Create(&models.BlockSpace{
			BlockID: row.BlockID,
			Space:   space,
			Sort:    row.Sort,
		}).Error; err != nil {
			return err
		}
	}
	return nil
}
