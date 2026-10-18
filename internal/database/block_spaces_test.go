package database_test

import (
	"testing"

	"github.com/rob121/cannon/internal/config"
	"github.com/rob121/cannon/internal/database"
	"github.com/rob121/cannon/internal/models"
)

func TestMigrateBackfillsBlockSpaces(t *testing.T) {
	site := &config.SiteConfig{
		ID: t.Name(),
		Database: config.DatabaseConfig{
			Type: "sqlite",
			DSN:  t.TempDir() + "/site.db",
		},
	}
	if err := database.Migrate(site); err != nil {
		t.Fatal(err)
	}
	db, err := database.Get(site.ID)
	if err != nil {
		t.Fatal(err)
	}

	row := models.Block{
		Name:   "FromSpaceCol",
		Type:   models.BlockTypeHTML,
		Space:  "header",
		Status: models.StatusActive,
		Sort:   2,
	}
	if err := db.Omit("Spaces").Create(&row).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Where("block_id = ?", row.BlockID).Delete(&models.BlockSpace{}).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Migrate(site); err != nil {
		t.Fatal(err)
	}
	var count int64
	if err := db.Model(&models.BlockSpace{}).Where("block_id = ? AND space = ? AND sort = ?", row.BlockID, "header", 2).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("expected backfilled membership, got count=%d", count)
	}
}
