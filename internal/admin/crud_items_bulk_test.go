package admin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/rob121/cannon/internal/config"
	"github.com/rob121/cannon/internal/database"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/sites"
)

func TestItemBulkAppliesToSelectedRows(t *testing.T) {
	site := &config.SiteConfig{ID: t.Name(), Database: config.DatabaseConfig{
		Type: "sqlite", DSN: t.TempDir() + "/items.db",
	}}
	db, err := database.Connect(site)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.Item{}); err != nil {
		t.Fatal(err)
	}
	for _, item := range []models.Item{
		{ItemID: 8, Title: "Unselected", Slug: "unselected", Status: models.ItemStatusDraft},
		{ItemID: 9, Title: "Selected second row", Slug: "second", Status: models.ItemStatusDraft},
		{ItemID: 10, Title: "Selected third row", Slug: "third", Status: models.ItemStatusDraft},
	} {
		if err := db.Create(&item).Error; err != nil {
			t.Fatal(err)
		}
	}
	values := url.Values{"bulk_action": {"publish"}, "item_ids": {"9", "10"}}
	req := httptest.NewRequest(http.MethodPost, "/admin/items/bulk?dir=asc&page_size=25&sort=sort", strings.NewReader(values.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = req.WithContext(sites.WithContext(context.Background(), site))
	rec := httptest.NewRecorder()
	(&Handler{}).itemBulk(rec, req)
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/admin/items?dir=asc&page_size=25&sort=sort" {
		t.Fatalf("response = %d, location %q, body %s", rec.Code, rec.Header().Get("Location"), rec.Body.String())
	}
	for id, want := range map[uint]models.ItemStatus{8: models.ItemStatusDraft, 9: models.ItemStatusPublished, 10: models.ItemStatusPublished} {
		var item models.Item
		if err := db.First(&item, id).Error; err != nil {
			t.Fatal(err)
		}
		if item.Status != want {
			t.Errorf("item %d status = %q, want %q", id, item.Status, want)
		}
	}
}
