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
)

func TestCloneItemCopiesContentAsDraft(t *testing.T) {
	path := t.TempDir() + "/clone-item.sqlite"
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
	ctx := content.WithLocale(sites.WithContext(context.Background(), site), "en-US")

	src := models.Item{
		Title:           "Original Post",
		Slug:            "original-post",
		Locale:          "en-US",
		Status:          models.ItemStatusPublished,
		Featured:        true,
		FeaturedSort:    3,
		Intro:           "Intro text",
		Body:            "Body text",
		MetaTitle:       "Meta",
		MetaDescription: "Desc",
		MetaKeywords:    "kw",
		Image:           "/media/hero.jpg",
		GalleryJSON:     `[{"path":"/a.jpg"}]`,
	}
	if err := db.Create(&src).Error; err != nil {
		t.Fatal(err)
	}
	tag := models.Tag{Name: "news", Slug: "news"}
	if err := db.Create(&tag).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&src).Association("Tags").Append(&tag); err != nil {
		t.Fatal(err)
	}
	group := models.Group{Name: "members", Status: models.StatusActive}
	if err := db.Create(&group).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&src).Association("Groups").Append(&group); err != nil {
		t.Fatal(err)
	}
	fieldGroup := models.ContentFieldGroup{Name: "details"}
	if err := db.Create(&fieldGroup).Error; err != nil {
		t.Fatal(err)
	}
	field := models.ContentField{
		FieldGroupID: fieldGroup.FieldGroupID,
		Name:         "color",
		Label:        "Color",
		Type:         "text",
		Status:       models.StatusActive,
	}
	if err := db.Create(&field).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.ItemFieldValue{ItemID: src.ItemID, FieldID: field.FieldID, Value: "blue"}).Error; err != nil {
		t.Fatal(err)
	}

	clone, err := content.CloneItem(ctx, db, src.ItemID)
	if err != nil {
		t.Fatal(err)
	}
	if clone.ItemID == 0 || clone.ItemID == src.ItemID {
		t.Fatalf("clone id = %d, source = %d", clone.ItemID, src.ItemID)
	}
	if clone.Title != "Copy of Original Post" {
		t.Fatalf("title = %q", clone.Title)
	}
	if clone.Slug == src.Slug {
		t.Fatalf("slug should be unique, got %q", clone.Slug)
	}
	if clone.Status != models.ItemStatusDraft {
		t.Fatalf("status = %q, want draft", clone.Status)
	}
	if clone.Featured || clone.FeaturedSort != 0 {
		t.Fatalf("featured = %v sort = %d", clone.Featured, clone.FeaturedSort)
	}
	if clone.Intro != src.Intro || clone.Body != src.Body || clone.Image != src.Image {
		t.Fatal("content fields not copied")
	}
	if clone.CanonicalURL != "" || clone.PreviewToken != "" {
		t.Fatal("preview/canonical should be cleared on clone")
	}

	var cloneTags []models.Tag
	db.Model(clone).Association("Tags").Find(&cloneTags)
	if len(cloneTags) != 1 || cloneTags[0].TagID != tag.TagID {
		t.Fatalf("tags = %#v", cloneTags)
	}
	var cloneGroups []models.Group
	db.Model(clone).Association("Groups").Find(&cloneGroups)
	if len(cloneGroups) != 1 || cloneGroups[0].GroupID != group.GroupID {
		t.Fatalf("groups = %#v", cloneGroups)
	}
	var fieldValue models.ItemFieldValue
	if err := db.Where("item_id = ? AND field_id = ?", clone.ItemID, field.FieldID).First(&fieldValue).Error; err != nil {
		t.Fatal(err)
	}
	if fieldValue.Value != "blue" {
		t.Fatalf("field value = %q", fieldValue.Value)
	}
}

func TestCloneItemTitlePreservesExistingCopyPrefix(t *testing.T) {
	path := t.TempDir() + "/clone-copy-title.sqlite"
	site := &config.SiteConfig{
		ID: t.Name() + "-title",
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
	ctx := content.WithLocale(sites.WithContext(context.Background(), site), "en-US")
	src := models.Item{Title: "Copy of Story", Slug: "copy-of-story", Locale: "en-US", Status: models.ItemStatusPublished}
	if err := db.Create(&src).Error; err != nil {
		t.Fatal(err)
	}
	clone, err := content.CloneItem(ctx, db, src.ItemID)
	if err != nil {
		t.Fatal(err)
	}
	if clone.Title != "Copy of Story" {
		t.Fatalf("title = %q, want unchanged copy prefix", clone.Title)
	}
}
