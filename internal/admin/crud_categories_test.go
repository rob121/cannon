package admin

import (
	"testing"

	"github.com/rob121/cannon/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestCategoryReorderWithinParent(t *testing.T) {
	db := testCategoryDB(t)
	parentID := uint(10)
	rows := []models.Category{
		{CategoryID: 1, Name: "Root A", Slug: "root-a", Sort: 0},
		{CategoryID: 2, Name: "Root B", Slug: "root-b", Sort: 1},
		{CategoryID: 3, Name: "Child A", Slug: "child-a", ParentID: &parentID, Sort: 0},
		{CategoryID: 4, Name: "Child B", Slug: "child-b", ParentID: &parentID, Sort: 1},
	}
	for _, row := range rows {
		if err := db.Create(&row).Error; err != nil {
			t.Fatal(err)
		}
	}

	if err := categoryReorder(db, 2, -1); err != nil {
		t.Fatal(err)
	}
	var roots []models.Category
	if err := db.Where("parent_id IS NULL OR parent_id = 0").Order("sort asc, category_id asc").Find(&roots).Error; err != nil {
		t.Fatal(err)
	}
	if len(roots) != 2 || roots[0].Slug != "root-b" || roots[1].Slug != "root-a" {
		t.Fatalf("roots = %#v", roots)
	}

	if err := categoryReorder(db, 4, -1); err != nil {
		t.Fatal(err)
	}
	var children []models.Category
	if err := db.Where("parent_id = ?", parentID).Order("sort asc, category_id asc").Find(&children).Error; err != nil {
		t.Fatal(err)
	}
	if len(children) != 2 || children[0].Slug != "child-b" || children[1].Slug != "child-a" {
		t.Fatalf("children = %#v", children)
	}
}

func TestCategorySortPositions(t *testing.T) {
	parentID := uint(5)
	pos := categorySortPositions([]models.Category{
		{CategoryID: 1, Sort: 0},
		{CategoryID: 2, Sort: 1},
		{CategoryID: 3, ParentID: &parentID, Sort: 0},
	})
	if pos[1].canMoveUp || !pos[1].canMoveDown {
		t.Fatalf("root middle position = %#v", pos[1])
	}
	if !pos[2].canMoveUp || pos[2].canMoveDown {
		t.Fatalf("only child position = %#v", pos[2])
	}
}

func testCategoryDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.Category{}); err != nil {
		t.Fatal(err)
	}
	return db
}
