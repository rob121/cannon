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

func TestSearchIndexFindsCustomField(t *testing.T) {
	ctx, db := searchTestContext(t)
	item := models.Item{
		Title:  "Indexed Widget",
		Slug:   "indexed-widget",
		Locale: "en-US",
		Status: models.ItemStatusPublished,
		Body:   "Body copy",
	}
	if err := db.Create(&item).Error; err != nil {
		t.Fatal(err)
	}
	group := models.ContentFieldGroup{Name: "specs"}
	if err := db.Create(&group).Error; err != nil {
		t.Fatal(err)
	}
	field := models.ContentField{
		FieldGroupID: group.FieldGroupID,
		Name:         "sku",
		Label:        "SKU",
		Type:         "text",
		Status:       models.StatusActive,
	}
	if err := db.Create(&field).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.ItemFieldValue{ItemID: item.ItemID, FieldID: field.FieldID, Value: "ALPHA-9000"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := content.UpsertSearchIndex(ctx, db, &item); err != nil {
		t.Fatal(err)
	}

	publicID, err := groups.PublicGroupID(db)
	if err != nil {
		t.Fatal(err)
	}
	if content.SearchIndexAvailable(db) {
		items, total, err := content.ListItems(ctx, []uint{publicID}, content.ListOptions{
			Query: "ALPHA-9000",
			Page:  1,
			Limit: 10,
		})
		if err != nil {
			t.Fatal(err)
		}
		if total != 1 || len(items) != 1 || items[0].ItemID != item.ItemID {
			t.Fatalf("FTS ListItems() = %d items, total %d, want item %d", len(items), total, item.ItemID)
		}
	}

	items, total, err := content.ListItems(ctx, []uint{publicID}, content.ListOptions{
		FieldFilters: map[string]string{"sku": "ALPHA"},
		Page:         1,
		Limit:        10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(items) != 1 {
		t.Fatalf("field filter got %d items, total %d", len(items), total)
	}
}

func searchTestContext(t *testing.T) (context.Context, *gorm.DB) {
	t.Helper()
	path := t.TempDir() + "/search.sqlite"
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
	return ctx, db
}
