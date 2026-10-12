package content_test

import (
	"context"
	"testing"

	"github.com/rob121/cannon/internal/config"
	"github.com/rob121/cannon/internal/content"
	"github.com/rob121/cannon/internal/database"
	"github.com/rob121/cannon/internal/groups"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/sites"
	"gorm.io/gorm"
)

func TestCategoryShowTitleInheritsFromParent(t *testing.T) {
	ctx, db := categorySettingsTestContext(t)

	parent := models.Category{Name: "Parent", Slug: "parent", InheritSettings: false, Status: models.StatusActive}
	if err := db.Create(&parent).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&parent).Update("show_title", false).Error; err != nil {
		t.Fatal(err)
	}
	child := models.Category{Name: "Child", Slug: "child", ParentID: &parent.CategoryID, ShowTitle: true, InheritSettings: true, Status: models.StatusActive}
	if err := db.Create(&child).Error; err != nil {
		t.Fatal(err)
	}

	got, err := content.CategoryShowTitle(ctx, &child)
	if err != nil {
		t.Fatal(err)
	}
	if got {
		t.Fatalf("ShowTitle = %v, want false from parent", got)
	}
}

func TestCategoryShowTitleUsesOwnWhenNotInheriting(t *testing.T) {
	ctx, db := categorySettingsTestContext(t)

	parentID := uint(99)
	parent := models.Category{CategoryID: parentID, Name: "Parent", Slug: "parent-own", ShowTitle: false, InheritSettings: false, Status: models.StatusActive}
	if err := db.Create(&parent).Error; err != nil {
		t.Fatal(err)
	}
	child := models.Category{Name: "Child", Slug: "child-own", ParentID: &parentID, ShowTitle: true, InheritSettings: false, Status: models.StatusActive}
	if err := db.Create(&child).Error; err != nil {
		t.Fatal(err)
	}

	got, err := content.CategoryShowTitle(ctx, &child)
	if err != nil {
		t.Fatal(err)
	}
	if !got {
		t.Fatalf("ShowTitle = %v, want true from child override", got)
	}
}

func TestCategoryShowDescriptionInheritsFromParent(t *testing.T) {
	ctx, db := categorySettingsTestContext(t)

	parent := models.Category{Name: "Parent", Slug: "parent-desc", InheritSettings: false, Status: models.StatusActive}
	if err := db.Create(&parent).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&parent).Update("show_description", false).Error; err != nil {
		t.Fatal(err)
	}
	child := models.Category{Name: "Child", Slug: "child-desc", ParentID: &parent.CategoryID, ShowDescription: true, InheritSettings: true, Status: models.StatusActive}
	if err := db.Create(&child).Error; err != nil {
		t.Fatal(err)
	}

	got, err := content.CategoryShowDescription(ctx, &child)
	if err != nil {
		t.Fatal(err)
	}
	if got {
		t.Fatalf("ShowDescription = %v, want false from parent", got)
	}
}

func TestCategoryListingSettingsInheritsFromParent(t *testing.T) {
	ctx, db := categorySettingsTestContext(t)

	parent := models.Category{Name: "Parent", Slug: "parent-list", InheritSettings: false, ListColumns: 2, ListPagination: true, ListPageSize: 12, Status: models.StatusActive}
	if err := db.Create(&parent).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&parent).Updates(map[string]any{
		"list_pagination":  false,
		"list_columns":     2,
		"list_page_size":   12,
		"show_read_more":   true,
		"leading_articles": 3,
	}).Error; err != nil {
		t.Fatal(err)
	}
	child := models.Category{Name: "Child", Slug: "child-list", ParentID: &parent.CategoryID, ListColumns: 4, ListPagination: true, ListPageSize: 50, InheritSettings: true, Status: models.StatusActive}
	if err := db.Create(&child).Error; err != nil {
		t.Fatal(err)
	}

	got, err := content.ResolveCategoryListingSettings(ctx, &child)
	if err != nil {
		t.Fatal(err)
	}
	if got.Columns != 2 || got.Pagination || got.PageSize != 12 || !got.ShowReadMore || got.LeadingItems != 3 {
		t.Fatalf("listing settings = %+v, want parent values 2/false/12/true/3", got)
	}
}

func TestCategoryListingSettingsUsesOwnReadMoreWhenNotInheriting(t *testing.T) {
	ctx, db := categorySettingsTestContext(t)

	parent := models.Category{Name: "Parent", Slug: "parent-read-more", InheritSettings: false, ShowReadMore: true, Status: models.StatusActive}
	if err := db.Create(&parent).Error; err != nil {
		t.Fatal(err)
	}
	child := models.Category{Name: "Child", Slug: "child-read-more", ParentID: &parent.CategoryID, ShowReadMore: false, InheritSettings: false, Status: models.StatusActive}
	if err := db.Create(&child).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&child).Updates(map[string]any{
		"inherit_settings": false,
		"show_read_more":   false,
	}).Error; err != nil {
		t.Fatal(err)
	}
	child.InheritSettings = false
	child.ShowReadMore = false

	got, err := content.ResolveCategoryListingSettings(ctx, &child)
	if err != nil {
		t.Fatal(err)
	}
	if got.ShowReadMore {
		t.Fatalf("ShowReadMore = %v, want child override false", got.ShowReadMore)
	}
}

func TestResolveItemDisplaySettingsUsesCategoryItemDisplaySetting(t *testing.T) {
	ctx, db := categorySettingsTestContext(t)

	cat := models.Category{
		Name:              "News",
		Slug:              "news-item-display",
		ItemTitleDisplay:  content.DisplayYes,
		ItemAuthorDisplay: content.DisplayNo,
		ItemDateDisplay:   content.DisplayYes,
		InheritSettings:   false,
		Status:            models.StatusActive,
	}
	if err := db.Create(&cat).Error; err != nil {
		t.Fatal(err)
	}
	item := models.Item{Title: "Story", Slug: "story", Category: &cat}

	got, err := content.ResolveItemDisplaySettings(ctx, &item, content.Settings{ShowTitle: false, ShowAuthor: true, ShowPublishedDate: false})
	if err != nil {
		t.Fatal(err)
	}
	if !got.ShowTitle || got.ShowAuthor || !got.ShowPublishedDate {
		t.Fatalf("item display = title:%v author:%v date:%v, want true/false/true from category", got.ShowTitle, got.ShowAuthor, got.ShowPublishedDate)
	}
}

func TestResolveItemDisplaySettingsInheritsCategoryItemDisplaySetting(t *testing.T) {
	ctx, db := categorySettingsTestContext(t)

	parent := models.Category{Name: "Parent", Slug: "parent-item-display", ItemTitleDisplay: content.DisplayNo, ItemAuthorDisplay: content.DisplayNo, ItemDateDisplay: content.DisplayNo, InheritSettings: false, Status: models.StatusActive}
	if err := db.Create(&parent).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&parent).Updates(map[string]any{
		"item_title_display":  content.DisplayNo,
		"item_author_display": content.DisplayNo,
		"item_date_display":   content.DisplayNo,
		"inherit_settings":    false,
	}).Error; err != nil {
		t.Fatal(err)
	}
	child := models.Category{Name: "Child", Slug: "child-item-display", ParentID: &parent.CategoryID, ItemTitleDisplay: content.DisplayYes, ItemAuthorDisplay: content.DisplayYes, ItemDateDisplay: content.DisplayYes, InheritSettings: true, Status: models.StatusActive}
	if err := db.Create(&child).Error; err != nil {
		t.Fatal(err)
	}
	item := models.Item{Title: "Story", Slug: "story", Category: &child}

	got, err := content.ResolveItemDisplaySettings(ctx, &item, content.Settings{ShowTitle: true, ShowAuthor: true, ShowPublishedDate: true})
	if err != nil {
		t.Fatal(err)
	}
	if got.ShowTitle || got.ShowAuthor || got.ShowPublishedDate {
		t.Fatalf("item display = title:%v author:%v date:%v, want inherited parent false values", got.ShowTitle, got.ShowAuthor, got.ShowPublishedDate)
	}
}

func TestFeaturedCategorySettingsRoundTrip(t *testing.T) {
	ctx, _ := categorySettingsTestContext(t)

	row := content.FeaturedCategoryDefaults(ctx)
	row.Name = "Featured Picks"
	row.Description = "<p>Curated work</p>"
	row.Image = "/media/featured.jpg"
	row.Template = "custom/featured.html"
	row.InheritSettings = false
	row.ShowTitle = false
	row.ShowDescription = false
	row.ShowReadMore = true
	row.ItemTitleDisplay = content.DisplayNo
	row.ItemAuthorDisplay = content.DisplayYes
	row.ItemDateDisplay = content.DisplayNo
	row.LeadingArticles = 3
	row.ListColumns = 2
	row.ListPagination = false
	row.ListPageSize = 12
	if err := content.SaveFeaturedCategorySettings(ctx, row); err != nil {
		t.Fatal(err)
	}

	got, err := content.FeaturedCategorySettings(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "Featured Picks" || got.Slug != "featured" || got.Description != "<p>Curated work</p>" || got.Image != "/media/featured.jpg" || got.Template != "custom/featured.html" {
		t.Fatalf("featured category identity = %+v, want saved display fields with virtual slug", got)
	}
	if got.InheritSettings || got.ShowTitle || got.ShowDescription || !got.ShowReadMore || got.ItemTitleDisplay != content.DisplayNo || got.ItemAuthorDisplay != content.DisplayYes || got.ItemDateDisplay != content.DisplayNo || got.LeadingArticles != 3 || got.ListColumns != 2 || got.ListPagination || got.ListPageSize != 12 {
		t.Fatalf("featured category settings = %+v, want saved custom settings", got)
	}

	resolved := content.ResolveFeaturedListingSettings(ctx, got)
	if resolved.Columns != 2 || resolved.Pagination || resolved.PageSize != 12 || !resolved.ShowReadMore || resolved.LeadingItems != 3 {
		t.Fatalf("resolved featured settings = %+v, want custom values", resolved)
	}
	if content.FeaturedShowTitle(ctx, got) || content.FeaturedShowDescription(ctx, got) {
		t.Fatalf("featured visibility should use custom hidden title/description")
	}
	itemDisplay := content.ResolveFeaturedItemDisplaySettings(ctx, got, content.Settings{ShowTitle: true, ShowAuthor: false, ShowPublishedDate: true})
	if itemDisplay.ShowTitle || !itemDisplay.ShowAuthor || itemDisplay.ShowPublishedDate {
		t.Fatalf("featured item display = %+v, want false/true/false", itemDisplay)
	}
}

func TestFeaturedCategorySettingsInheritUsesOwnSettingsWithoutParent(t *testing.T) {
	ctx, _ := categorySettingsTestContext(t)

	row := content.FeaturedCategoryDefaults(ctx)
	row.InheritSettings = true
	row.ShowTitle = false
	row.ShowDescription = false
	row.ShowReadMore = true
	row.LeadingArticles = 3
	row.ListColumns = 2
	row.ListPagination = false
	row.ListPageSize = 12

	resolved := content.ResolveFeaturedListingSettings(ctx, row)
	if resolved.Columns != 2 || resolved.Pagination || resolved.PageSize != 12 || !resolved.ShowReadMore || resolved.LeadingItems != 3 {
		t.Fatalf("inherited featured settings = %+v, want own top-level settings", resolved)
	}
	if content.FeaturedShowTitle(ctx, row) || content.FeaturedShowDescription(ctx, row) {
		t.Fatalf("inherited featured title/description should use own top-level settings")
	}
}

func TestCategoryItemColumnClass(t *testing.T) {
	if cms := content.CategoryItemColumnClass(1); cms != "col-12" {
		t.Fatalf("1 column: %q", cms)
	}
	if cms := content.CategoryItemColumnClass(4); cms != "col-md-6 col-lg-3" {
		t.Fatalf("4 columns: %q", cms)
	}
}

func TestNormalizeCategoryLeadingArticles(t *testing.T) {
	if got := content.NormalizeCategoryLeadingArticles(-1); got != 0 {
		t.Fatalf("negative leading articles = %d, want 0", got)
	}
	if got := content.NormalizeCategoryLeadingArticles(12); got != 12 {
		t.Fatalf("leading articles = %d, want 12", got)
	}
	if got := content.NormalizeCategoryLeadingArticles(26); got != 25 {
		t.Fatalf("over max leading articles = %d, want 25", got)
	}
}

func TestSplitCategoryLeadingItems(t *testing.T) {
	items := []models.Item{
		{ItemID: 1}, {ItemID: 2}, {ItemID: 3}, {ItemID: 4}, {ItemID: 5}, {ItemID: 6},
		{ItemID: 7}, {ItemID: 8}, {ItemID: 9}, {ItemID: 10}, {ItemID: 11}, {ItemID: 12},
		{ItemID: 13},
	}
	settings := content.CategoryListingSettings{LeadingItems: 8, Pagination: true, PageSize: 6}

	leading, remaining := content.SplitCategoryLeadingItems(items[:6], settings, 1)
	if len(leading) != 6 || len(remaining) != 0 {
		t.Fatalf("page 1 split: leading=%d remaining=%d, want 6/0", len(leading), len(remaining))
	}

	leading, remaining = content.SplitCategoryLeadingItems(items[6:12], settings, 2)
	if len(leading) != 2 || len(remaining) != 4 || leading[0].ItemID != 7 || remaining[0].ItemID != 9 {
		t.Fatalf("page 2 split: leading=%v remaining=%v, want items 7-8 leading and item 9 first remaining", leading, remaining)
	}

	leading, remaining = content.SplitCategoryLeadingItems(items[12:], settings, 3)
	if len(leading) != 0 || len(remaining) != 1 || remaining[0].ItemID != 13 {
		t.Fatalf("page 3 split: leading=%v remaining=%v, want no leading and item 13 remaining", leading, remaining)
	}
}

func TestCategoryItemListClass(t *testing.T) {
	if cls := content.CategoryItemListClass(1); cls != "site-item-list" {
		t.Fatalf("1 column: %q", cls)
	}
	if cls := content.CategoryItemListClass(2); cls != "site-item-list site-item-list--cols-2" {
		t.Fatalf("2 columns: %q", cls)
	}
	if cls := content.CategoryItemListClass(4); cls != "site-item-list site-item-list--cols-4" {
		t.Fatalf("4 columns: %q", cls)
	}
}

func categorySettingsTestContext(t *testing.T) (context.Context, *gorm.DB) {
	t.Helper()
	path := t.TempDir() + "/category-settings.sqlite"
	site := &config.SiteConfig{
		ID: t.Name(),
		Database: config.DatabaseConfig{
			Type: "sqlite",
			DSN:  path,
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
