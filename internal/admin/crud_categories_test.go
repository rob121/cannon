package admin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/rob121/cannon/internal/config"
	"github.com/rob121/cannon/internal/database"
	"github.com/rob121/cannon/internal/groups"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/sites"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestCategoryFormSavesRelatedItemDisplay(t *testing.T) {
	site := &config.SiteConfig{ID: t.Name(), Database: config.DatabaseConfig{Type: "sqlite", DSN: t.TempDir() + "/site.db"}}
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
	publicID, err := groups.PublicGroupID(db)
	if err != nil {
		t.Fatal(err)
	}
	ctx := sites.WithContext(context.Background(), site)
	var id uint
	for _, mode := range []string{"no", "yes", "inherit"} {
		values := url.Values{"name": {"News"}, "slug": {"news"}, "status": {"active"}, "item_related_display": {mode}, "group_ids": {strconv.FormatUint(uint64(publicID), 10)}}
		req := httptest.NewRequest(http.MethodPost, "/admin/categories/new", strings.NewReader(values.Encode())).WithContext(ctx)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()
		(&Handler{}).categoryForm(rec, req, id)
		if rec.Code != http.StatusSeeOther {
			t.Fatalf("save %q: status %d, body %s", mode, rec.Code, rec.Body.String())
		}
		var cat models.Category
		if err := db.Where("slug = ?", "news").First(&cat).Error; err != nil {
			t.Fatal(err)
		}
		if cat.ItemRelatedDisplay != mode {
			t.Fatalf("saved mode = %q, want %q", cat.ItemRelatedDisplay, mode)
		}
		id = cat.CategoryID
	}
}

func TestCategoryFormSavesCategoryNameDisplay(t *testing.T) {
	site := &config.SiteConfig{ID: t.Name(), Database: config.DatabaseConfig{Type: "sqlite", DSN: t.TempDir() + "/site.db"}}
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
	publicID, err := groups.PublicGroupID(db)
	if err != nil {
		t.Fatal(err)
	}
	ctx := sites.WithContext(context.Background(), site)
	var id uint
	for _, mode := range []string{"no", "yes", "inherit"} {
		values := url.Values{"name": {"News"}, "slug": {"news"}, "status": {"active"}, "item_category_display": {mode}, "group_ids": {strconv.FormatUint(uint64(publicID), 10)}}
		req := httptest.NewRequest(http.MethodPost, "/admin/categories/new", strings.NewReader(values.Encode())).WithContext(ctx)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()
		(&Handler{}).categoryForm(rec, req, id)
		if rec.Code != http.StatusSeeOther {
			t.Fatalf("save %q: status %d, body %s", mode, rec.Code, rec.Body.String())
		}
		var cat models.Category
		if err := db.Where("slug = ?", "news").First(&cat).Error; err != nil {
			t.Fatal(err)
		}
		if cat.ItemCategoryDisplay != mode {
			t.Fatalf("saved mode = %q, want %q", cat.ItemCategoryDisplay, mode)
		}
		id = cat.CategoryID
	}
}

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
