package blocks_test

import (
	"context"
	"testing"

	"github.com/rob121/cannon/internal/blocks"
	"github.com/rob121/cannon/internal/config"
	"github.com/rob121/cannon/internal/database"
	"github.com/rob121/cannon/internal/groups"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/sites"
	"gorm.io/gorm"
)

func testBlocksDB(t *testing.T) (context.Context, *gorm.DB) {
	t.Helper()
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
	if err := groups.EnsureDefaults(db); err != nil {
		t.Fatal(err)
	}
	return sites.WithContext(context.Background(), site), db
}

func TestRepairBlockSpacesBackfill(t *testing.T) {
	_, db := testBlocksDB(t)

	legacy := models.Block{
		Name:   "Legacy",
		Type:   models.BlockTypeHTML,
		Space:  "footer",
		Status: models.StatusActive,
		Sort:   3,
	}
	if err := db.Omit("Spaces").Create(&legacy).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Where("block_id = ?", legacy.BlockID).Delete(&models.BlockSpace{}).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.RepairBlockSpacesSchema(db); err != nil {
		t.Fatal(err)
	}
	var memberships []models.BlockSpace
	if err := db.Where("block_id = ?", legacy.BlockID).Find(&memberships).Error; err != nil {
		t.Fatal(err)
	}
	if len(memberships) != 1 {
		t.Fatalf("expected 1 membership, got %d", len(memberships))
	}
	if memberships[0].Space != "footer" || memberships[0].Sort != 3 {
		t.Fatalf("unexpected membership: %+v", memberships[0])
	}
}

func TestListForSpaceMultiMembership(t *testing.T) {
	ctx, db := testBlocksDB(t)
	publicID, err := groups.PublicGroupID(db)
	if err != nil {
		t.Fatal(err)
	}
	public := models.Group{GroupID: publicID}

	row := models.Block{
		Name:   "Shared",
		Type:   models.BlockTypeHTML,
		Status: models.StatusActive,
	}
	if err := db.Omit("Spaces").Create(&row).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&row).Association("Groups").Replace([]models.Group{public}); err != nil {
		t.Fatal(err)
	}
	if err := blocks.ReplaceBlockSpaces(db, row.BlockID, []string{"sidebar", "footer"}); err != nil {
		t.Fatal(err)
	}
	other := models.Block{Name: "Sidebar Only", Type: models.BlockTypeHTML, Status: models.StatusActive}
	if err := db.Omit("Spaces").Create(&other).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&other).Association("Groups").Replace([]models.Group{public}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.BlockSpace{BlockID: other.BlockID, Space: "sidebar", Sort: 0}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&models.BlockSpace{}).
		Where("block_id = ? AND space = ?", row.BlockID, "sidebar").
		Update("sort", 1).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&models.BlockSpace{}).
		Where("block_id = ? AND space = ?", row.BlockID, "footer").
		Update("sort", 0).Error; err != nil {
		t.Fatal(err)
	}

	sidebar, err := blocks.ListForSpace(ctx, db, "sidebar")
	if err != nil {
		t.Fatal(err)
	}
	if len(sidebar) != 2 {
		t.Fatalf("sidebar: got %d blocks", len(sidebar))
	}
	if sidebar[0].BlockID != other.BlockID || sidebar[1].BlockID != row.BlockID {
		t.Fatalf("sidebar order wrong: %+v", sidebar)
	}
	if sidebar[1].Sort != 1 {
		t.Fatalf("expected join sort overlay 1, got %d", sidebar[1].Sort)
	}

	footer, err := blocks.ListForSpace(ctx, db, "footer")
	if err != nil {
		t.Fatal(err)
	}
	if len(footer) != 1 || footer[0].BlockID != row.BlockID {
		t.Fatalf("footer: %+v", footer)
	}

	spaces, err := blocks.DistinctSpaces(db)
	if err != nil {
		t.Fatal(err)
	}
	if len(spaces) != 2 || spaces[0] != "footer" || spaces[1] != "sidebar" {
		t.Fatalf("distinct spaces: %v", spaces)
	}
}

func TestReplaceBlockSpacesRoundTrip(t *testing.T) {
	_, db := testBlocksDB(t)

	row := models.Block{Name: "RoundTrip", Type: models.BlockTypeHTML, Status: models.StatusActive}
	if err := db.Omit("Spaces").Create(&row).Error; err != nil {
		t.Fatal(err)
	}
	if err := blocks.ReplaceBlockSpaces(db, row.BlockID, []string{" a ", "b", "a"}); err != nil {
		t.Fatal(err)
	}
	var names []string
	if err := db.Model(&models.BlockSpace{}).Where("block_id = ?", row.BlockID).Order("space asc").Pluck("space", &names).Error; err != nil {
		t.Fatal(err)
	}
	if len(names) != 2 || names[0] != "a" || names[1] != "b" {
		t.Fatalf("spaces after save: %v", names)
	}
	var primary string
	if err := db.Model(&models.Block{}).Select("space").Where("block_id = ?", row.BlockID).Scan(&primary).Error; err != nil {
		t.Fatal(err)
	}
	if primary != "a" {
		t.Fatalf("primary space: %q", primary)
	}

	if err := blocks.ReplaceBlockSpaces(db, row.BlockID, []string{"b"}); err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&models.BlockSpace{}).Where("block_id = ?", row.BlockID).Pluck("space", &names).Error; err != nil {
		t.Fatal(err)
	}
	if len(names) != 1 || names[0] != "b" {
		t.Fatalf("after remove: %v", names)
	}
	if err := blocks.ReplaceBlockSpaces(db, row.BlockID, nil); err != blocks.ErrNoSpaces {
		t.Fatalf("expected ErrNoSpaces, got %v", err)
	}
}
