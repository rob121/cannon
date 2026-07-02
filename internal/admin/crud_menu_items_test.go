package admin

import (
	"testing"

	"github.com/rob121/cannon/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestMenuItemReorder(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.MenuItem{}); err != nil {
		t.Fatal(err)
	}
	parentID := uint(10)
	rows := []models.MenuItem{
		{MenuItemID: 1, MenuID: 1, Name: "First", Sort: 0},
		{MenuItemID: 2, MenuID: 1, Name: "Second", Sort: 1},
		{MenuItemID: 3, MenuID: 1, Name: "Child A", ParentID: &parentID, Sort: 0},
		{MenuItemID: 4, MenuID: 1, Name: "Child B", ParentID: &parentID, Sort: 1},
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatal(err)
	}
	if err := menuItemReorder(db, 2, -1); err != nil {
		t.Fatal(err)
	}
	var first, second models.MenuItem
	if err := db.First(&first, 1).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.First(&second, 2).Error; err != nil {
		t.Fatal(err)
	}
	if first.Sort != 1 || second.Sort != 0 {
		t.Fatalf("expected top-level swap, got first=%d second=%d", first.Sort, second.Sort)
	}

	if err := menuItemReorder(db, 4, -1); err != nil {
		t.Fatal(err)
	}
	var childA, childB models.MenuItem
	if err := db.First(&childA, 3).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.First(&childB, 4).Error; err != nil {
		t.Fatal(err)
	}
	if childA.Sort != 1 || childB.Sort != 0 {
		t.Fatalf("expected sibling swap, got childA=%d childB=%d", childA.Sort, childB.Sort)
	}
}

func TestMenuItemSortPositions(t *testing.T) {
	parentID := uint(10)
	rows := []models.MenuItem{
		{MenuItemID: 1, MenuID: 1, Sort: 0},
		{MenuItemID: 2, MenuID: 1, Sort: 1},
		{MenuItemID: 3, MenuID: 1, ParentID: &parentID, Sort: 0},
	}
	pos := menuItemSortPositions(rows)
	if pos[1].canMoveUp || !pos[1].canMoveDown {
		t.Fatalf("first top-level item: %+v", pos[1])
	}
	if !pos[2].canMoveUp || pos[2].canMoveDown {
		t.Fatalf("last top-level item: %+v", pos[2])
	}
	if pos[3].canMoveUp || pos[3].canMoveDown {
		t.Fatalf("only child item: %+v", pos[3])
	}
}
