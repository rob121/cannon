package blocks_test

import (
	"context"
	"testing"

	"github.com/rob121/cannon/internal/blocks"
	"github.com/rob121/cannon/internal/config"
	"github.com/rob121/cannon/internal/database"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/sites"
)

func TestEnsureAdminDashboardBlocksSeedsOnce(t *testing.T) {
	site := &config.SiteConfig{
		ID:       "admin-dash-seed",
		Name:     "Seed",
		Database: config.DatabaseConfig{Type: "sqlite", DSN: t.TempDir() + "/site.db"},
	}
	if err := database.Migrate(site); err != nil {
		t.Fatal(err)
	}
	ctx := sites.WithContext(context.Background(), site)
	if err := blocks.EnsureAdminDashboardBlocks(ctx); err != nil {
		t.Fatal(err)
	}
	if err := blocks.EnsureAdminDashboardBlocks(ctx); err != nil {
		t.Fatal(err)
	}
	db, err := database.Get(site.ID)
	if err != nil {
		t.Fatal(err)
	}
	var count int64
	if err := db.Model(&models.Block{}).Where("type LIKE ?", "admin-%").Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 6 {
		t.Fatalf("expected 6 admin dashboard blocks, got %d", count)
	}
	var spaces []string
	if err := db.Model(&models.BlockSpace{}).
		Joins("JOIN blocks ON blocks.block_id = block_spaces.block_id").
		Where("blocks.type LIKE ?", "admin-%").
		Order("block_spaces.space asc, block_spaces.sort asc").
		Pluck("block_spaces.space", &spaces).Error; err != nil {
		t.Fatal(err)
	}
	want := []string{
		"admin-dashboard-main", "admin-dashboard-main",
		"admin-dashboard-top",
		"admin-dashboard-widgets", "admin-dashboard-widgets", "admin-dashboard-widgets",
	}
	if len(spaces) != len(want) {
		t.Fatalf("spaces=%v want %v", spaces, want)
	}
	for i := range want {
		if spaces[i] != want[i] {
			t.Fatalf("spaces[%d]=%q want %q (all=%v)", i, spaces[i], want[i], spaces)
		}
	}
}

func TestIsAdminDashboardBlockType(t *testing.T) {
	if !blocks.IsAdminDashboardBlockType(models.BlockTypeAdminReviewQueue) {
		t.Fatal("expected review queue true")
	}
	if blocks.IsAdminDashboardBlockType(models.BlockTypeHTML) {
		t.Fatal("expected html false")
	}
}
