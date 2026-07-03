package admin

import (
	"testing"

	"github.com/rob121/cannon/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestItemReorderWithinCategory(t *testing.T) {
	db := testItemDB(t)
	catID := uint(7)
	rows := []models.Item{
		{ItemID: 1, Title: "A", Slug: "a", Sort: 0},
		{ItemID: 2, Title: "B", Slug: "b", Sort: 1},
		{ItemID: 3, Title: "C1", Slug: "c1", CategoryID: &catID, Sort: 0},
		{ItemID: 4, Title: "C2", Slug: "c2", CategoryID: &catID, Sort: 1},
	}
	for _, row := range rows {
		if err := db.Create(&row).Error; err != nil {
			t.Fatal(err)
		}
	}

	if err := itemReorder(db, 2, -1); err != nil {
		t.Fatal(err)
	}
	var uncategorized []models.Item
	if err := db.Where("category_id IS NULL OR category_id = 0").Order("sort asc, item_id asc").Find(&uncategorized).Error; err != nil {
		t.Fatal(err)
	}
	if len(uncategorized) != 2 || uncategorized[0].Slug != "b" || uncategorized[1].Slug != "a" {
		t.Fatalf("uncategorized = %#v", uncategorized)
	}

	if err := itemReorder(db, 4, -1); err != nil {
		t.Fatal(err)
	}
	var categorized []models.Item
	if err := db.Where("category_id = ?", catID).Order("sort asc, item_id asc").Find(&categorized).Error; err != nil {
		t.Fatal(err)
	}
	if len(categorized) != 2 || categorized[0].Slug != "c2" || categorized[1].Slug != "c1" {
		t.Fatalf("categorized = %#v", categorized)
	}
}

func TestItemSortPositions(t *testing.T) {
	catID := uint(3)
	pos := itemSortPositions([]models.Item{
		{ItemID: 1, Sort: 0},
		{ItemID: 2, Sort: 1},
		{ItemID: 3, CategoryID: &catID, Sort: 0},
	})
	if pos[3].canMoveUp || pos[3].canMoveDown {
		t.Fatalf("single categorized item = %#v", pos[3])
	}
	if !pos[2].canMoveUp || pos[2].canMoveDown {
		t.Fatalf("last uncategorized item = %#v", pos[2])
	}
}

func TestItemFeaturedReorder(t *testing.T) {
	db := testItemDB(t)
	rows := []models.Item{
		{ItemID: 1, Title: "A", Slug: "a", Featured: true, FeaturedSort: 0},
		{ItemID: 2, Title: "B", Slug: "b", Featured: true, FeaturedSort: 1},
		{ItemID: 3, Title: "C", Slug: "c", Featured: false, FeaturedSort: 0, Sort: 0},
	}
	for _, row := range rows {
		if err := db.Create(&row).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := itemFeaturedReorder(db, 2, -1); err != nil {
		t.Fatal(err)
	}
	var featured []models.Item
	if err := db.Where("featured = ?", true).Order("featured_sort asc, item_id asc").Find(&featured).Error; err != nil {
		t.Fatal(err)
	}
	if len(featured) != 2 || featured[0].Slug != "b" || featured[1].Slug != "a" {
		t.Fatalf("featured order = %#v", featured)
	}
}

func TestItemFeaturedSortPositions(t *testing.T) {
	pos := itemFeaturedSortPositions([]models.Item{
		{ItemID: 1, FeaturedSort: 0},
		{ItemID: 2, FeaturedSort: 1},
	})
	if !pos[2].canMoveUp || pos[2].canMoveDown {
		t.Fatalf("last featured item = %#v", pos[2])
	}
}

func testItemDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.Item{}); err != nil {
		t.Fatal(err)
	}
	return db
}
