package admin

import (
	"testing"

	"github.com/rob121/cannon/internal/blocks"
	"github.com/rob121/cannon/internal/config"
	"github.com/rob121/cannon/internal/database"
	"github.com/rob121/cannon/internal/models"
)

func TestBlockReorderUpdatesSpaceAndLegacySort(t *testing.T) {
	site := &config.SiteConfig{
		ID:       "block-reorder-test",
		Name:     "Reorder",
		Database: config.DatabaseConfig{Type: "sqlite", DSN: t.TempDir() + "/site.db"},
	}
	if err := database.Migrate(site); err != nil {
		t.Fatal(err)
	}
	db, err := database.Get(site.ID)
	if err != nil {
		t.Fatal(err)
	}
	a := models.Block{Name: "A", Type: models.BlockTypeHTML, Space: "sidebar", Status: models.StatusActive, Sort: 0, Metadata: "{}"}
	b := models.Block{Name: "B", Type: models.BlockTypeHTML, Space: "sidebar", Status: models.StatusActive, Sort: 1, Metadata: "{}"}
	if err := db.Create(&a).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&b).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.BlockSpace{BlockID: a.BlockID, Space: "sidebar", Sort: 0}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.BlockSpace{BlockID: b.BlockID, Space: "sidebar", Sort: 1}).Error; err != nil {
		t.Fatal(err)
	}

	if err := blockReorder(db, a.BlockID, "sidebar", 1); err != nil {
		t.Fatal(err)
	}

	var spaces []models.BlockSpace
	if err := db.Where("space = ?", "sidebar").Order("sort ASC").Find(&spaces).Error; err != nil {
		t.Fatal(err)
	}
	if len(spaces) != 2 || spaces[0].BlockID != b.BlockID || spaces[1].BlockID != a.BlockID {
		t.Fatalf("space order = %+v", spaces)
	}
	var gotA, gotB models.Block
	if err := db.First(&gotA, a.BlockID).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.First(&gotB, b.BlockID).Error; err != nil {
		t.Fatal(err)
	}
	if gotB.Sort != 0 || gotA.Sort != 1 {
		t.Fatalf("legacy sort A=%d B=%d", gotA.Sort, gotB.Sort)
	}
}

func TestSortBlocksByPrimarySpace(t *testing.T) {
	rows := []models.Block{
		{BlockID: 1, Name: "z", Sort: 5, Spaces: []models.BlockSpace{{Space: "footer", Sort: 5}}},
		{BlockID: 2, Name: "a", Sort: 1, Spaces: []models.BlockSpace{{Space: "admin-dashboard-widgets", Sort: 1}}},
		{BlockID: 3, Name: "b", Sort: 0, Spaces: []models.BlockSpace{{Space: "admin-dashboard-widgets", Sort: 0}}},
	}
	sortBlocksByPrimarySpace(rows, false)
	if blocks.PrimarySpace(rows[0]) != "admin-dashboard-widgets" || rows[0].BlockID != 3 {
		t.Fatalf("got %+v", rows)
	}
	if rows[1].BlockID != 2 || rows[2].BlockID != 1 {
		t.Fatalf("got %+v", rows)
	}
	pos := blockSortPositionsByPrimarySpace(rows)
	if !pos[3].canMoveDown || pos[3].canMoveUp {
		t.Fatalf("block 3 position %+v", pos[3])
	}
	if !pos[2].canMoveUp || pos[2].canMoveDown {
		t.Fatalf("block 2 position %+v", pos[2])
	}
	if pos[1].canMoveUp || pos[1].canMoveDown {
		t.Fatalf("block 1 position %+v", pos[1])
	}
}
